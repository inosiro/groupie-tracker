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
	SequenceIndex int    `json:"sequence_index"`
	ArtistID      int    `json:"artist_id"`
	ArtistName    string `json:"artist_name"`
	LocationKey   string `json:"location_key"`
	LocationLabel string `json:"location_label"`
	JoinedDates   string `json:"joined_dates"` // "10-11-2000"+\n+"20-11-2000"
}

type Geometry struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"` // [lng, lat]
}
