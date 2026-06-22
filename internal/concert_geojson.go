package internal

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Helper to parse Groupie Trackers' dates ("DD-MM-YYYY" or "*DD-MM-YYYY")
func parseConcertDate(dateStr string) (time.Time, error) {
	cleanDate := strings.TrimPrefix(dateStr, "*")
	return time.Parse("02-01-2006", cleanDate)
}

// FlatConcert holds the flat representation of a concert for sorting
type FlatConcert struct {
	LocationKey   string
	LocationLabel string
	DateStr       string
	ParsedDate    time.Time
	Lng           float64
	Lat           float64
}

func BuildGeoJSON(ctx context.Context, h *WebHandler, artist Artist, relation Relation) FeatureCollection {
	var flatConcerts []FlatConcert

	// fill the FlatConcert struct
	for locationKey, dates := range relation.DatesLocations {
		lng, lat, ok := h.Cache.Lookup(locationKey)
		if !ok {
			geoData, err := h.Api.GetCoords(ctx, locationKey)
			if err != nil {
				fmt.Println(err)
				continue
			}
			// parse the first location, it has the best probability to be correct
			lng = geoData.Features[0].Geometry.Coordinates[0]
			lat = geoData.Features[0].Geometry.Coordinates[1]
			h.Cache.Store(locationKey, lng, lat)
		}
		label := strings.ReplaceAll(locationKey, "_", " ")
		label = strings.ReplaceAll(label, "-", ", ")
		for _, date := range dates {
			parsedDate, err := parseConcertDate(date)
			if err != nil {
				parsedDate = time.Unix(0, 0)
			}
			flatConcerts = append(flatConcerts, FlatConcert{
				LocationKey:   locationKey,
				LocationLabel: label,
				DateStr:       date,
				ParsedDate:    parsedDate,
				Lng:           lng,
				Lat:           lat,
			})
		}
	}

	// sort concerts by date
	sort.Slice(flatConcerts, func(i, j int) bool {
		return flatConcerts[i].ParsedDate.Before(flatConcerts[j].ParsedDate)
	})

	// build json features that are sorted by date
	features := make([]Feature, 0, len(flatConcerts))
	for i, c := range flatConcerts {
		features = append(features, Feature{
			Type: "Feature",
			Geometry: Geometry{
				Type:        "Point",
				Coordinates: [2]float64{c.Lng, c.Lat},
			},
			Properties: ConcertProperties{
				SequenceIndex: i + 1,
				ArtistID:      artist.ID,
				ArtistName:    artist.Name,
				LocationKey:   c.LocationKey,
				LocationLabel: c.LocationLabel,
				Date:          c.DateStr,
			},
		})

	}

	return FeatureCollection{
		Type:     "FeatureCollection",
		Features: features,
	}
}
