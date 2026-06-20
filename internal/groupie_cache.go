package internal

import (
	"context"
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"
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
	mu         sync.Mutex            // Protects concurrent access to data/expiry
	data       []Artist              // Cached artists list
	expiry     time.Time             // Time when cache becomes invalid
	locData    *LocationIndex        // Cached locations index
	locExpiry  time.Time             // Time when locations cache becomes invalid
	dateData   *DateIndex            // Cached dates index
	dateExpiry time.Time             // Time when dates cache becomes invalid
	client     *Client               // API client for fetching fresh data
	ttl        time.Duration         // Time-to-live duration (e.g., 5 minutes)
	coords     map[string][2]float64 // Cache of locations coordinates from static CSV
}

// NewCache creates a new cache with the given TTL and loads static coordinates from CSV
// Called from: main.go during server initialization
//
// Example: NewCache(apiClient, 5*time.Minute)
// This means the artists list will be cached for 5 minutes.
// After 5 minutes, the next request will fetch fresh data from the API.
func NewCache(c *Client, ttl time.Duration) *Cache {
	now := time.Now()
	cache := &Cache{
		client:     c,
		expiry:     now, // Start with expired cache to force first fetch
		locExpiry:  now,
		dateExpiry: now,
		ttl:        ttl,
		coords:     make(map[string][2]float64),
	}

	// Load the static location coordinates from CSV
	if err := cache.loadStaticCoords(); err != nil {
		log.Printf("Warning: Failed to load static coordinates from CSV: %v", err)
	}

	return cache
}

// loadStaticCoords reads the location coordinates from CSV and populates the cache.
// It supports fallback paths to ensure unit tests run successfully from both root and subdirectories.
func (c *Cache) loadStaticCoords() error {
	path := "./scripts/output.csv"
	csvFile, err := os.Open(path)
	if err != nil {
		// Fallback for tests running inside the internal/ directory
		path = "../scripts/output.csv"
		csvFile, err = os.Open(path)
		if err != nil {
			return err
		}
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	header, err := reader.Read()
	if err != nil {
		return err
	}
	cols := make(map[string]int)
	for i, name := range header {
		cols[name] = i
	}

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // Skip malformed rows
		}
		lng, err := strconv.ParseFloat(row[cols["lng"]], 64)
		if err != nil {
			continue
		}
		lat, err := strconv.ParseFloat(row[cols["lat"]], 64)
		if err != nil {
			continue
		}

		c.coords[row[cols["location_id"]]] = [2]float64{lng, lat}
	}
	return nil
}

// Lookup implements the CoordResolver interface by retrieving coordinates from the cached CSV data.
// It uses a mutex to prevent data races if coordinates are updated dynamically.
func (c *Cache) Lookup(locationKey string) (lng float64, lat float64, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	coords, exist := c.coords[locationKey]
	if !exist {
		return 0, 0, false
	}
	return coords[0], coords[1], true
}

// Store dynamically caches newly fetched coordinates.
func (c *Cache) Store(locationKey string, lng float64, lat float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.coords[locationKey] = [2]float64{lng, lat}
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
