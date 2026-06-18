package internal

import (
	"sort"
	"strings"
)

func BuildGeoJSON(artist Artist, relation Relation, resolver CoordResolver) FeatureCollection {
	features := make([]Feature, 0)

	locationKeys := make([]string, 0, len(relation.DatesLocations))
	for locationKey := range relation.DatesLocations {
		locationKeys = append(locationKeys, locationKey)
	}
	sort.Strings(locationKeys)

	for _, locationKey := range locationKeys {
		dates := relation.DatesLocations[locationKey]

		lng, lat, ok := resolver.Lookup(locationKey)
		if !ok {
			// skip unmapped locations
			continue
		}

		label := strings.ReplaceAll(locationKey, "_", " ")
		label = strings.ReplaceAll(label, "-", ", ")

		sort.Strings(dates)

		for _, date := range dates {
			features = append(features, Feature{
				Type: "Feature",
				Geometry: Geometry{
					Type:        "Point",
					Coordinates: [2]float64{lng, lat},
				},
				Properties: ConcertProperties{
					ArtistID:      artist.ID,
					ArtistName:    artist.Name,
					LocationKey:   locationKey,
					LocationLabel: label,
					Date:          date,
				},
			})
		}
	}
	return FeatureCollection{
		Type:     "FeatureCollection",
		Features: features,
	}

}
