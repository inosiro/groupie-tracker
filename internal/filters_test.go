package internal

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
)

type filterTestCache struct {
	locations LocationIndex
	dates     DateIndex
	coords    map[string][2]float64
}

func (c filterTestCache) Artists(ctx context.Context) ([]Artist, error) {
	return nil, nil
}

func (c filterTestCache) Locations(ctx context.Context) (LocationIndex, error) {
	return c.locations, nil
}

func (c filterTestCache) Dates(ctx context.Context) (DateIndex, error) {
	return c.dates, nil
}

func (c filterTestCache) Lookup(locationKey string) (lng float64, lat float64, ok bool) {
	return c.coords[locationKey][0], c.coords[locationKey][1], true
}

func (c filterTestCache) Store(locationKey string, lng float64, lat float64) {
	c.coords[locationKey] = [2]float64{lng, lat}
}

func TestLocationTokensNormalizesSeparators(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{name: "space", query: "Texas USA", want: []string{"texas", "usa"}},
		{name: "comma space", query: "Texas, USA", want: []string{"texas", "usa"}},
		{name: "hyphen", query: "Texas-USA", want: []string{"texas", "usa"}},
		{name: "comma", query: "Texas,USA", want: []string{"texas", "usa"}},
		{name: "spaced hyphen", query: "Texas - USA", want: []string{"texas", "usa"}},
		{name: "multiple separators", query: "New--York,  USA", want: []string{"new", "york", "usa"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := locationTokens(tt.query)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("locationTokens(%q) = %#v, want %#v", tt.query, got, tt.want)
			}
		})
	}
}

func TestProjectUsesOnlyStandardPackages(t *testing.T) {
	data := readProjectFile(t, "go.mod")
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "require" || strings.HasPrefix(line, "require ") || strings.HasPrefix(line, "require(") {
			t.Fatalf("go.mod contains non-standard module requirements: %q", line)
		}
	}
}

func readProjectFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err == nil {
		return data
	}

	data, err = os.ReadFile("../" + path)
	if err != nil {
		t.Fatalf("failed to read project file %q: %v", path, err)
	}
	return data
}

func TestFilterUIIncludesRangeFilter(t *testing.T) {
	data := readProjectFile(t, "templates/index.html")

	if !strings.Contains(string(data), `type="range"`) {
		t.Fatal("expected at least one range filter input")
	}
}

func TestFilterUIIncludesCheckboxFilter(t *testing.T) {
	data := readProjectFile(t, "templates/index.html")

	if !strings.Contains(string(data), `type="checkbox"`) {
		t.Fatal("expected at least one checkbox filter input")
	}
}

func TestCreationDateFilterIncludesExpectedArtists(t *testing.T) {
	result := applyAuditFilter(FilterVM{
		CreationFrom: 1995,
		CreationTo:   2000,
	})

	assertIncludesArtists(t, result,
		"SOJA",
		"Mamonas Assassinas",
		"Thirty Seconds to Mars",
		"Nickelback",
		"NWA",
		"Gorillaz",
		"Linkin Park",
		"Eminem",
		"Coldplay",
	)
}

func TestFirstAlbumFilterIncludesExpectedArtists(t *testing.T) {
	result := applyAuditFilter(FilterVM{
		FirstAlbumFrom: 1990,
		FirstAlbumTo:   1992,
	})

	assertIncludesArtists(t, result,
		"Pearl Jam",
		"Red Hot Chili Peppers",
	)
}

func TestMembersCheckboxFilterIncludesExpectedArtists(t *testing.T) {
	result := applyAuditFilter(FilterVM{
		Members: []int{6},
	})

	assertIncludesArtists(t, result,
		"Pink Floyd",
		"Arctic Monkeys",
		"Linkin Park",
		"Foo Fighters",
	)
}

func TestConcertLocationFilterIncludesExpectedArtists(t *testing.T) {
	result := applyAuditFilter(FilterVM{
		Location: "Texas, USA",
	})

	assertIncludesArtists(t, result,
		"R3HAB",
		"Logic",
		"Joyner Lucas",
		"Twenty One Pilots",
	)
}

func TestCreationDateAndSingleMemberFiltersIncludeExpectedArtists(t *testing.T) {
	result := applyAuditFilter(FilterVM{
		CreationFrom: 1970,
		CreationTo:   2000,
		Members:      []int{1},
	})

	assertIncludesArtists(t, result,
		"Bobby McFerrins",
		"Eminem",
	)
}

func TestCreationDateAfter2010AndFirstAlbumAfter2010IncludeExpectedArtists(t *testing.T) {
	result := applyAuditFilter(FilterVM{
		CreationFrom:   2011,
		FirstAlbumFrom: 2011,
	})

	assertIncludesArtists(t, result,
		"XXXTentacion",
		"Juice Wrld",
		"Alec Benjamin",
		"Post Malone",
	)
}

func TestWashingtonConcertsAndMoreThanThreeMembersIncludeExpectedArtists(t *testing.T) {
	result := applyAuditFilter(FilterVM{
		Location:   "Washington, USA",
		MembersMin: 4,
	})

	assertIncludesArtists(t, result,
		"The Rolling Stones",
	)
}

func TestFirstAlbumRangeAndMaximumFourMembersIncludeExpectedArtists(t *testing.T) {
	result := applyAuditFilter(FilterVM{
		FirstAlbumFrom: 1980,
		FirstAlbumTo:   1990,
		MembersMax:     4,
	})

	assertIncludesArtists(t, result,
		"Phil Collins",
		"Bobby McFerrins",
		"Red Hot Chili Peppers",
		"Metallica",
	)
}

func TestResetFiltersShowsAllArtists(t *testing.T) {
	artists := auditArtists()
	result := applyAuditFilter(FilterVM{})

	if len(result) != len(artists) {
		t.Fatalf("reset filters returned %d artists, want %d", len(result), len(artists))
	}
	assertIncludesArtists(t, result, artistNames(artists)...)
}

func applyAuditFilter(filter FilterVM) []Artist {
	artists := auditArtists()
	return ApplyFilters(context.Background(), auditCache(), artists, filter)
}

func auditCache() filterTestCache {
	return filterTestCache{
		locations: LocationIndex{Index: []Locations{
			{ID: 18, Locations: []string{"texas-usa"}},
			{ID: 19, Locations: []string{"austin-texas-usa"}},
			{ID: 20, Locations: []string{"dallas-texas-usa"}},
			{ID: 21, Locations: []string{"houston-texas-usa"}},
			{ID: 22, Locations: []string{"washington-usa"}},
		}},
	}
}

func auditArtists() []Artist {
	return []Artist{
		artistFixture(1, "SOJA", 1997, "01-01-2002", 5),
		artistFixture(2, "Mamonas Assassinas", 1995, "01-01-1995", 5),
		artistFixture(3, "Thirty Seconds to Mars", 1998, "01-01-2002", 4),
		artistFixture(4, "Nickelback", 1995, "01-01-1996", 4),
		artistFixture(5, "NWA", 1995, "01-01-1988", 5),
		artistFixture(6, "Gorillaz", 1998, "01-01-2001", 4),
		artistFixture(7, "Linkin Park", 1996, "01-01-2000", 6),
		artistFixture(8, "Eminem", 1996, "01-01-1999", 1),
		artistFixture(9, "Coldplay", 1996, "01-01-2000", 4),
		artistFixture(10, "Pearl Jam", 1990, "01-01-1991", 5),
		artistFixture(11, "Red Hot Chili Peppers", 1983, "01-01-1990", 4),
		artistFixture(12, "Pink Floyd", 1965, "01-01-1967", 6),
		artistFixture(13, "Arctic Monkeys", 2002, "01-01-2006", 6),
		artistFixture(14, "Foo Fighters", 1994, "01-01-1995", 6),
		artistFixture(15, "Bobby McFerrins", 1977, "01-01-1982", 1),
		artistFixture(16, "XXXTentacion", 2013, "01-01-2017", 1),
		artistFixture(17, "Juice Wrld", 2015, "01-01-2018", 1),
		artistFixture(18, "R3HAB", 2007, "01-01-2014", 1),
		artistFixture(19, "Logic", 2009, "01-01-2014", 1),
		artistFixture(20, "Joyner Lucas", 2007, "01-01-2015", 1),
		artistFixture(21, "Twenty One Pilots", 2009, "01-01-2009", 2),
		artistFixture(22, "The Rolling Stones", 1962, "01-01-1964", 5),
		artistFixture(23, "Alec Benjamin", 2013, "01-01-2018", 1),
		artistFixture(24, "Post Malone", 2011, "01-01-2016", 1),
		artistFixture(25, "Phil Collins", 1981, "01-01-1981", 1),
		artistFixture(26, "Metallica", 1981, "01-01-1983", 4),
		artistFixture(27, "Out Of Range Band", 1960, "01-01-1970", 2),
	}
}

func artistFixture(id int, name string, creationDate int, firstAlbum string, memberCount int) Artist {
	members := make([]string, memberCount)
	for i := range members {
		members[i] = name + " member"
	}
	return Artist{
		ID:           id,
		Name:         name,
		CreationDate: creationDate,
		FirstAlbum:   firstAlbum,
		Members:      members,
	}
}

func assertIncludesArtists(t *testing.T, artists []Artist, expectedNames ...string) {
	t.Helper()

	got := make(map[string]bool, len(artists))
	for _, artist := range artists {
		got[artist.Name] = true
	}

	for _, name := range expectedNames {
		if !got[name] {
			t.Fatalf("expected results to include %q; got %v", name, artistNames(artists))
		}
	}
}

func artistNames(artists []Artist) []string {
	names := make([]string, 0, len(artists))
	for _, artist := range artists {
		names = append(names, artist.Name)
	}
	return names
}

func TestMatchesLocationTokensRequiresAllTokens(t *testing.T) {
	locations := []string{"new_york-usa", "texas-usa"}

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{name: "all tokens match", query: "New-York, USA", want: true},
		{name: "case insensitive", query: "TEXAS, usa", want: true},
		{name: "missing token fails", query: "Texas Canada", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesLocationTokens(locations, tt.query)
			if got != tt.want {
				t.Fatalf("matchesLocationTokens(%#v, %q) = %t, want %t", locations, tt.query, got, tt.want)
			}
		})
	}
}
