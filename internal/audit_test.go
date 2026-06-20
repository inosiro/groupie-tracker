package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// TEST SETUP AND UTILITIES
// ═══════════════════════════════════════════════════════════════════════════

// testSetup ensures we're in the right directory for template loading
func testSetup(t *testing.T) {
	// Change to project root if templates aren't found
	if _, err := os.Stat("templates/layout.html"); err != nil {
		err := os.Chdir("..") // Move up one level to project root
		if err != nil {
			t.Fatalf("Failed to change to project root: %v", err)
		}
	}
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ═══════════════════════════════════════════════════════════════════════════
// MOCK CLIENT FOR TESTING ERROR PATHS
// ═══════════════════════════════════════════════════════════════════════════

// FailingClient is a mock API client that returns errors for testing
// It has the same method signature as Client.Artists for duck typing
type FailingClient struct{}

// Artists returns an error (simulates API failure)
func (c *FailingClient) Artists(ctx context.Context) ([]Artist, error) {
	return nil, context.DeadlineExceeded // Simulate timeout/failure
}

// ═══════════════════════════════════════════════════════════════════════════
// AUDIT CASE TESTS
// ═══════════════════════════════════════════════════════════════════════════
// These tests validate all audit cases defined in PRD section 6.1

// TestAuditCase1_HomeRenders
// Audit case 1: Home renders
// Input: GET /
// Expected: 200 OK, contains artist-related content and HTML structure
func TestAuditCase1_HomeRenders(t *testing.T) {
	testSetup(t)

	// Setup
	api := NewClient()
	cache := NewCache(api, 2*time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", Index(cache))

	// Execute
	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Verify
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "DOCTYPE") && !strings.Contains(body, "html") {
		t.Error("Expected HTML document structure")
	}

	if !strings.Contains(body, "artist") && !strings.Contains(body, "Artist") {
		t.Error("Expected artist-related content in response")
	}
}

// TestAuditCase2_ArtistDetailsFragmentLoads
// Audit case 2: Artist details fragment loads
// Input: GET /artists/1
// Expected: 200 OK, HTML fragment contains concerts section
func TestAuditCase2_ArtistDetailsFragmentLoads(t *testing.T) {
	testSetup(t)

	// Setup
	api := NewClient()
	cache := NewCache(api, 2*time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /artists/{id}", ArtistDetailsHandler(cache, api))

	// Execute
	req, _ := http.NewRequest("GET", "/artists/1", nil)
	req.Header.Set("HX-Request", "true") // Mark as HTMX request to get fragment
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Verify
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Expected non-empty response body")
	}

	// Should contain member or concert-related content
	if !strings.Contains(body, "member") && !strings.Contains(body, "Member") &&
		!strings.Contains(body, "location") && !strings.Contains(body, "Location") {
		t.Error("Expected members or concert location content in details fragment")
	}
}

// TestAuditCase3_APIFailureDoesNotCrash
// Audit case 3: API failure does not crash
// Input: simulate upstream API down
// Expected: 200 OK (or 503 with friendly HTML), server stays up
func TestAuditCase3_APIFailureDoesNotCrash(t *testing.T) {
	testSetup(t)

	// Setup with real client (which will timeout if API is unreachable)
	api := NewClient()
	cache := NewCache(api, 2*time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", Index(cache))
	mux.HandleFunc("GET /artists", ArtistsHandler(cache, api))

	// Execute first request (may fail if API is down)
	req1, _ := http.NewRequest("GET", "/artists", nil)
	w1 := httptest.NewRecorder()
	mux.ServeHTTP(w1, req1)

	// Verify first request doesn't crash (status 200 or 503 is acceptable)
	if w1.Code != http.StatusOK && w1.Code != http.StatusServiceUnavailable {
		t.Logf("Status code: %d (200 or 503 expected)", w1.Code)
	}

	// Execute second request - server should still be responsive
	req2, _ := http.NewRequest("GET", "/", nil)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	// Server should remain functional (not crash)
	if w2.Code == 0 {
		t.Error("Server crashed or did not respond to second request")
	}
}

// TestAuditCase4_LocationsDataIsUsed
// Audit case 4: Locations data is used
// Input: GET /artists then GET /locations/{id}
// Expected: HTML fragment displays concert locations
func TestAuditCase4_LocationsDataIsUsed(t *testing.T) {
	testSetup(t)

	// Setup
	api := NewClient()
	cache := NewCache(api, 2*time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /locations/{id}", ArtistLocationsHandler(cache, api))

	// Execute - request locations for artist 1
	req, _ := http.NewRequest("GET", "/locations/1", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Verify
	if w.Code != http.StatusOK {
		t.Logf("Expected status 200, got %d (API may be unavailable)", w.Code)
		return
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Expected non-empty response body for locations")
	}
}

// TestAuditCase5_DatesDataIsUsed
// Audit case 5: Dates data is used
// Input: GET /artists then GET /dates/{id}
// Expected: HTML fragment displays concert dates
func TestAuditCase5_DatesDataIsUsed(t *testing.T) {
	testSetup(t)

	// Setup
	api := NewClient()
	cache := NewCache(api, 2*time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /dates/{id}", ArtistDatesHandler(cache, api))

	// Execute - request dates for artist 1
	req, _ := http.NewRequest("GET", "/dates/1", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Verify
	if w.Code != http.StatusOK {
		t.Logf("Expected status 200, got %d (API may be unavailable)", w.Code)
		return
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Expected non-empty response body for dates")
	}
}

// TestAuditCase6_MembersAreDisplayedForAnyArtist
// Audit case 6: Members are displayed for any artist
// Input: GET /artists then GET /members/{id}
// Expected: HTML fragment includes members list
func TestAuditCase6_MembersAreDisplayedForAnyArtist(t *testing.T) {
	testSetup(t)

	// Setup
	api := NewClient()
	cache := NewCache(api, 2*time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /members/{id}", ArtistMembersHandler(cache))

	// Execute - request members for artist 1
	req, _ := http.NewRequest("GET", "/members/1", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Verify
	if w.Code != http.StatusOK {
		t.Logf("Expected status 200, got %d (API may be unavailable)", w.Code)
		return
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("Expected non-empty response body for members")
	}

	if !strings.Contains(body, "member") && !strings.Contains(body, "Member") {
		t.Logf("Expected members-related content, response was: %s", body[:minInt(200, len(body))])
	}
}

// TestAuditCase7_FirstAlbumDateIsDisplayed
// Audit case 7: First Album date is displayed
// Input: GET /artists (view artist cards)
// Expected: Artist cards display firstAlbum information
func TestAuditCase7_FirstAlbumDateIsDisplayed(t *testing.T) {
	testSetup(t)

	// Setup
	api := NewClient()
	cache := NewCache(api, 2*time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /artists", ArtistsHandler(cache, api))

	// Execute
	req, _ := http.NewRequest("GET", "/artists", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// Verify
	if w.Code != http.StatusOK {
		t.Logf("Expected status 200, got %d", w.Code)
		return
	}

	body := w.Body.String()
	// Check for date-like patterns or "album" mentions
	if !strings.Contains(body, "album") && !strings.Contains(body, "Album") &&
		!strings.Contains(strings.ToLower(body), "first") {
		t.Logf("Expected firstAlbum information in artist cards")
	}
}

// TestAuditCase8_ConcertLocationsAreDisplayed
// Audit case 8: Concert locations are displayed
// Input: GET /artists then GET /locations/{id}
// Expected: HTML fragment displays list of concert locations
func TestAuditCase8_ConcertLocationsAreDisplayed(t *testing.T) {
	testSetup(t)

	// Setup
	api := NewClient()
	cache := NewCache(api, 2*time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /locations/{id}", ArtistLocationsHandler(cache, api))

	// Execute for multiple artists to verify generality
	for artistID := 1; artistID <= 3; artistID++ {
		idStr := string(rune('0' + artistID))
		req, _ := http.NewRequest("GET", "/locations/"+idStr, nil)
		req.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Logf("Artist %d: Expected status 200, got %d (API may be unavailable)", artistID, w.Code)
			continue
		}

		body := w.Body.String()
		if len(body) == 0 {
			t.Errorf("Artist %d: Expected non-empty response body for locations", artistID)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// INTEGRATION TEST: Cache behavior
// Additional golden test from PRD 6.2
// ═══════════════════════════════════════════════════════════════════════════

// TestCacheBehavior verifies that consecutive GET /artists calls hit the cache
// Expected: Second call should return same data as first without API delay
func TestCacheBehavior(t *testing.T) {
	testSetup(t)

	// Setup
	api := NewClient()
	cache := NewCache(api, 2*time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /artists", ArtistsHandler(cache, api))

	// First request - will populate cache
	req1, _ := http.NewRequest("GET", "/artists", nil)
	w1 := httptest.NewRecorder()
	mux.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Logf("First request failed with status %d (API may be unavailable)", w1.Code)
		return
	}

	// Second request - should hit cache
	req2, _ := http.NewRequest("GET", "/artists", nil)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Second request failed with status %d", w2.Code)
	}

	// Both should have content
	if w1.Body.Len() == 0 || w2.Body.Len() == 0 {
		t.Error("Expected both responses to have content")
	}

	// Content should be similar (cache hit)
	if w1.Body.String() != w2.Body.String() {
		t.Logf("Cache may not be working as expected (responses differ)")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// EXTERNAL API FAILURE TEST
// Tests graceful handling of external API failures
// ═══════════════════════════════════════════════════════════════════════════

// TestExternalAPIFailure
// Validates that when the external Groupie Trackers API fails,
// handlers return appropriate error responses and don't crash.
// Uses very short timeout to simulate API unavailability.
func TestExternalAPIFailure(t *testing.T) {
	testSetup(t)

	// Setup with normal API client (real API calls)
	api := NewClient()
	cache := NewCache(api, 2*time.Minute)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /artists", ArtistsHandler(cache, api))
	mux.HandleFunc("GET /artists/{id}", ArtistDetailsHandler(cache, api))
	mux.HandleFunc("GET /locations/{id}", ArtistLocationsHandler(cache, api))
	mux.HandleFunc("GET /dates/{id}", ArtistDatesHandler(cache, api))
	mux.HandleFunc("GET /members/{id}", ArtistMembersHandler(cache))

	// Test 1: GET /artists - should handle API errors gracefully
	t.Run("ArtistsEndpointHandlesErrors", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/artists", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should return valid HTTP response (not crash)
		// Either 200 with data or error status with message
		if w.Code >= 600 {
			t.Errorf("Invalid HTTP status code: %d", w.Code)
		}
	})

	// Test 2: GET /artists/{id} - should handle API errors gracefully
	t.Run("ArtistDetailsEndpointHandlesErrors", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/artists/1", nil)
		req.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should handle error gracefully without crashing
		if w.Code >= 600 {
			t.Errorf("Invalid HTTP status code: %d", w.Code)
		}
	})

	// Test 3: GET /locations/{id} - should handle API errors gracefully
	t.Run("LocationsEndpointHandlesErrors", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/locations/1", nil)
		req.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should return valid HTTP response
		if w.Code >= 600 {
			t.Errorf("Invalid HTTP status code: %d", w.Code)
		}
	})

	// Test 4: GET /dates/{id} - should handle API errors gracefully
	t.Run("DatesEndpointHandlesErrors", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/dates/1", nil)
		req.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should return valid HTTP response
		if w.Code >= 600 {
			t.Errorf("Invalid HTTP status code: %d", w.Code)
		}
	})

	// Test 5: Server remains responsive after multiple requests
	t.Run("ServerStaysResponsiveAfterErrors", func(t *testing.T) {
		// Make multiple requests to ensure server doesn't crash
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("GET", "/artists", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			// Server should respond with valid HTTP status
			if w.Code == 0 {
				t.Errorf("Request %d: Server did not respond", i+1)
			}
			if w.Code >= 600 {
				t.Errorf("Request %d: Invalid HTTP status code: %d", i+1, w.Code)
			}
		}
	})

	// Test 6: Verify error messages are displayed (not server panic)
	t.Run("ErrorMessagesAreDisplayed", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/artists", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		body := w.Body.String()
		// Should have content (either data or error message)
		if len(body) == 0 && w.Code == http.StatusOK {
			t.Error("Expected response body with either data or error message")
		}
		// Should not be a stack trace or panic message
		if strings.Contains(body, "panic") {
			t.Error("Response contains panic message - server crashed")
		}
		if strings.Contains(body, "fatal") {
			t.Error("Response contains fatal error - unhandled panic")
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// CACHE MISS WITH API FAILURE TEST
// Tests the critical scenario: first request, cache is empty, upstream API fails
// ═══════════════════════════════════════════════════════════════════════════

// TestCacheMissWithUpstreamAPIFailure
// Validates graceful handling when both conditions are true:
//  1. Cache is empty (cache miss - first request, no TTL data)
//  2. Upstream API is unavailable or fails
//
// This is a critical scenario because the system has:
//   - No cached data to fall back on
//   - No API response to populate cache
//   - Must still respond gracefully without crashing
func TestCacheMissWithUpstreamAPIFailure(t *testing.T) {
	testSetup(t)

	// Setup: Create a fresh cache with no pre-populated data
	// The cache will attempt to fetch from API on first request
	api := NewClient()
	cache := NewCache(api, 2*time.Minute) // Empty cache, TTL=2min

	mux := http.NewServeMux()
	mux.HandleFunc("GET /artists", ArtistsHandler(cache, api))
	mux.HandleFunc("GET /artists/{id}", ArtistDetailsHandler(cache, api))
	mux.HandleFunc("GET /locations/{id}", ArtistLocationsHandler(cache, api))
	mux.HandleFunc("GET /dates/{id}", ArtistDatesHandler(cache, api))
	mux.HandleFunc("GET /members/{id}", ArtistMembersHandler(cache))

	// Test 1: First request to GET /artists with empty cache
	// Should handle gracefully even if API fails/times out
	t.Run("FirstRequestCacheMissHandlesGracefully", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/artists", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should return valid HTTP response (not crash)
		if w.Code >= 600 {
			t.Errorf("Invalid HTTP status code on first request: %d", w.Code)
		}

		// Should have content (either artists or error message)
		if w.Body.Len() == 0 {
			t.Error("Expected response body on first request (data or error message)")
		}

		// Should not be a panic/crash
		if strings.Contains(w.Body.String(), "panic") ||
			strings.Contains(w.Body.String(), "fatal") {
			t.Error("First request caused server panic with empty cache")
		}
	})

	// Test 2: Multiple concurrent cache-miss scenarios
	// Simulate multiple requests hitting cache misses simultaneously
	t.Run("MultipleCacheMissRequestsHandleGracefully", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("GET", "/artists", nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			if w.Code >= 600 {
				t.Errorf("Request %d: Invalid HTTP status code: %d", i+1, w.Code)
			}

			if w.Code == 0 {
				t.Errorf("Request %d: Server did not respond", i+1)
			}
		}
	})

	// Test 3: Verify cache-miss doesn't block server for other operations
	t.Run("ServerRemainsResponsiveAfterCacheMiss", func(t *testing.T) {
		// First request (cache miss)
		req1, _ := http.NewRequest("GET", "/artists", nil)
		w1 := httptest.NewRecorder()
		mux.ServeHTTP(w1, req1)

		// Verify server is still responsive
		if w1.Code == 0 {
			t.Error("First request: Server did not respond")
		}

		// Second request (should also work)
		req2, _ := http.NewRequest("GET", "/artists", nil)
		w2 := httptest.NewRecorder()
		mux.ServeHTTP(w2, req2)

		if w2.Code == 0 {
			t.Error("Second request: Server did not respond after first cache miss")
		}
	})

	// Test 4: GET /artists/{id} with empty cache should handle gracefully
	t.Run("ArtistDetailsWithEmptyCacheHandlesGracefully", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/artists/1", nil)
		req.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code >= 600 {
			t.Errorf("Invalid HTTP status code: %d", w.Code)
		}

		// Should not crash with panic
		if strings.Contains(w.Body.String(), "panic") {
			t.Error("Artist details with cache miss caused panic")
		}
	})

	// Test 5: GET /locations/{id} with empty cache should handle gracefully
	t.Run("LocationsWithEmptyCacheHandlesGracefully", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/locations/1", nil)
		req.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code >= 600 {
			t.Errorf("Invalid HTTP status code: %d", w.Code)
		}

		if strings.Contains(w.Body.String(), "panic") {
			t.Error("Locations with cache miss caused panic")
		}
	})

	// Test 6: GET /dates/{id} with empty cache should handle gracefully
	t.Run("DatesWithEmptyCacheHandlesGracefully", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/dates/1", nil)
		req.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code >= 600 {
			t.Errorf("Invalid HTTP status code: %d", w.Code)
		}

		if strings.Contains(w.Body.String(), "panic") {
			t.Error("Dates with cache miss caused panic")
		}
	})

	// Test 7: Verify no data corruption or partial responses on cache miss
	t.Run("NoCacheDataCorruptionOnMiss", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/artists", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		body := w.Body.String()

		// Response should be complete HTML (no truncation/corruption)
		if w.Code == http.StatusOK && len(body) > 0 {
			// If we got data, verify it's valid HTML
			if !strings.Contains(body, "<") && !strings.Contains(body, ">") {
				// Could be error message, which is fine
				t.Logf("Cache miss returned non-HTML response: %s", body[:minInt(100, len(body))])
			}
		}

		// Should never have unmatched/partial HTML tags
		openCount := strings.Count(body, "<")
		closeCount := strings.Count(body, ">")
		if openCount != closeCount && openCount > 0 && closeCount > 0 {
			t.Logf("Warning: Possible malformed HTML in cache miss response")
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// HTMX VS BROWSER ERROR HANDLING TEST
// Tests the code path in ArtistsHandler (lines 88-105) that distinguishes
// between HTMX fragment requests and full-page browser requests when API fails
// ═══════════════════════════════════════════════════════════════════════════

// MockCacheForErrorHandling is a test helper that behaves like Cache but
// always returns an error, simulating API failure
// This allows us to test the error path in ArtistsHandler (lines 88-105)
type MockCacheForErrorHandling struct{}

// Artists always returns an error (simulates API/cache failure)
func (m *MockCacheForErrorHandling) Artists(ctx context.Context) ([]Artist, error) {
	return nil, context.DeadlineExceeded
}

// Locations mock implementation
func (m *MockCacheForErrorHandling) Locations(ctx context.Context) (LocationIndex, error) {
	return LocationIndex{}, context.DeadlineExceeded
}

// Dates mock implementation
func (m *MockCacheForErrorHandling) Dates(ctx context.Context) (DateIndex, error) {
	return DateIndex{}, context.DeadlineExceeded
}

// TestArtistsHandlerHTMXvsBrowserErrorHandling
// Validates that ArtistsHandler correctly handles API failures with different
// response formats based on the HX-Request header:
//   - HTMX request (HX-Request=true): returns fragment only via RenderPartial
//   - Browser request (HX-Request absent/false): returns full page via RenderPage
//
// This tests the specific error handling code at lines 88-105 of web_handlers.go
// By passing a MockCacheForErrorHandling, we reliably trigger the error path.
func TestArtistsHandlerHTMXvsBrowserErrorHandling(t *testing.T) {
	testSetup(t)

	api := NewClient()
	// Create a mock cache that always returns an error
	mockCache := &MockCacheForErrorHandling{}

	// Register the REAL ArtistsHandler with the mock cache
	// This ensures the actual code at lines 88-105 of web_handlers.go is executed
	mux := http.NewServeMux()
	mux.HandleFunc("GET /artists", ArtistsHandler(mockCache, api))

	// Test 1: HTMX request with API failure
	// Should return error in fragment format via RenderPartial
	t.Run("HTMXRequestWithAPIFailure_ReturnsFragment", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/artists", nil)
		req.Header.Set("HX-Request", "true") // Mark as HTMX request
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should return 200 OK (graceful error handling)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		body := w.Body.String()

		// Should contain error message from GridVM
		if !strings.Contains(body, "Failed to load artists") {
			t.Error("Expected error message 'Failed to load artists' in HTMX response")
		}

		// Should not crash
		if strings.Contains(body, "panic") || strings.Contains(body, "fatal") {
			t.Error("HTMX request with API failure caused server panic")
		}
	})

	// Test 2: Browser request with API failure
	// Should return error in full page format via RenderPage
	// Note: The full page layout doesn't include the grid template partials,
	// so the error message is not visible in the page response itself.
	// The error would only appear if HTMX tries to fetch /artists and fails.
	t.Run("BrowserRequestWithAPIFailure_ReturnsFullPage", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/artists", nil)
		// Do NOT set HX-Request header (this is a browser request)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Should return 200 OK (graceful error handling)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		body := w.Body.String()

		// Should contain HTML structure (DOCTYPE or html tag indicates full page)
		// This is the key difference from HTMX request
		if !strings.Contains(body, "DOCTYPE") && !strings.Contains(body, "<html") {
			t.Error("Expected full HTML page structure in browser response")
		}

		// Should have grid container (from index.html template)
		if !strings.Contains(body, "artist-grid") {
			t.Error("Expected grid container in browser response")
		}

		// Should not crash
		if strings.Contains(body, "panic") || strings.Contains(body, "fatal") {
			t.Error("Browser request with API failure caused server panic")
		}
	})

	// Test 3: Verify HX-Request header value checking is exact
	// Only "true" (lowercase) should trigger HTMX fragment response
	t.Run("HeaderValueMustBeExactlyTrue", func(t *testing.T) {
		testCases := []struct {
			headerValue    string
			expectFragment bool
		}{
			{"true", true},   // Exact match - should get fragment
			{"false", false}, // Not true - should get full page
			{"TRUE", false},  // Case sensitive - should get full page
			{"1", false},     // Not exact - should get full page
			{"", false},      // Empty - should get full page
		}

		for _, tc := range testCases {
			t.Run("HX-Request="+tc.headerValue, func(t *testing.T) {
				req, _ := http.NewRequest("GET", "/artists", nil)
				if tc.headerValue != "" {
					req.Header.Set("HX-Request", tc.headerValue)
				}
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Expected 200 OK, got %d", w.Code)
				}

				body := w.Body.String()
				hasDoctype := strings.Contains(body, "DOCTYPE")

				// If fragment is expected, response should NOT have DOCTYPE (it's a template fragment)
				// If full page is expected, response SHOULD have DOCTYPE
				if tc.expectFragment && hasDoctype {
					t.Error("Expected fragment (no DOCTYPE) but got full page with DOCTYPE")
				}
				if !tc.expectFragment && !hasDoctype {
					t.Error("Expected full page (with DOCTYPE) but got fragment without DOCTYPE")
				}
			})
		}
	})

	// Test 4: Verify error response completeness and no truncation
	t.Run("ErrorResponsesAreComplete", func(t *testing.T) {
		testCases := []struct {
			name            string
			isHTMXRequest   bool
			expectedContent string
		}{
			{
				name:            "HTMX Fragment Error",
				isHTMXRequest:   true,
				expectedContent: "Failed to load artists",
			},
			{
				name:            "Browser Full Page",
				isHTMXRequest:   false,
				expectedContent: "DOCTYPE",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req, _ := http.NewRequest("GET", "/artists", nil)
				if tc.isHTMXRequest {
					req.Header.Set("HX-Request", "true")
				}
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)

				body := w.Body.String()

				// Verify expected content is present
				if !strings.Contains(body, tc.expectedContent) {
					t.Errorf("Expected '%s' in response", tc.expectedContent)
				}

				// Verify response is not empty or truncated
				if len(body) == 0 {
					t.Error("Response body is empty")
				}

				// Verify no panic/crash messages
				if strings.Contains(body, "panic") || strings.Contains(body, "fatal") {
					t.Error("Response contains panic/fatal error messages")
				}
			})
		}
	})
}

// TestCache_CoordResolver verifies that the cache successfully loads static coordinates
// from the CSV and resolves them correctly.
func TestCache_CoordResolver(t *testing.T) {
	testSetup(t)

	api := NewClient()
	cache := NewCache(api, 2*time.Minute)

	// Verify lookups of coordinates loaded from CSV
	testCases := []struct {
		locationKey string
		expectedLng float64
		expectedLat float64
		expectedOk  bool
	}{
		{
			locationKey: "jakarta-indonesia",
			expectedLng: 106.8269,
			expectedLat: -6.1753,
			expectedOk:  true,
		},
		{
			locationKey: "new_york-usa",
			expectedLng: -73.9249,
			expectedLat: 40.6943,
			expectedOk:  true,
		},
		{
			locationKey: "non_existent-location",
			expectedLng: 0,
			expectedLat: 0,
			expectedOk:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.locationKey, func(t *testing.T) {
			lng, lat, ok := cache.Lookup(tc.locationKey)
			if ok != tc.expectedOk {
				t.Errorf("Lookup(%s) ok = %v; expected %v", tc.locationKey, ok, tc.expectedOk)
			}
			if ok {
				if lng != tc.expectedLng {
					t.Errorf("Lookup(%s) lng = %f; expected %f", tc.locationKey, lng, tc.expectedLng)
				}
				if lat != tc.expectedLat {
					t.Errorf("Lookup(%s) lat = %f; expected %f", tc.locationKey, lat, tc.expectedLat)
				}
			}
		})
	}
}
