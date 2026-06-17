package internal

// ═══════════════════════════════════════════════════════════════════════════
// VIEW MODELS FOR TEMPLATE RENDERING
// ═══════════════════════════════════════════════════════════════════════════
// View models are data structures optimized for template rendering.
// They separate presentation concerns from data model concerns.
//
// Transformation Flow:
//   Artist (API) → ArtistCardVM (template-ready)
//   Relation (API) → LocationConcertVM (template-ready)
//   Locations (API) → LocationsVM (template-ready)
//   Dates (API) → DatesVM (template-ready)
//   Artists → GridVM (page context)

// ArtistCardVM is the view model for rendering a single artist card
// Contains all information needed for a card: image, name, stats, members
// This is separate from the API Artist struct to allow custom transformations
//
// Used by templates:
//   - artist_card.html (in artist_grid loop)
//   - artist_details.html (selected artist header)
type ArtistCardVM struct {
	ID           int      // Unique artist ID (used in HTMX buttons)
	Name         string   // Artist/band name (displayed)
	Image        string   // Image URL (src attribute)
	MembersCount int      // Total member count (calculated from len(Members))
	Members      []string // Array of member names (listed in details)
	CreationDate int      // Year formed (displayed in card)
	FirstAlbum   string   // First album date string (displayed in card)
	Details      *DetailsVM
	Locations    *LocationsVM
	Dates        *DatesVM
	ShowMembers  bool
}

// LocationConcertVM represents concerts for a specific location
// This is derived from the Relation.DatesLocations map
//
// Why a struct instead of a map?
//   - Maps don't have stable iteration order in Go
//   - Templates need to iterate through locations consistently
//   - Converting map → slice of structs ensures deterministic rendering
//
// Used by templates:
//   - artist_details.html (concert grouping loop)
type LocationConcertVM struct {
	Location string   // City/location name (concert venue location)
	Dates    []string // Array of concert dates for that location
}

// DetailsVM is the view model for the artist details fragment
// Contains full context for displaying a single artist's concert information
//
// Used by templates:
//   - artist_details.html (full details view)
//
// Rendering context:
//   - Artist info (members, creation date, image)
//   - Concerts grouped by location with dates
//   - Displayed when user clicks "Show Concerts" button
type DetailsVM struct {
	Artist   ArtistCardVM        // Artist information
	Concerts []LocationConcertVM // Concerts organized by location
}

// GridVM is the view model for the artist grid
// Contains all context needed to render the artist list page
//
// Used by templates:
//   - artist_grid.html (main container)
//   - Conditionally renders: error banner OR empty state OR results
//
// Rendering context:
//   - Error state: shows error banner if API failed
//   - Empty state: shows "no results" if filters match nothing
//   - Success state: shows grid of artist cards with result count
type GridVM struct {
	Artists  []ArtistCardVM // Filtered list of artists (0-52)
	Query    FilterVM       // Active filters (for UI feedback)
	Error    string         // Error message (if HasError=true)
	Count    int            // Number of results
	HasError bool           // True if API call failed
}

// FilterVM contains validated and parsed filter parameters
// These are extracted from the HTTP query string and validated
//
// Validation rules (see filter_query.go):
//   - Numeric fields must parse as integers
//   - Invalid values are set to 0 (meaning "no filter")
//   - String fields are passed through (sanitized by template auto-escape)
//
// Application rules (see filter_apply.go):
//   - All criteria use AND logic (all must match)
//   - 0 values mean "ignore this filter"
//   - Query searches both artist name and member names
//
// Used by:
//   - web_handlers.go: passed to ApplyFilters()
//   - templates: displayed in filter UI for user feedback
type FilterVM struct {
	// Search and text filters
	Query string // Free-text search (artist name or member name)
	Sort  string // Sort field: name|creation|members

	// Date/year range filters (for artists)
	CreationFrom   int // Minimum creation year
	CreationTo     int // Maximum creation year
	FirstAlbumFrom int // Minimum first album year
	FirstAlbumTo   int // Maximum first album year

	// Member count filters
	MembersMin int   // Minimum number of members (Range)
	MembersMax int   // Maximum number of members (Range)
	Members    []int // Specific member counts (Checkbox selection)

	// Concert/relation filters
	Location         string   // Substring match in concert locations
	Date             string   // Substring match in concert dates
	IncludeRelations bool     // Flag to apply expensive relation filtering
	Eras             []int    // Selected decades (e.g., 1960, 1970)
	Regions          []string // Selected regions (e.g., "europe", "north_america")
}

// LocationsVM is the view model for the artist locations fragment
// Used by templates: artist_locations.html
type LocationsVM struct {
	ID        int
	Locations []string
}

// DatesVM is the view model for the artist dates fragment
// Used by templates: artist_dates.html
type DatesVM struct {
	ID    int
	Dates []string
}
