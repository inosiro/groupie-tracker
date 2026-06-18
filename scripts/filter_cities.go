package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// normalize converts strings to the format used in Groupie Tracker locations (e.g., "New York" -> "new_york")
func normalize(s string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), " ", "_")
}

// countryCodeAliases maps normalized country names/ISO codes from worldcities.csv
// to the normalized short codes used in locations.txt.
// This helps match "united_kingdom" to "uk", "united_states" to "usa", etc.
var countryCodeAliases = map[string]string{
	"united_states":        "usa",
	"us":                   "usa",
	"usa":                  "usa", // self-alias
	"united_kingdom":       "uk",
	"gb":                   "uk",
	"uk":                   "uk", // self-alias
	"czechia":              "czechia",
	"cz":                   "czechia",
	"new_zealand":          "new_zealand",
	"nz":                   "new_zealand",
	"united_arab_emirates": "united_arab_emirates",
	"ae":                   "united_arab_emirates",
	"netherlands_antilles": "netherlands_antilles",
	"an":                   "netherlands_antilles",
	"french_polynesia":     "french_polynesia",
	"pf":                   "french_polynesia",
	"new_caledonia":        "new_caledonia",
	"nc":                   "new_caledonia",
	"japan":                "japan",
	"jp":                   "japan",
	"jpn":                  "japan",
	"germany":              "germany",
	"de":                   "germany",
	"deu":                  "germany",
	"france":               "france",
	"fr":                   "france",
	"fra":                  "france",
	"spain":                "spain",
	"es":                   "spain",
	"esp":                  "spain",
	"italy":                "italy",
	"it":                   "italy",
	"ita":                  "italy",
	"canada":               "canada",
	"ca":                   "canada",
	"can":                  "canada",
	"australia":            "australia",
	"au":                   "australia",
	"aus":                  "australia",
	"brazil":               "brazil",
	"br":                   "brazil",
	"bra":                  "brazil",
	"mexico":               "mexico",
	"mx":                   "mexico",
	"mex":                  "mexico",
	"argentina":            "argentina",
	"ar":                   "argentina",
	"arg":                  "argentina",
	"colombia":             "colombia",
	"co":                   "colombia",
	"col":                  "colombia",
	"peru":                 "peru",
	"pe":                   "peru",
	"per":                  "peru",
	"chile":                "chile",
	"cl":                   "chile",
	"chl":                  "chile",
	"philippines":          "philippines",
	"ph":                   "philippines",
	"phl":                  "philippines",
	"indonesia":            "indonesia",
	"id":                   "indonesia",
	"idn":                  "indonesia",
	"thailand":             "thailand",
	"th":                   "thailand",
	"tha":                  "thailand",
}

// TargetLocation represents a location from locations.txt
type TargetLocation struct {
	City        string
	CountryCode string // Normalized country code, e.g., "usa", "uk"
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run filter_cities.go <locations.txt> <worldcities.csv> <output.csv>")
		os.Exit(1)
	}

	locsFile, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("Error opening locations file: %v\n", err)
		os.Exit(1)
	}
	defer locsFile.Close()

	targets := make(map[TargetLocation]bool)
	scanner := bufio.NewScanner(locsFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Handle multiple dashes by taking the last part as the country
		parts := strings.Split(line, "-")
		if len(parts) == 2 {
			country := parts[1]
			city := parts[0]
			targets[TargetLocation{City: normalize(city), CountryCode: normalize(country)}] = true
		} else {
			fmt.Printf("Warning: Malformed location entry in locations.txt: %s\n", line)
		}
	}
	if scanner.Err() != nil {
		fmt.Printf("Error reading locations file: %v\n", scanner.Err())
	}

	csvFile, err := os.Open(os.Args[2])
	if err != nil {
		fmt.Printf("Error opening CSV file: %v\n", err)
		os.Exit(1)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	header, err := reader.Read()
	if err != nil {
		fmt.Printf("Error reading CSV header: %v\n", err)
		os.Exit(1)
	}

	// Map column names to indices for flexibility
	cols := make(map[string]int)
	for i, name := range header {
		cols[name] = i
	}

	// Ensure required columns exist
	requiredCols := []string{"city_ascii", "lat", "lng", "country", "iso2", "iso3", "capital", "admin_name"}
	for _, col := range requiredCols {
		if _, ok := cols[col]; !ok {
			fmt.Printf("Error: Missing required column '%s' in worldcities.csv\n", col)
			os.Exit(1)
		}
	}

	outFile, err := os.Create(os.Args[3])
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	// Write the required header
	writer.Write([]string{"location_id", "city_ascii", "lat", "lng", "country"})

	matchedCount := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Warning: Error reading CSV row: %v\n", err)
			continue // Skip malformed rows
		}

		normalizedCity := normalize(row[cols["city_ascii"]])
		normalizedAdminName := normalize(row[cols["admin_name"]])
		normalizedCapital := normalize(row[cols["capital"]])

		// Collect all possible normalized country identifiers for this CSV row
		csvCountryIdentifiers := make(map[string]bool)
		csvCountryIdentifiers[normalize(row[cols["country"]])] = true
		csvCountryIdentifiers[normalize(row[cols["iso2"]])] = true
		csvCountryIdentifiers[normalize(row[cols["iso3"]])] = true

		// Add aliases for these identifiers
		for id := range csvCountryIdentifiers {
			if alias, ok := countryCodeAliases[id]; ok {
				csvCountryIdentifiers[alias] = true
			}
		}

		// Check if this CSV row matches any of our target locations
		for targetLoc := range targets {
			cityToken := normalizedCity + " " + normalizedCapital + " " + normalizedAdminName
			// if (normalizedCity == targetLoc.City || normalizedCapital == targetLoc.City || normalizedAdminName == targetLoc.City) && csvCountryIdentifiers[targetLoc.CountryCode] {
			if strings.Contains(cityToken, targetLoc.City) && csvCountryIdentifiers[targetLoc.CountryCode] {
				writer.Write([]string{
					targetLoc.City + "-" + targetLoc.CountryCode,
					row[cols["city_ascii"]],
					row[cols["lat"]],
					row[cols["lng"]],
					row[cols["country"]],
				})
				delete(targets, targetLoc) // Remove from targets to avoid duplicate matches and track unmatched
				matchedCount++
				break // Matched this CSV row, move to the next CSV row
			}
		}
	}

	fmt.Printf("Filtering complete. Matched %d locations. Results saved to %s\n", matchedCount, os.Args[3])

	if len(targets) > 0 {
		fmt.Println("\nUnmatched locations from locations.txt:")
		for targetLoc := range targets {
			fmt.Printf("- %s-%s\n", targetLoc.City, targetLoc.CountryCode)
		}
	}
}
