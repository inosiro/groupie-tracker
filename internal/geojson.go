package internal

type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

type Feature struct {
	Type       string            `json:"type"`
	Properties ConcertProperties `json:"properties"`
	Geometry   Geometry          `json:"geometry"`
}

type ConcertProperties struct {
	ArtistID      int    `json:"artist_id"`
	ArtistName    string `json:"artist_name"`
	LocationKey   string `json:"location_key"`
	LocationLabel string `json:"location_label"`
	Date          string `json:"date"`
}

type Geometry struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"` // [lng, lat]
}
