package internal

import (
	"net/http"
	"os"
	"text/template"
)

// ═══════════════════════════════════════════════════════════════════════════
// TEMPLATE RENDERING ENGINE
// ═══════════════════════════════════════════════════════════════════════════
// Responsible for parsing and caching HTML templates, then rendering them
// with data. Templates are cached at init() time for performance.
//
// Architecture:
//   - Templates are stored in templates/ directory
//   - Cached in global variables at server startup
//   - RenderPage() returns full pages (with layout)
//   - RenderPartial() returns fragments only (for HTMX swaps)

var (
	// layoutTemplate is the base HTML page structure
	// Used by all page renders
	layoutTemplate *template.Template

	// indexTemplate combines layout + index page content
	// Used for: GET / (home page)
	indexTemplate *template.Template

	// Includes: artist_grid, artist_card, empty_state, error_banner partials
	// Used for: GET /artists (with HX-Request=true for HTMX fragments)
	gridTemplate *template.Template

	// detailsTemplate renders concert details for a single artist
	// Used for: GET /artists/{id} (concert details fragment)
	detailsTemplate *template.Template

	// locationsTemplate renders concert locations for a single artist
	locationsTemplate *template.Template

	// datesTemplate renders concert dates for a single artist
	datesTemplate *template.Template

	// membersTemplate renders artist members
	membersTemplate *template.Template
)

// init() is called once at server startup to parse and cache templates
// This avoids re-parsing templates on every request (performance critical)
func init() {
	// Try to find templates - check both relative and parent directory
	templateDir := "templates"
	if _, err := os.Stat(templateDir); err != nil {
		// If templates not found in current dir, try parent
		if _, err := os.Stat("../" + templateDir); err == nil {
			templateDir = "../" + templateDir
		} else {
			// Templates not found - tests may not need them
			return
		}
	}

	// ─────────────────────────────────────────────────────────────────────────
	// Parse layout template
	// This is the base HTML document structure (DOCTYPE, head, body wrapper)
	// ─────────────────────────────────────────────────────────────────────────
	layoutTemplate = template.Must(template.ParseFiles(
		templateDir + "/layout.html",
	))

	// ─────────────────────────────────────────────────────────────────────────
	// Parse index page template
	// Combines layout + index content (search/filter form + grid container)
	// Template hierarchy: layout defines {{template "content" .}}
	//                     index content is rendered into that slot
	// ─────────────────────────────────────────────────────────────────────────
	indexTemplate = template.Must(template.ParseFiles(
		templateDir+"/layout.html",
		templateDir+"/index.html",
		templateDir+"/partials/artist_grid.html",
		templateDir+"/partials/artist_card.html",
		templateDir+"/partials/empty_state.html",
		templateDir+"/partials/error_banner.html",
		templateDir+"/partials/artist_members.html",
		templateDir+"/partials/artist_details.html",
		templateDir+"/partials/artist_locations.html",
		templateDir+"/partials/artist_dates.html",
	))

	// We can reuse the indexTemplate for partials if we want to be truly DRY,
	// as it already contains all necessary partial definitions.
	gridTemplate = indexTemplate

	// ─────────────────────────────────────────────────────────────────────────
	// Used for displaying concert dates grouped by location
	// ─────────────────────────────────────────────────────────────────────────
	detailsTemplate = template.Must(template.ParseFiles(
		templateDir + "/partials/artist_details.html",
	))

	// ─────────────────────────────────────────────────────────────────────────
	// Parse locations partial template
	// ─────────────────────────────────────────────────────────────────────────
	locationsTemplate = template.Must(template.ParseFiles(
		templateDir + "/partials/artist_locations.html",
	))

	// ─────────────────────────────────────────────────────────────────────────
	// Parse dates partial template
	// ─────────────────────────────────────────────────────────────────────────
	datesTemplate = template.Must(template.ParseFiles(
		templateDir + "/partials/artist_dates.html",
	))

	// Parse members partial template
	membersTemplate = template.Must(template.ParseFiles(
		templateDir + "/partials/artist_members.html",
	))
}

// RenderPage renders a full HTML page with layout
// Used for: GET / (returns full page structure)
//
// Parameters:
//   - w: http.ResponseWriter (writes to HTTP response)
//   - pageName: which page to render (currently only "index" supported)
//   - data: view model to pass into template (GridVM, nil, etc.)
//
// Process:
//  1. Set Content-Type header to HTML
//  2. Execute the page template with layout wrapper
//  3. Template system renders data into HTML
//  4. Write to response writer
//
// Note: layout.html defines the {{template "content" .}} slot
//
//	which is filled with page-specific content
func RenderPage(w http.ResponseWriter, pageName string, data interface{}) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch pageName {
	case "index":
		// Render: layout.html with index.html nested inside
		// This produces a complete HTML page
		return indexTemplate.ExecuteTemplate(w, "layout", data)
	default:
		// Fallback to index
		return indexTemplate.ExecuteTemplate(w, "layout", data)
	}
}

// RenderPartial renders a partial HTML template (no layout wrapper)
// Used for: GET /artists (HTMX fragments), GET /artists/{id} (concert details)
//
// Parameters:
//   - w: http.ResponseWriter (writes to HTTP response)
//   - partialName: which partial to render (artist_grid, artist_details)
//   - data: view model to pass into template (GridVM, DetailsVM)
//
// Process:
//  1. Set Content-Type header to HTML
//  2. Execute the partial template directly (no layout)
//  3. HTMX on the browser will swap this fragment into a container
//
// Why separate RenderPartial?
//   - HTMX requests should NOT include layout (DOCTYPE, head, body tags)
//   - Only the content fragment is returned and swapped
//   - Keeps template logic clean and explicit
//
// HTMX Pattern Example:
//
//	Button: <button hx-get="/artists/42" hx-target="#details-42" hx-swap="innerHTML">
//	Server returns: <div class="artist-details">...</div> (fragment only)
//	HTMX swaps into: <div id="details-42"> (the target)
func RenderPartial(w http.ResponseWriter, partialName string, data interface{}) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	switch partialName {
	case "artist_grid":
		// Render the artist grid partial (used by GET /artists)
		// Includes conditional logic for error, empty state, or results
		return gridTemplate.ExecuteTemplate(w, "artist_grid", data)

	case "error_banner":
		// Render the standalone error banner partial
		return gridTemplate.ExecuteTemplate(w, "error_banner", data)

	case "empty_state":
		// Render the standalone empty state partial
		return gridTemplate.ExecuteTemplate(w, "empty_state", data)

	case "artist_details":
		// Render the artist details partial (used by GET /artists/{id})
		// Shows members list + concerts by location
		return detailsTemplate.ExecuteTemplate(w, "artist_details", data)

	case "artist_locations":
		// Render the artist locations partial
		return locationsTemplate.ExecuteTemplate(w, "artist_locations", data)

	case "artist_dates":
		// Render the artist dates partial
		return datesTemplate.ExecuteTemplate(w, "artist_dates", data)

	case "artist_members":
		// Render the artist members partial
		return membersTemplate.ExecuteTemplate(w, "artist_members", data)

	default:
		// Fallback to grid
		return gridTemplate.ExecuteTemplate(w, "artist_grid", data)
	}
}
