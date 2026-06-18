package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// HTTP REQUEST HANDLERS
// ═══════════════════════════════════════════════════════════════════════════
// These handlers process HTTP requests and return HTML responses.
// They orchestrate the data flow: Cache → Filters → View Models → Templates.
//
// HTMX Integration:
//   - If HX-Request header is true: return HTML fragment (partial template)
//   - Otherwise: return full page (layout + content)
//
// Main endpoints:
//   1. GET /                 → Index (page shell)
//   2. GET /artists          → Artist grid
//   3. GET /artists/{id}     → Artist concerts
//   4. GET /members/{id}     → Artist members
//   5. GET /locations/{id}   → Artist locations
//   6. GET /dates/{id}       → Artist dates
//   7. GET /dates/{id}       → Artist dates

// getArtistByID is a helper to fetch the artist list and find a specific ID.
// It centralizes error handling for the cache and the "not found" case.
func getArtistByID(ctx context.Context, cache CacheProvider, id int) (Artist, error) {
	artists, err := cache.Artists(ctx)
	if err != nil {
		return Artist{}, err
	}
	for _, a := range artists {
		if a.ID == id {
			return a, nil
		}
	}
	return Artist{}, fmt.Errorf("artist not found")
}

// Index handler renders the home page shell
// Endpoint: GET /
// Returns: Full HTML page (layout.html + index.html)
//
// Process:
//  1. If path is not "/", set 404 status (acts as catch-all)
//  2. Render the index page template
//  3. On browser load, HTMX triggers GET /artists to populate the grid
//
// Why separate Index handler?
//   - Acts as a fallback for non-existent routes (returning the UI shell)
//   - All subsequent data loads use GET /artists (partial update)
//   - Keeps concerns separated: layout vs. content
//
// Note: The actual artist grid is loaded via HTMX (see index.html template)
func Index(cache CacheProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set 404 status for non-existent paths while still rendering the index UI
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)

			// If this was an HTMX request for a non-existent partial/route,
			// return an error fragment instead of the full page shell.
			if r.Header.Get("HX-Request") == "true" {
				RenderPartial(w, "error_banner", "404: The requested resource was not found.")
				return
			}

			// For direct browser access to a 404 path, render the index shell with an error state.
			if err := RenderPage(w, "index", GridVM{HasError: true, Error: "404: Page not found"}); err != nil {
				http.Error(w, "failed to render page", http.StatusInternalServerError)
			}
			return
		}

		// Render full page with index layout
		// The page will load /artists via HTMX on load
		if err := RenderPage(w, "index", nil); err != nil {
			http.Error(w, "failed to render page", http.StatusInternalServerError)
		}
	}
}

// ArtistsHandler returns the artist grid
// Endpoint: GET /artists
// Returns:
//   - If HX-Request=true: HTML fragment (artist_grid.html)
//   - Otherwise: full page (layout + index + grid)
//
// Data Flow:
//  1. Fetch cached artists list (cache handles TTL)
//  2. Convert to view models (for template rendering)
//  3. Return HTML fragment or full page based on HX-Request header
//
// Error Handling:
//   - If API call fails: return error banner with friendly message
//   - Server continues to work (graceful degradation)
//
// Performance:
//   - With cache HIT: ~5-10ms (no external API call)
//   - With cache MISS: ~500-1000ms (includes API call)
func ArtistsHandler(cache CacheProvider, api *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create context with 5-second timeout for API calls
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Step 1: Parse and apply filters
		filter := ParseFilter(r)

		// Step 2: Fetch cached artists list
		// - Cache HIT: returns immediately from in-memory storage
		// - Cache MISS: fetches from external API and caches result
		artists, err := cache.Artists(ctx)
		if err != nil {
			// Upstream API failed - show error but don't crash
			gridVM := GridVM{
				HasError: true,
				Error:    "Failed to load artists. Please try again later.",
				Count:    0,
			}

			isHTMXRequest := r.Header.Get("HX-Request") == "true"
			if isHTMXRequest {
				// HTMX request: return fragment only
				RenderPartial(w, "artist_grid", gridVM)
			} else {
				// Browser request: return full page with error
				RenderPage(w, "index", gridVM)
			}
			return
		}

		// Step 3: Apply the filtering logic using cached bulk indices
		filteredArtists := ApplyFilters(ctx, cache, artists, filter)

		// Step 4: Convert to view models for template rendering
		// View models separate template concerns from data model concerns
		var cardVMs []ArtistCardVM
		for _, artist := range filteredArtists {
			cardVMs = append(cardVMs, ArtistCardVM{
				ID:           artist.ID,
				Name:         artist.Name,
				Image:        artist.Image,
				MembersCount: len(artist.Members),
				Members:      artist.Members,
				CreationDate: artist.CreationDate,
				FirstAlbum:   artist.FirstAlbum,
			})
		}

		gridVM := GridVM{
			Artists:  cardVMs,
			Query:    filter,       // Include filters for UI feedback
			Count:    len(cardVMs), // Number of results
			HasError: false,
		}

		// Step 5: Return HTML fragment or full page
		isHTMXRequest := r.Header.Get("HX-Request") == "true"
		if isHTMXRequest {
			// If triggered by the auto-load on the grid, check if we are on a specific resource page
			if r.Header.Get("HX-Trigger") == "artist-grid" {
				currentURL := r.Header.Get("HX-Current-Url")
				if currentURL != "" && !strings.HasSuffix(currentURL, "/") &&
					!strings.HasSuffix(currentURL, "/artists") && !strings.HasSuffix(currentURL, "/artists/") {
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			// HTMX triggered this: return only the grid fragment
			// Browser will swap it into #artist-grid div
			RenderPartial(w, "artist_grid", gridVM)
		} else {
			// Direct browser request: return full page including grid
			RenderPage(w, "index", gridVM)
		}
	}
}

// ArtistDetailsHandler returns artist details fragment (members + concerts)
// Endpoint: GET /artists/{id}
// Path Parameter:
//   - {id}: artist ID (1-52 for Groupie Trackers)
//
// Returns:
//   - HTML fragment (artist_details.html) with members list and concerts by location
//
// This satisfies the "event-driven" requirement:
//   - User clicks "Show Concerts" button in artist card
//   - Button has: hx-get="/artists/{id}" hx-target="#details-{id}"
//   - Server returns HTML fragment with concert data
//   - HTMX swaps fragment into the card
//
// Data Flow:
//  1. Extract and validate artist ID from URL path
//  2. Fetch cached artists list to get artist details
//  3. Fetch concert relations from external API for that artist
//  4. Transform relation data (map → slice for template iteration)
//  5. Return HTML fragment
//
// Note: Relation data is fetched on-demand (not cached)
// Future optimization: could add per-artist relation caching
func ArtistDetailsHandler(cache CacheProvider, api *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract artist ID from URL path parameter
		// Pattern: /artists/{id}
		idStr := r.PathValue("id") // Go 1.22+ path parameter extraction
		if idStr == "" {
			http.Error(w, "artist id required", http.StatusBadRequest)
			return
		}

		// Parse and validate ID as integer
		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			http.Error(w, "invalid artist id", http.StatusBadRequest)
			return
		}

		// Create context with 5-second timeout
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		artist, err := getArtistByID(ctx, cache, id)
		if err != nil {
			if r.Header.Get("HX-Request") != "true" {
				RenderPage(w, "index", GridVM{HasError: true, Error: err.Error()})
				return
			}
			RenderPartial(w, "error_banner", err.Error())
			return
		}

		// Step 3: Fetch concert relations for this artist
		// This is an external API call for per-artist concert/tour data
		relation, err := api.Relation(ctx, id)
		if err != nil {
			if r.Header.Get("HX-Request") != "true" {
				RenderPage(w, "index", GridVM{HasError: true, Error: "Failed to load concerts"})
				return
			}
			// For HTMX, return 200 so the error banner actually swaps into the UI
			RenderPartial(w, "error_banner", "Failed to load concerts")
			return
		}

		// Step 4: Transform relation data structure
		// Maps have random iteration order, so sort locations first for stable output
		var locations []string
		for location := range relation.DatesLocations {
			locations = append(locations, location)
		}
		sort.Strings(locations)

		var concerts []LocationConcertVM
		for _, location := range locations {
			concerts = append(concerts, LocationConcertVM{
				Location: location,
				Dates:    relation.DatesLocations[location],
			})
		}

		// Step 5: Create view model for template
		detailsVM := DetailsVM{
			Artist: ArtistCardVM{
				ID:           artist.ID,
				Name:         artist.Name,
				Image:        artist.Image,
				MembersCount: len(artist.Members),
				Members:      artist.Members,
				CreationDate: artist.CreationDate,
				FirstAlbum:   artist.FirstAlbum,
			},
			Concerts: concerts,
		}

		// Step 6: Return HTML fragment
		isHTMXRequest := r.Header.Get("HX-Request") == "true"
		if isHTMXRequest {
			// HTMX will swap this into the #details-{id} div in the artist card
			RenderPartial(w, "artist_details", detailsVM)
		} else {
			// Direct browser request: return full page showing only this artist
			artistCard := detailsVM.Artist
			artistCard.Details = &detailsVM
			RenderPage(w, "index", GridVM{
				Artists: []ArtistCardVM{artistCard},
				Count:   1,
			})
		}
	}
}

// ArtistLocationsHandler returns artist locations fragment
// Endpoint: GET /locations/{id}
func ArtistLocationsHandler(cache CacheProvider, api *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		if idStr == "" {
			http.Error(w, "artist id required", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			http.Error(w, "invalid artist id", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Step 1: Verify artist exists
		artists, err := cache.Artists(ctx)
		if err != nil {
			RenderPartial(w, "error_banner", "Service unavailable")
			return
		}

		var artist Artist
		found := false
		for _, a := range artists {
			if a.ID == id {
				artist = a
				found = true
				break
			}
		}

		if !found {
			RenderPartial(w, "error_banner", "Artist not found")
			return
		}

		locations, err := api.Locations(ctx, id)
		if err != nil {
			RenderPartial(w, "error_banner", "Failed to load locations")
			return
		}

		locationsVM := LocationsVM{
			ID:        id,
			Locations: locations.Locations,
		}

		if r.Header.Get("HX-Request") == "true" {
			RenderPartial(w, "artist_locations", locationsVM)
		} else {
			card := ArtistCardVM{
				ID:           artist.ID,
				Name:         artist.Name,
				Image:        artist.Image,
				MembersCount: len(artist.Members),
				Members:      artist.Members,
				CreationDate: artist.CreationDate,
				FirstAlbum:   artist.FirstAlbum,
				Locations:    &locationsVM,
			}
			RenderPage(w, "index", GridVM{
				Artists: []ArtistCardVM{card},
				Count:   1,
			})
		}
	}
}

// ArtistDatesHandler returns artist dates fragment
// Endpoint: GET /dates/{id}
func ArtistDatesHandler(cache CacheProvider, api *Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		if idStr == "" {
			http.Error(w, "artist id required", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			http.Error(w, "invalid artist id", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Step 1: Verify artist exists
		artists, err := cache.Artists(ctx)
		if err != nil {
			RenderPartial(w, "error_banner", "Service unavailable")
			return
		}

		var artist Artist
		found := false
		for _, a := range artists {
			if a.ID == id {
				artist = a
				found = true
				break
			}
		}

		if !found {
			RenderPartial(w, "error_banner", "Artist not found")
			return
		}

		dates, err := api.Dates(ctx, id)
		if err != nil {
			RenderPartial(w, "error_banner", "Failed to load dates")
			return
		}

		datesVM := DatesVM{
			ID:    id,
			Dates: dates.Dates,
		}

		if r.Header.Get("HX-Request") == "true" {
			RenderPartial(w, "artist_dates", datesVM)
		} else {
			card := ArtistCardVM{
				ID:           artist.ID,
				Name:         artist.Name,
				Image:        artist.Image,
				MembersCount: len(artist.Members),
				Members:      artist.Members,
				CreationDate: artist.CreationDate,
				FirstAlbum:   artist.FirstAlbum,
				Dates:        &datesVM,
			}
			RenderPage(w, "index", GridVM{
				Artists: []ArtistCardVM{card},
				Count:   1,
			})
		}
	}
}

// ArtistMembersHandler returns artist members fragment
// Endpoint: GET /members/{id}
func ArtistMembersHandler(cache CacheProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		if idStr == "" {
			http.Error(w, "artist id required", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil || id <= 0 {
			http.Error(w, "invalid artist id", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		artists, err := cache.Artists(ctx)
		if err != nil {
			RenderPartial(w, "error_banner", "Failed to load artists")
			return
		}

		for _, artist := range artists {
			if artist.ID == id {
				// Found the artist, render members partial
				vm := ArtistCardVM{
					ID:           artist.ID,
					Name:         artist.Name,
					Image:        artist.Image,
					Members:      artist.Members,
					MembersCount: len(artist.Members),
					CreationDate: artist.CreationDate,
					FirstAlbum:   artist.FirstAlbum,
					ShowMembers:  true,
				}

				if r.Header.Get("HX-Request") == "true" {
					RenderPartial(w, "artist_members", vm)
				} else {
					RenderPage(w, "index", GridVM{
						Artists: []ArtistCardVM{vm},
						Count:   1,
					})
				}
				return
			}
		}

		RenderPartial(w, "error_banner", "Artist not found")
	}
}

// ParseFilter extracts filter parameters from the request query string
func ParseFilter(r *http.Request) FilterVM {
	filter := FilterVM{
		Query:    r.URL.Query().Get("q"),
		Sort:     r.URL.Query().Get("sort"),
		Location: r.URL.Query().Get("location"),
		Date:     r.URL.Query().Get("date"),
	}

	// Helper to safely parse ints
	p := func(key string) int {
		val := r.URL.Query().Get(key)
		if val != "" {
			if num, err := strconv.Atoi(val); err == nil && num > 0 {
				return num
			}
		}
		return 0
	}

	filter.CreationFrom = p("creation_from")
	filter.CreationTo = p("creation_to")
	filter.FirstAlbumFrom = p("first_album_from")
	filter.FirstAlbumTo = p("first_album_to")
	filter.MembersMin = p("members_min")
	filter.MembersMax = p("members_max")

	// Parse multi-select checkboxes for member counts
	for _, mStr := range r.URL.Query()["members"] {
		if num, err := strconv.Atoi(mStr); err == nil && num > 0 {
			filter.Members = append(filter.Members, num)
		}
	}

	// Parse multi-select checkboxes for eras
	for _, eStr := range r.URL.Query()["eras"] {
		if num, err := strconv.Atoi(eStr); err == nil && num > 0 {
			filter.Eras = append(filter.Eras, num)
		}
	}

	// Parse multi-select checkboxes for regions
	filter.Regions = r.URL.Query()["regions"]

	// Check if include_relations checkbox is checked (HTMX sends "1" for checked)
	filter.IncludeRelations = r.URL.Query().Get("include_relations") == "1"

	return filter
}

type GeoJSONHandler struct {
	Cache    *Cache
	API      *Client
	Resolver CoordResolver
}

func (h *GeoJSONHandler) ArtistConcertsGeoJSON(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid artist id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	artists, err := h.Cache.Artists(ctx)
	if err != nil {
		http.Error(w, "failed to load artists", http.StatusInternalServerError)
		return
	}

	var artist Artist
	found := false
	for _, a := range artists {
		if a.ID == id {
			artist = a
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "artist not found", http.StatusNotFound)
		return
	}

	relation, err := h.API.Relation(ctx, id)
	if err != nil {
		http.Error(w, "failed to load relation", http.StatusBadGateway)
		return
	}

	fc := BuildGeoJSON(artist, relation, h.Resolver)

	w.Header().Set("Content-Type", "application/geo+json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	json.NewEncoder(w).Encode(fc)
}
