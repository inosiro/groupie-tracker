package internal

import (
	"context"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// CACHING LAYER (TTL-based in-memory cache)
// ═══════════════════════════════════════════════════════════════════════════
// Reduces external API calls by caching the artists list with a configurable TTL.
// This is critical for performance: the artists list (52 artists) is fetched
// once and reused across all requests until expiry.
//
// Architecture:
//   - Initialized with: NewCache(apiClient, 2*time.Minute)
//   - Used by handlers via: cache.Artists(ctx)
//   - Thread-safe: uses sync.Mutex
//
// Data Flow:
//   Request → Handler → Cache.Artists() → [HIT: return cached] or [MISS: API call]

// CacheProvider is an interface for retrieving cached artist data
// This allows tests to inject mock implementations
type CacheProvider interface {
	// Artists returns the cached artists list or fetches from API if cache expired
	Artists(ctx context.Context) ([]Artist, error)
	// Locations returns the cached locations index or fetches from API
	Locations(ctx context.Context) (LocationIndex, error)
	// Dates returns the cached dates index or fetches from API
	Dates(ctx context.Context) (DateIndex, error)
}

// Cache stores the artists list with an expiry time
type Cache struct {
	mu         sync.Mutex     // Protects concurrent access to data/expiry
	data       []Artist       // Cached artists list
	expiry     time.Time      // Time when cache becomes invalid
	locData    *LocationIndex // Cached locations index
	locExpiry  time.Time      // Time when locations cache becomes invalid
	dateData   *DateIndex     // Cached dates index
	dateExpiry time.Time      // Time when dates cache becomes invalid
	client     *Client        // API client for fetching fresh data
	ttl        time.Duration  // Time-to-live duration (e.g., 2 minutes)
}

// NewCache creates a new cache with the given TTL
// Called from: main.go during server initialization
//
// Example: NewCache(apiClient, 2*time.Minute)
// This means the artists list will be cached for 2 minutes.
// After 2 minutes, the next request will fetch fresh data from the API.
func NewCache(c *Client, ttl time.Duration) *Cache {
	now := time.Now()
	return &Cache{
		client:     c,
		expiry:     now, // Start with expired cache to force first fetch
		locExpiry:  now,
		dateExpiry: now,
		ttl:        ttl,
	}
}

// Artists returns the cached artists list or fetches fresh data if cache expired
// Called by: ArtistsHandler, ArtistDetailsHandler (every request that needs artists)
//
// Logic:
//  1. Lock mutex to prevent concurrent modifications
//  2. Check if cache is still valid (now.Before(expiry) && data != nil)
//  3. If HIT: return cached data immediately
//  4. If MISS: call API client to fetch fresh data
//  5. Store new data in cache and update expiry time
//  6. Return data (or error)
func (c *Cache) Artists(ctx context.Context) ([]Artist, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if cache is still valid
	if time.Now().Before(c.expiry) && c.data != nil {
		return c.data, nil
	}

	// Cache miss: fetch fresh data from external API
	artists, err := c.client.Artists(ctx)
	if err != nil {
		return nil, err
	}

	// Store in cache and set new expiry
	c.data = artists
	c.expiry = time.Now().Add(c.ttl)

	// Pre-fetch auxiliary data (locations and dates) after initial loading of artists.
	// This populates the cache in the background to ensure fast filtering on subsequent requests.
	go func() {
		// Use a fresh context with a reasonable timeout for background tasks
		bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, _ = c.Locations(bgCtx)
		_, _ = c.Dates(bgCtx)
	}()

	return artists, nil
}

// Locations returns the cached locations index or fetches fresh data if cache expired
func (c *Cache) Locations(ctx context.Context) (LocationIndex, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if cache is still valid
	if time.Now().Before(c.locExpiry) && c.locData != nil {
		return *c.locData, nil
	}

	locs, err := c.client.AllLocations(ctx)
	if err != nil {
		return LocationIndex{}, err
	}

	c.locData = &locs
	c.locExpiry = time.Now().Add(c.ttl)
	return *c.locData, nil
}

// Dates returns the cached dates index or fetches fresh data if cache expired
func (c *Cache) Dates(ctx context.Context) (DateIndex, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Now().Before(c.dateExpiry) && c.dateData != nil {
		return *c.dateData, nil
	}

	dates, err := c.client.AllDates(ctx)
	if err != nil {
		return DateIndex{}, err
	}

	c.dateData = &dates
	c.dateExpiry = time.Now().Add(c.ttl)
	return *c.dateData, nil
}
