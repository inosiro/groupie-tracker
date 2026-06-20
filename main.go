// ═══════════════════════════════════════════════════════════════════════════
// GROUPIE TRACKER - MAIN SERVER ENTRY POINT
// ═══════════════════════════════════════════════════════════════════════════
//
// KEY FEATURES:
// ✓ View concert dates grouped by location
// ✓ View focused lists of members, locations, and dates
// ✓ Error handling (upstream API failures don't crash server)
// ✓ Cache optimization (reduces external API calls)
//
// DATA FLOW:
//   External API (Groupie Trackers)
//           ↓
//   API Client (groupie_api.go)
//           ↓
//   Cache Layer (groupie_cache.go) [TTL=2min]
//           ↓
//   HTTP Handlers (web_handlers.go)
//           ↓
//   View Models (web_viewmodels.go)
//           ↓
//   Templates (html/template)
//           ↓
//   Browser (HTMX updates DOM)
//
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"groupie-tracker/internal"
	"log"
	"net/http"
	"time"
)

func main() {
	// ───────────────────────────────────────────────────────────────────────
	// STEP 1: INITIALIZE DEPENDENCIES
	// ───────────────────────────────────────────────────────────────────────
	// Create API client and cache at startup.
	// These are singletons shared across all request handlers.

	// api: HTTP client for Groupie Trackers external API
	// Used by: Cache (for fetching artists list)
	//          ArtistDetailsHandler (for fetching concert relations)
	api := internal.NewClient()

	// cache: TTL-based cache for artists list
	// TTL: 2 minutes (configurable)
	// - Stores: []Artist (52 artists from Groupie Trackers)
	// - Hit path: ~1ms (returns cached data)
	// - Miss path: ~500-1000ms (external API call + cache store)
	//
	// Why 2 minutes?
	//   - Short enough: data changes are reflected quickly
	//   - Long enough: significantly reduces API calls (10x reduction typical)
	//   - Goldilocks: balances freshness vs. performance
	cache := internal.NewCache(api, 5*time.Minute)

	// ───────────────────────────────────────────────────────────────────────
	// STEP 2: SETUP HTTP ROUTING
	// ───────────────────────────────────────────────────────────────────────
	// Use Go 1.22+ http.ServeMux with method+pattern syntax (e.g. "GET /path")
	mux := http.NewServeMux()

	// ─ PAGES ─────────────────────────────────────────────────────────────
	// Full page responses (with layout wrapper)

	// Route: / (Catch-all)
	// Handler: Index
	// Purpose: Landing page and fallback for non-existent routes
	mux.HandleFunc("/", internal.Index(cache))

	// ─ ENDPOINTS ─────────────────────────────────────────────────────────
	// Data endpoints (return HTML fragments for HTMX, or full page if direct)

	// Route: GET /artists
	// Handler: ArtistsHandler
	// Returns:
	//   - If HX-Request=true: HTML fragment (artist_grid.html partial)
	//   - Otherwise: Full page (for direct browser access)
	mux.HandleFunc("GET /artists", internal.ArtistsHandler(cache, api))

	// Route: GET /artists/{id}
	// Path param: {id} = artist ID (1-52)
	// Handler: ArtistDetailsHandler
	// Returns: HTML fragment (artist_details.html partial) with:
	//   - Artist member list
	//   - Concert dates grouped by location
	// Purpose: Show concert details when user clicks "Show Concerts" button
	// Triggered by: HTMX button in artist card (hx-get="/artists/{id}")
	mux.HandleFunc("GET /artists/{id}", internal.ArtistDetailsHandler(cache, api))

	// Route: GET /members/{id}
	mux.HandleFunc("GET /members/{id}", internal.ArtistMembersHandler(cache))

	// Route: GET /locations/{id}
	// Purpose: Show concert locations for a specific artist
	mux.HandleFunc("GET /locations/{id}", internal.ArtistLocationsHandler(cache, api))

	// Route: GET /dates/{id}
	// Purpose: Show concert dates for a specific artist
	mux.HandleFunc("GET /dates/{id}", internal.ArtistDatesHandler(cache, api))

	// Route: GET /artists/{id}/concerts.geojson
	// Purpose: retrieve the geoJSON of concerts for a specific artist

	geoHandler := &internal.GeoJSONHandler{
		Cache:    cache,
		API:      api,
		Resolver: cache,
	}
	mux.HandleFunc("GET /artists/{id}/concerts.geojson", geoHandler.ArtistConcertsGeoJSON)

	// ─ HEALTH CHECK ──────────────────────────────────────────────────────

	// Route: GET /healthz
	// Handler: Inline health check
	// Returns: 200 OK with "ok" text
	// Purpose: Uptime monitoring, container health probes, load balancer checks
	// Requirement: FR6 (must not crash, stay responsive)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// ─ STATIC FILES ──────────────────────────────────────────────────────

	// Route: GET /static/*
	// Handler: Static file server
	// Serves: CSS, HTMX library
	// Files:
	//   - /static/styles.css      (CSS styling)
	//   - /static/htmx.min.js     (HTMX library for interactive updates)
	//
	// Flow: GET /static/styles.css
	//       → StripPrefix("/static/") removes prefix
	//       → FileServer serves from "static/" directory
	//       → /static/styles.css → static/styles.css (on disk)
	mux.HandleFunc("GET /static/", func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP(w, r)
	})

	// ───────────────────────────────────────────────────────────────────────
	// STEP 3: START SERVER
	// ───────────────────────────────────────────────────────────────────────

	// Log server startup
	log.Println("Server running at http://localhost:8080")

	// Start HTTP server on port 8080
	// ListenAndServe blocks indefinitely (until server is shut down)
	// If server fails to start, log.Fatal terminates the program
	//
	// Example URLs to test:
	//   http://localhost:8080/                    (home page)
	//   http://localhost:8080/healthz              (health check)
	log.Fatal(http.ListenAndServe(":8080", mux))
}
