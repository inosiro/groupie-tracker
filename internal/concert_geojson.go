package internal

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

func BuildGeoJSON(ctx context.Context, h *WebHandler, artist Artist, relation Relation) FeatureCollection {
	features := make([]Feature, 0)

	locationKeys := make([]string, 0, len(relation.DatesLocations))
	for locationKey := range relation.DatesLocations {
		locationKeys = append(locationKeys, locationKey)
	}
	sort.Strings(locationKeys)

	for _, locationKey := range locationKeys {
		dates := relation.DatesLocations[locationKey]

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
