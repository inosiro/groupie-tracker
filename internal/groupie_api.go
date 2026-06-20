package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ═══════════════════════════════════════════════════════════════════════════
// EXTERNAL API CLIENT
// ═══════════════════════════════════════════════════════════════════════════
// This module handles all communication with the Groupie Trackers external API.
// It fetches artist lists, concert relations, locations, and dates.

const BaseURL = "https://groupietrackers.herokuapp.com/api"
const Nominatim = "https://nominatim.openstreetmap.org"

// Client wraps an http.Client for Groupie Trackers API calls
type Client struct {
	http *http.Client
}

// NewClient creates a new API client with default HTTP client
func NewClient() *Client {
	return &Client{http: &http.Client{}}
}

// fetch is a helper method that performs JSON requests to the API
// It handles context cancellation, request creation, and response decoding.
//
// Parameters:
//   - ctx: context for request cancellation (timeout support)
//   - url: full URL to API endpoint
//   - target: pointer to struct where JSON response will be decoded
//
// Returns: error if request fails, network error, or JSON decode error
func (c *Client) fetch(ctx context.Context, url string, target any) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:152.0) Gecko/20100101 Firefox/152.0")
	req.Header.Set("Referrer", "localhost:8080")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("upstream api returned status %d", res.StatusCode)
	}

	return json.NewDecoder(res.Body).Decode(target)
}

// Artists fetches the complete list of artists from the external API
// Called by: Cache.Artists() (via TTL check)
// Returns: []Artist (empty slice on error) and error
//
// External API call: GET https://groupietrackers.herokuapp.com/api/artists
// Typical response: JSON array of 52 artists with id, name, image, members, dates, etc.
func (c *Client) Artists(ctx context.Context) ([]Artist, error) {
	var a []Artist
	return a, c.fetch(ctx, BaseURL+"/artists", &a)
}

// Relation fetches concert/tour information for a specific artist
// Called by: ArtistDetailsHandler (for displaying concerts when user clicks "Show Concerts")
// Returns: Relation struct (location→dates mapping) and error
//
// External API call: GET https://groupietrackers.herokuapp.com/api/relation/{id}
// Example response: {"id": 1, "datesLocations": {"New York": ["01-01-2018"], "Paris": ["05-02-2018"]}}
//
// Performance note: This is fetched on-demand (not cached by default in cache.go)
// Consider adding per-artist relation caching if needed.
func (c *Client) Relation(ctx context.Context, id int) (Relation, error) {
	var r Relation
	return r, c.fetch(ctx, fmt.Sprintf("%s/relation/%d", BaseURL, id), &r)
}

// AllLocations fetches locations for all artists in one call
func (c *Client) AllLocations(ctx context.Context) (LocationIndex, error) {
	var l LocationIndex
	return l, c.fetch(ctx, BaseURL+"/locations", &l)
}

// AllDates fetches dates for all artists in one call
func (c *Client) AllDates(ctx context.Context) (DateIndex, error) {
	var d DateIndex
	return d, c.fetch(ctx, BaseURL+"/dates", &d)
}

// Locations fetches concert locations for a specific artist
// Called by: ArtistLocationsHandler
// Returns: Locations struct and error
//
// External API call: GET https://groupietrackers.herokuapp.com/api/locations/{id}
func (c *Client) Locations(ctx context.Context, id int) (Locations, error) {
	var l Locations
	return l, c.fetch(ctx, fmt.Sprintf("%s/locations/%d", BaseURL, id), &l)
}

// Dates fetches concert dates for a specific artist
// Called by: ArtistDatesHandler
// Returns: Dates struct and error
//
// External API call: GET https://groupietrackers.herokuapp.com/api/dates/{id}
func (c *Client) Dates(ctx context.Context, id int) (Dates, error) {
	var d Dates
	return d, c.fetch(ctx, fmt.Sprintf("%s/dates/%d", BaseURL, id), &d)
}

func (c *Client) GetCoords(ctx context.Context, locationKey string) (FeatureCollection, error) {
	var geoData FeatureCollection
	loc := strings.ReplaceAll(locationKey, "-", "+")
	nominatimURL := fmt.Sprintf("%s/search?q=%s&format=geojson", Nominatim, loc)
	return geoData, c.fetch(ctx, nominatimURL, &geoData)
}
