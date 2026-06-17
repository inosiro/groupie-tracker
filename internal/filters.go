package internal

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// ApplyFilters orchestrates the filtering process.
func ApplyFilters(ctx context.Context, cache CacheProvider, artists []Artist, filter FilterVM) []Artist {
	var result []Artist

	var locMap map[int][]string
	var dateMap map[int][]string

	// If concert filters are requested, fetch the bulk indices in parallel.
	if filter.Location != "" || filter.Date != "" || len(filter.Regions) > 0 {
		var wg sync.WaitGroup

		// Only fetch locations if needed for city or region filtering
		if filter.Location != "" || len(filter.Regions) > 0 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if locIdx, err := cache.Locations(ctx); err == nil {
					m := make(map[int][]string)
					for _, l := range locIdx.Index {
						m[l.ID] = l.Locations
					}
					locMap = m
				}
			}()
		}

		// Only fetch dates if needed for date filtering
		if filter.Date != "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if dateIdx, err := cache.Dates(ctx); err == nil {
					m := make(map[int][]string)
					for _, d := range dateIdx.Index {
						m[d.ID] = d.Dates
					}
					dateMap = m
				}
			}()
		}
		wg.Wait()
	}

	for _, artist := range artists {
		if !matchesMetadata(artist, filter) {
			continue
		}

		if (locMap != nil || dateMap != nil) && !matchesRelations(artist.ID, locMap, dateMap, filter) {
			continue
		}

		result = append(result, artist)
	}

	// if filter.Sort != "" {
	// 	result = sortArtists(result, filter.Sort)
	// }
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	// result = sortArtists(result, "name")
	return result
}

func matchesMetadata(artist Artist, filter FilterVM) bool {
	// General Search (Query)
	// if filter.Query != "" {
	// 	query := strings.ToLower(filter.Query)
	// 	nameMatch := strings.Contains(strings.ToLower(artist.Name), query)
	// 	memberMatch := false
	// 	for _, m := range artist.Members {
	// 		if strings.Contains(strings.ToLower(m), query) {
	// 			memberMatch = true
	// 			break
	// 		}
	// 	}
	// 	if !nameMatch && !memberMatch {
	// 		return false
	// 	}
	// }

	// Creation Year (Range Filter)
	// If Eras (decades) are selected, the custom range sliders are logically ignored/disabled.
	if len(filter.Eras) == 0 {
		if filter.CreationFrom > 0 && artist.CreationDate < filter.CreationFrom {
			return false
		}
		if filter.CreationTo > 0 && artist.CreationDate > filter.CreationTo {
			return false
		}
	}

	// First Album Year (Range Filter)
	if filter.FirstAlbumFrom > 0 || filter.FirstAlbumTo > 0 {
		albumYear := extractYear(artist.FirstAlbum)
		if filter.FirstAlbumFrom > 0 && albumYear < filter.FirstAlbumFrom {
			return false
		}
		if filter.FirstAlbumTo > 0 && albumYear > filter.FirstAlbumTo {
			return false
		}
	}

	// Member Count (Range AND Checkbox Filter)
	memberCount := len(artist.Members)
	// Checkbox multi-selection (if selected)
	if len(filter.Members) > 0 {
		found := false
		for _, c := range filter.Members {
			// If 8 is selected (labeled 8+ in UI), match 8 or more members
			if (c == 8 && memberCount >= 8) || c == memberCount {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Eras (Decade Checkbox Filter)
	if len(filter.Eras) > 0 {
		match := false
		for _, era := range filter.Eras {
			if artist.CreationDate >= era && artist.CreationDate < era+10 {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	// Range selection
	// If specific member counts are selected via checkboxes, the range inputs are logically ignored.
	if len(filter.Members) == 0 {
		if filter.MembersMin > 0 && memberCount < filter.MembersMin {
			return false
		}
		if filter.MembersMax > 0 && memberCount > filter.MembersMax {
			return false
		}
	}

	return true
}

func matchesRelations(id int, locMap map[int][]string, dateMap map[int][]string, filter FilterVM) bool {
	// Regions (Geo-filtering)
	if len(filter.Regions) > 0 {
		regionKeywords := map[string][]string{
			"north_america": {"usa", "mexico", "canada"},
			"europe":        {"uk", "france", "germany", "italy", "spain", "portugal", "netherlands", "belgium", "switzerland", "austria", "denmark", "sweden", "norway", "finland", "poland", "greece", "ireland", "belarus", "romania", "hungary", "czechia"},
			"oceania":       {"australia", "new_zealand"},
			"asia":          {"japan", "china", "india", "korea", "thailand"},
			"south_america": {"brazil", "argentina", "chile", "colombia"},
		}

		regionMatch := false
		for _, loc := range locMap[id] {
			locLower := strings.ToLower(loc)
			for _, region := range filter.Regions {
				keywords, ok := regionKeywords[region]
				if !ok {
					continue
				}
				for _, kw := range keywords {
					if strings.Contains(locLower, kw) {
						regionMatch = true
						break
					}
				}
				if regionMatch {
					break
				}
			}
			if regionMatch {
				break
			}
		}
		if !regionMatch {
			return false
		}
	}

	// If specific regions are selected, the location text input is logically ignored.
	if len(filter.Regions) == 0 && filter.Location != "" {
		if !matchesLocationTokens(locMap[id], filter.Location) {
			return false
		}
	}

	if filter.Date != "" {
		found := false
		for _, d := range dateMap[id] {
			if strings.Contains(d, filter.Date) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func locationTokens(query string) []string {
	parts := strings.FieldsFunc(strings.ToLower(query), func(r rune) bool {
		return r == ' ' || r == ',' || r == '-'
	})

	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token != "" {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

func matchesLocationTokens(locations []string, query string) bool {
	tokens := locationTokens(query)
	if len(tokens) == 0 {
		return true
	}

	locationData := strings.ToLower(strings.Join(locations, " "))
	for _, token := range tokens {
		if !strings.Contains(locationData, token) {
			return false
		}
	}
	return true
}

func extractYear(dateStr string) int {
	parts := strings.Split(dateStr, "-")
	if len(parts) == 3 {
		year, _ := strconv.Atoi(parts[2])
		return year
	}
	return 0
}

// func sortArtists(artists []Artist, sortBy string) []Artist {
// 	switch sortBy {
// 	case "name":
// 		sort.Slice(artists, func(i, j int) bool { return artists[i].Name < artists[j].Name })
// 	case "creation":
// 		sort.Slice(artists, func(i, j int) bool { return artists[i].CreationDate < artists[j].CreationDate })
// 	case "members":
// 		sort.Slice(artists, func(i, j int) bool { return len(artists[i].Members) < len(artists[j].Members) })
// 	}
// 	return artists
// }
