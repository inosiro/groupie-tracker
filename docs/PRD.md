## PRD — Groupie Tracker (Go + HTMX)

### 1. Problem Statement

The current Groupie Trackers project requires building a user-friendly website that consumes a provided API and visualizes band/artist information (artists, locations, dates, relations).   

### 2. User / Use Case

**Primary user:** a visitor exploring bands/artists and their concerts.

**Core use cases**

1. **Browse artists** in a list/grid (cards) with image/name and basic stats. (artists endpoint contains image, name, members, creation date, first album, etc.)
2. **View artist details** (members + concerts by location and date) when the user clicks an artist card. (relation links locations and dates)
3. **Filter artists** by metadata (creation year, first album year, member count) or concert data (location, date).

### 3. HTTP/UI contract

* `GET /` — renders the shell page (layout + initial artist list view).
* `GET /artists` — returns **HTML fragment** for the artist list.
* `GET /artists/{id}` — returns **HTML fragment** for a single artist’s details (concerts).
* `GET /healthz` — returns `200 OK` for uptime checks.

### 4. Functional Requirements (Rules)

#### FR1 — Consume the given Groupie Tracker API

The backend must call the provided API and manipulate the data to display it on a website.   
**Example:** On initial load, fetch `/artists` and render cards with `Name`, `Image`, and `Members`.

#### FR2 — Display multiple data visualizations

The UI should visualize band info via components such as cards, lists, tables, etc.   
**Example:** Artists as cards; concerts as a table grouped by city.

#### FR3 — Implement at least one event/action (client ↔ server)

Project explicitly requires an event/action involving a client call to the server (request/response).
**HTMX example:** Clicking “Show concerts” triggers `hx-get="/artists/12"` and swaps a details panel with the returned HTML fragment.

#### FR4 — Robustness and error handling (must not crash)

The website/server cannot crash; handle errors gracefully.
**Example:** If the external API is unreachable, render an error banner and keep the server responsive.

#### FR5 — Backend must be Go, using only standard packages

Backend implementation must be in Go, and only standard library packages are allowed.   
**Example:** Use `net/http`, `html/template`, `encoding/json`, `context`, `time`, `sync`.

#### FR6 — Caching to minimize API calls

Adopt a simple cache (TTL-based) for artists list and relations to reduce latency and external API load.
**Example:** Cache artists for 2 minutes; relations cached per artist for 2 minutes.

#### FR7 — Advanced Filtering
The application must support multi-criteria filtering:
* **Range Filters**: Band creation date (year) and First Album date (year).
* **Checkbox Filters**: Number of members (one or multiple selection) and a toggle for "Include Relations" to enable expensive geo-filtering.
* **Text Filters**: Specific concert filters (location/date).

#### FR8 — Geolocalization and Mapping
The application must visualize concert locations on an interactive map.
* **Geocoding**: The server or client must convert location strings (e.g., "london-uk") into geographic coordinates.
* **Map Visualization**: Use **MapLibre GL JS** to render maps.
* **UI Animation**: Clicking a "Show Map" action should trigger an HTMX request that facilitates a card-flip animation to show the map on the "back" of the artist card.

### 5. Non-Goals (Out of Scope)

* Building a full SPA or introducing a frontend framework (React/Vue).
* Persisting user accounts, favorites, or comments (no DB).
* Third-party Go packages (disallowed by project constraints).

### 6. Acceptance Criteria

#### 6.1 Audit Cases (as tests)

(Represented as HTTP requests with expected outcomes.)

* [ ] **Audit case 1: Home renders**
  * Input: `GET /`
  * Expected: `200 OK`, contains “Bands & Artists” heading and at least one artist card container.

* [ ] **Audit case 2: Artist details fragment loads**
  * Input: `GET /artists/1`
  * Expected: `200 OK`, HTML fragment contains concerts section grouped by location (from relation data).

* [ ] **Audit case 3: API failure does not crash**
  * Input: simulate upstream API down (via test stub or by forcing client error)
  * Expected: `200 OK` (or `503` with friendly HTML), server stays up for subsequent requests.

* [ ] **Audit case 4: Locations data is used**
  * Input: `GET /artists` (select any artist), the `GET /locations/{id}`
  * Expected: HTML fragment displays concert locations (from `/api/locations/{id}` endpoint).

* [ ] **Audit case 5: Dates data is used**
  * Input: `GET /artists` (select any artist), then `GET /dates/{id}`
  * Expected: HTML fragment displays concert dates (from `/api/dates/{id}` endpoint).

* [ ] **Audit case 6: Members are displayed for any artist**
  * Input: `GET /artists` (select any artist), then `GET /members/{id}`
  * Expected: HTML fragment includes members list with at least one member name displayed.

* [ ] **Audit case 7: First Album date is displayed**
  * Input: `GET /artists` (view artist cards)
  * Expected: Artist cards display firstAlbum information in expected format.

* [ ] **Audit case 8: Concert locations are displayed**
  * Input: `GET /artists` (select any artist with concert data), then `GET /locations/{id}`
  * Expected: HTML fragment displays a list of concert locations.

* [ ] **Audit case 9: External API failure handled gracefully**
  * Input: Simulate external Groupie Trackers API failure, then make requests to:
    - `GET /artists`
    - `GET /artists/{id}`
    - `GET /locations/{id}`
    - `GET /dates/{id}`
  * Expected: 
    - All endpoints return error response (not 5xx crash)
    - Error messages are user-friendly
    - Server remains responsive for subsequent requests
    - No unhandled panic or crash

* [ ] **Audit case 10: Cache miss with upstream API failure**
  * Input: First request (empty cache, cache miss) when upstream API is unavailable, then:
    - `GET /artists` (no cached data available)
    - `GET /artists/{id}` (cache miss + API unavailable)
    - `GET /locations/{id}` (cache miss + API unavailable)
    - `GET /dates/{id}` (cache miss + API unavailable)
    - Multiple concurrent requests (stress test)
  * Expected:
    - System returns graceful error (not 5xx crash)
    - No panic or unhandled exceptions
    - Server remains responsive for all requests
    - Response is complete (no HTML corruption/truncation)
    - No server deadlock or resource leak

* [ ] **Audit case 11: Range filtering works**
  * Input: `GET /artists?creation_from=1960&creation_to=1980`
  * Expected: Only bands formed between 1960 and 1980 are returned.

* [ ] **Audit case 12: Checkbox filtering works**
  * Input: `GET /artists?members=4&members=5`
  * Expected: Only bands with exactly 4 or 5 members are returned.

* [ ] **Audit case 13: Concert location/date filtering works**
  * Input: `GET /artists?location=london&include_relations=1`
  * Expected: Only bands that have played in London are returned.

* [ ] **Audit case 14: Map rendering**
  * Input: Click "Show Map" on an artist card.
  * Expected: The card animates (rotates), an HTMX request fetches the map context, and MapLibre renders markers for all concert locations associated with that artist.

#### 6.2 Additional Golden Tests

* [ ] **Cache behavior:** two consecutive `GET /artists` calls within TTL should hit cache (validate by instrumentation/log counter).

### 7. Implementation Approach (High Level)

#### Decision Summary

* **We choose:** Go server-rendered templates + **HTMX** for interactions returning HTML fragments.
* **Because:** It keeps the project **clean, simple**, reduces the use of JS and makes search/filter extensions straightforward by adding query params and partial templates.

#### Proposed Structure (extendable)

```
groupie-web/
├─ go.mod
├─ main.go
├─ internal/
│  ├─ audit_test.go
│  ├─ filters_test.go
│  ├─ filters.go
│  ├─ groupie_api.go       # API client for /artists, /relation, /locations, and /dates
│  ├─ groupie_cache.go     # Thread-safe TTL cache for the artists list
│  ├─ groupie_models.go    # Core data structures mapping to API responses
│  ├─ web_handlers.go      # HTTP handlers for main page and HTMX fragments
│  ├─ web_render.go        # template parsing + render helpers (full vs partial)
│  └─ web_viewmodels.go    # structs passed into templates (ArtistCardVM, GridVM, DetailsVM)
├─ templates/
│  ├─ layout.html          # base layout (includes htmx)
│  ├─ index.html           # full page body 
│  └─ partials/
│     ├─ artist_grid.html       # grid/list of cards (HTMX swap target)
│     ├─ artist_card.html       # individual artist card component
│     ├─ artist_locations.html  # concert locations list fragment
│     ├─ artist_members.html    # band members list fragment
│     ├─ artist_dates.html      # concert dates list fragment
│     ├─ artist_details.html    # relations fragment (locations + dates)
│     ├─ error_banner.html      # optional reusable error partial
│     └─ empty_state.html       # optional reusable empty state
└─ static/
   ├─ htmx.min.js          # HTMX library for AJAX interactions
   └─ styles.css
```

#### HTMX Interaction Pattern

* Full page loads return **layout + page template**.
* HTMX requests return **partials only** (no full layout).
* Grid updates use `GET /artists` to refresh the grid.
* Details use `GET /artists/{id}` to fill an expandable region.
* **Error handling:** When API fails:
  - If `HX-Request: true` header present: return error fragment (via RenderPartial)
  - Otherwise: return full page with error (via RenderPage)
  - This allows graceful degradation whether user is in browser or HTMX context

#### Sketch (1 diagram)

```
Browser (HTMX)
  |  GET /                      -> full HTML (layout + index)
  |
  |  hx-get /artists?q=...       -> HTML fragment (artist grid)
  |
  |  hx-get /artists/{id}        -> HTML fragment (artist details)
  v
Go Server (net/http)
  -> groupie.Client fetches API (/artists, /relation/{id})  
  -> Cache (TTL) to reduce calls                                
  -> html/template renders pages/partials
```

### 8. Milestones

1. **M1 — Baseline rendering:** `GET /` renders artist cards from API with caching + error handling.
2. **M2 — HTMX details action:** click a card button loads `/artists/{id}` fragment and swaps into the card (no custom JS).
3. **M3 — Hardening + tests:** timeouts, upstream errors; add golden tests (net/http/httptest).

### 9. Risks / Open Questions

* **Risk:** Upstream API availability/latency — must not crash; require timeouts and friendly degradation.
* **Risk:** HTML fragment consistency — must ensure partial templates don’t duplicate layout and are safe to swap.