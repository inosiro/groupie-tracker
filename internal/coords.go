package internal

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
)

type CoordResolver interface {
	Lookup(locationKey string) (lng float64, lat float64, ok bool)
	Store(locationKey string, lng float64, lat float64)
}

type FileCoordResolver struct {
	mu     sync.Mutex
	coords map[string][2]float64
}

// for missing locations, fetch it from nominatim API
func NewFileCoordResolver() *FileCoordResolver {
	csvFile, err := os.Open("./scripts/output.csv")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening CSV file: %v\n", err)
		os.Exit(1)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	header, err := reader.Read()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading csv header: %v\n", err)
		os.Exit(1)
	}
	cols := make(map[string]int)
	for i, name := range header {
		cols[name] = i
	}
	fcr := &FileCoordResolver{
		coords: map[string][2]float64{},
	}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Warning: Error reading CSV row: %v\n", err)
			continue // Skip malformed rows
		}
		lng, err := strconv.ParseFloat(row[cols["lng"]], 64)
		if err != nil {
			fmt.Printf("Error parsing lng: %v\n", err)
			continue
		}
		lat, err := strconv.ParseFloat(row[cols["lat"]], 64)
		if err != nil {
			fmt.Printf("Error parsing lat: %v\n", err)
			continue
		}

		fcr.coords[row[cols["location_id"]]] = [2]float64{lng, lat}
	}
	return fcr
}

func (r *FileCoordResolver) Lookup(locationKey string) (lng float64, lat float64, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	coords, exist := r.coords[locationKey]
	if !exist {
		//fetch the location from nominatim
		// if cannot fetch
		return 0, 0, false
	}
	return coords[0], coords[1], true
}

func (r *FileCoordResolver) Store(locationKey string, lng float64, lat float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.coords[locationKey] = [2]float64{lng, lat}
}
