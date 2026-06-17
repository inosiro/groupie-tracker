package internal

// ═══════════════════════════════════════════════════════════════════════════
// GROUPIE TRACKER DATA MODELS
// ═══════════════════════════════════════════════════════════════════════════
// These structs map directly to the Groupie Trackers external API responses.
// They define the core data entities used throughout the application.
//
// Data Flow: External API → Models → Cache → Handlers → View Models → Templates

// Artist represents a band/artist from the Groupie Trackers API
// Source: GET https://groupietrackers.herokuapp.com/api/artists
type Artist struct {
	ID           int      `json:"id"`           // Unique artist identifier
	Image        string   `json:"image"`        // URL to artist image (displayed in cards)
	Name         string   `json:"name"`         // Artist/band name (searchable)
	Members      []string `json:"members"`      // Array of member names (searchable)
	CreationDate int      `json:"creationDate"` // Year band was formed (filterable)
	FirstAlbum   string   `json:"firstAlbum"`   // First album release date string
}

// Relation represents concert information for a specific artist
// Source: GET https://groupietrackers.herokuapp.com/api/relation/{id}
// This is fetched per-artist when viewing concert details.
type Relation struct {
	ID             int                 `json:"id"`             // Artist ID (matches Artist.ID)
	DatesLocations map[string][]string `json:"datesLocations"` // Map of location→dates (concert data)
	// Example: {"Athens": ["12-01-2018", "15-01-2018"], "Paris": ["22-03-2018"]}
}

// ArtistDetails is a composite view combining artist + relation data
// (Currently unused in HTMX architecture, kept for potential API endpoints)
// type ArtistDetails struct {
// 	Artist   Artist   `json:"artist"`
// 	Relation Relation `json:"relation"`
// }

// Locations represents the concert locations for a specific artist
// Source: GET https://groupietrackers.herokuapp.com/api/locations/{id}
type Locations struct {
	ID        int      `json:"id"`
	Locations []string `json:"locations"`
}

// Dates represents the concert dates for a specific artist
// Source: GET https://groupietrackers.herokuapp.com/api/dates/{id}
type Dates struct {
	ID    int      `json:"id"`
	Dates []string `json:"dates"`
}

// LocationIndex represents the bulk response for all locations
type LocationIndex struct {
	Index []Locations `json:"index"`
}

// DateIndex represents the bulk response for all dates
type DateIndex struct {
	Index []Dates `json:"index"`
}
