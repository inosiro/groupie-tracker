# Template Documentation

## Overview
This document describes all template files in the `templates/` directory and their roles in the HTMX-driven event architecture.

---

## Core Templates

### `layout.html` - Base Page Layout

**Purpose:** Wrapper for all full-page responses. Defines the document structure: DOCTYPE, meta tags, scripts, CSS.

**Key Responsibilities:**
- Defines the base HTML5 structure
- Includes meta tags for character encoding and viewport scaling
- Loads `htmx.min.js` script (deferred, non-blocking)
- Loads `styles.css` stylesheet
- Provides injection point for page content

**Template System:**
- `{{define "layout"}}` - defines the base layout
- `{{template "content" .}}` - injection point for page-specific content
- Page templates (e.g., `index.html`) define the "content" template

**HTMX Integration:**
- htmx.min.js is included once on page init
- All subsequent interactions use HTMX to swap partial HTML
- No custom JavaScript needed

**Data Flow:**
```
User → Browser → GET / → Server returns: layout + index content
```

---

### `index.html` - Home Page Content

**Purpose:** Provides the main page shell with search and filter form, results container, and HTMX configuration.

**Key Responsibilities:**
- Renders the header ("Bands & Artists")
- Provides the filters form with all search/filter criteria
- Contains the results container that HTMX populates
- Configures HTMX triggers for various input interactions

**Data Flow:**
1. Page loads: layout + this content
2. On load: HTMX `#artist-grid` triggers `GET /artists` (fetches initial list)
3. User searches/filters: form `hx-get` updates `#artist-grid`
4. Server returns: HTML fragment (`artist_grid.html`)
5. HTMX swaps into `#artist-grid` div

**HTMX Attributes:**
- `hx-get="/artists"` - HTTP GET endpoint
- `hx-target="#artist-grid"` - CSS selector where response is inserted
- `hx-swap="innerHTML"` - replace contents of target div
- `hx-push-url="true"` - update browser URL bar with filter state
- `hx-trigger` - multiple triggers for different inputs:
  - `submit` - when user clicks "Apply" button
  - `change delay:150ms` - when user selects sort dropdown
  - `keyup changed delay:300ms/400ms` - search/filter inputs (debounced)

**Filter Parameters:**
- `q` - free-text search (band name or member names)
- `members_min`, `members_max` - member count range
- `creation_from`, `creation_to` - artist creation year range
- `sort` - sort by: relevance, name, creation year, or members

---

## Partial Templates

### `partials/artist_card.html` - Artist Card

**Purpose:** Renders a single artist card within the grid. Included by `artist_grid.html` in a `{{range}}` loop.

**Card Content:**
- Artist image (album cover)
- Artist/band name
- Member count
- Creation year
- First album date
- "Show Concerts" button (interactive)

**HTMX Interaction:**
When user clicks "Show Concerts" button:
- Button: `hx-get="/artists/{{.ID}}"`
- Target: `#details-{{.ID}}` (the container below)
- Action: server returns HTML fragment (`artist_details.html`)
- Result: HTMX swaps fragment into the details div

**Data from GridVM.Artists[]:**
- `.ID` - int (used in HTMX button and div id)
- `.Name` - string (displayed)
- `.Image` - string (URL for img src)
- `.MembersCount` - int (displayed)
- `.CreationDate` - int (displayed)
- `.FirstAlbum` - string (displayed)
- `.Members` - []string (passed to details)

**Details Container:**
- Initially empty div with id `#details-{{.ID}}`
- HTMX swaps concert details fragment into this div on button click

---

### `partials/artist_details.html` - Artist Details & Concerts

**Purpose:** Renders detailed concert information for a single artist. Displayed when user clicks "Show Concerts" button in artist card.

**Data Flow:**
1. User clicks button in `artist_card.html`
2. Button: `hx-get="/artists/{ID}"` → `GET /artists/42`
3. Handler: `ArtistDetailsHandler`
4. Fetches: artist from cache + relation from API
5. Returns: this fragment (`artist_details.html`)
6. HTMX: swaps into `#details-42` div in the card
7. User sees: member list + concerts grouped by location

**Rendered Sections:**

**Artist Header & Members:**
- Artist name (h2)
- Member count
- Members list (ul with each member as li)

**Concerts Section:**
- If no concerts: "No concert information available"
- If concerts exist: grouped by location with dates listed
- Each concert location shows dates as a list

**Data from DetailsVM:**
- `.Artist` - ArtistCardVM (name, image, members, dates)
- `.Concerts` - []LocationConcertVM (location→dates mapping)
  - Each item: `{Location: "City", Dates: ["01-01-2018", "05-02-2018"]}`

---

### `partials/artist_grid.html` - Artist Grid Container

**Purpose:** Renders the filtered artist list. Returned by `GET /artists` with or without filters.

**Render Logic:**
- If API error: show error banner
- Else if no results: show empty state
- Else: show result count + grid of artist cards

**Data from Handler:**
- `.Artists` - []ArtistCardVM (filtered list, 0-52 items)
- `.Count` - int (length of results)
- `.HasError` - bool (true if API call failed)
- `.Error` - string (error message, if HasError=true)
- `.Query` - FilterVM (active filters for UI feedback)

**States:**
1. **Error State:** If upstream API call failed
   - Displays error banner with error message
2. **Empty State:** If filters matched nothing
   - Shows "No artists found" message
3. **Success State:** Display results
   - Shows result count header
   - Renders grid of artist cards (loops through .Artists)

---

### `partials/empty_state.html` - Empty Results State

**Purpose:** Displayed when search/filter criteria match no artists.

**Trigger Scenarios:**
- Search "xyz" with no matching artists
- Filter: `members_min=100` (no band has that many members)
- Filter: `creation_from=2025` (bands from future don't exist yet)

**Message:** "No artists found. Try adjusting your filters."

**User Action:** Adjust filters or go back to view all artists

---

### `partials/error_banner.html` - Error Display

**Purpose:** Displayed when the external Groupie Trackers API call fails. Satisfies FR6: "must not crash; handle errors gracefully"

**Trigger Scenarios:**
- External API `/artists` endpoint is down
- Network timeout during API call
- Invalid API response format

**Handler Behavior:**
- `ArtistsHandler` catches API error
- Creates `GridVM` with `HasError=true`
- Returns this banner instead of grid
- Server continues to work for subsequent requests

**Message:** Friendly error message (no technical details)

---

## HTMX Architecture Summary

The templates work together in an event-driven architecture:

```
Index Form Input
    ↓
    HTMX triggers GET /artists?q=...&members_min=...
    ↓
    Server filters artists
    ↓
    Returns artist_grid.html fragment (with artist_card partials)
    ↓
    HTMX swaps into #artist-grid
    ↓
User sees updated results

    ↓ [User clicks "Show Concerts" button]
    ↓
    HTMX triggers GET /artists/{id}
    ↓
    Server fetches artist details + concerts
    ↓
    Returns artist_details.html fragment
    ↓
    HTMX swaps into #details-{id}
    ↓
User sees concert details
```

**Key Benefits:**
- No custom JavaScript needed
- Entire page state managed via URL parameters (hx-push-url)
- Back button works for navigation
- Partial page updates (no full page reloads)
- Graceful error handling with error banners
- Responsive empty states and loading states
