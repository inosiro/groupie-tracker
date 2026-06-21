package internal

type CoordResolver interface {
	Lookup(locationKey string) (lng float64, lat float64, ok bool)
	Store(locationKey string, lng float64, lat float64)
}
