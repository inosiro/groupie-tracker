Below is a concrete adaptation plan that takes **your existing sample structure** (Go + templates + cache + a JSON “details” endpoint + JS) and reshapes it into a **clean HTMX-first layout**, plus **exact endpoint contracts** and **partial template responsibilities**—optimized for easy extension to **filtering**, **search**, and **geo-ready** features.

## 1\) Adapt the Existing Structure → HTMX-Friendly Folder Layout

### Current sample (summary)

You currently have:

* `groupie/` with `api.go`, `models.go`, `cache.go`
* `templates/layout.html` + `templates/index.html` rendering the list
* a JSON endpoint `/api/artist?id=` and `static/app.js` building HTML in JS

That already has a good separation for the API client and caching. We’ll keep those ideas, but:

* move code under `internal/…` to clarify boundaries,
* replace the JSON endpoint + JS rendering with **HTML fragments** returned from the server,
* split templates into `pages/` and `partials/`.

### What moves where (mapping from your sample)

* `groupie/` → `internal/groupie/`
* `renderIndex` + handler code currently in `main.go` → `internal/web/handlers.go` + `internal/web/render.go`
* `/api/artist?id=` handler (JSON) → `GET /artists/{id}` handler (HTML fragment)
* `static/app.js` becomes unnecessary for the “Show Concerts” action; HTMX handles the request/response swap (still satisfies the project’s event/action requirement)

## 2\) Exact Endpoint Contracts (Requests, Params, Responses)

The project API you consume is made of **artists, locations, dates, relation** and you must display these in a user-friendly website. Your sample already calls `/artists` and `/relation/{id}` (via `Client.Artists` and `Client.Relation`). We’ll standardize app endpoints around that.

### Common HTMX behavior rule

* If request has header `HX-Request: true`, server returns **partial HTML** (no layout).
* Otherwise, server returns **full page** (layout + page).

This keeps endpoints usable directly in the browser while allowing HTMX fragment swaps.

### **GET /** — Home page shell

**Purpose:** Render the main page with search/filter controls and a container for results.  
**Response:**

* Full HTML: `templates/layout.html` + `templates/pages/index.html`
* The index page triggers HTMX to load `/artists` on page load (so grid rendering logic stays consistent).

**Status codes:**

* `200 OK` always if possible.
* If upstream API is down, page still loads, but the grid container swaps to an error partial (server must not crash).

### **GET /artists** — Artist list (full page OR fragment)

**Purpose:** Provide the artist grid/list, filtered by query params.

**Query params (extensible):**

* `q` — search term (matches artist name and member names)
* `creation\_from`, `creation\_to` — numeric year range
* `first\_album\_from`, `first\_album\_to` — numeric year range (parsed from `FirstAlbum` string when possible)
* `members\_min`, `members\_max` — numeric range
* `sort` — e.g. `name|creation|members` (optional)
* `page`, `page\_size` — optional pagination (recommended for extendability)

**Response body:**

* If `HX-Request:true`: `templates/partials/artist\_grid.html`
* Else: full page (`layout.html` + `pages/index.html`) with the grid included (or server-side redirect to `/`).

**Status codes:**

* `200 OK` for valid requests.
* `400 Bad Request` for invalid numeric params (or treat invalids as ignored—your choice, but must be safe and not crash).

**Data source:**

* Cached artists list via TTL cache (your sample already does this).

### **GET /artists/{id}** — Artist details fragment

**Purpose:** Return detail panel HTML for a given artist (members + concerts by location/date).

**Path param:**

* `id` integer

**Response body:**

* `templates/partials/artist\_details.html` (fragment only; it will be swapped into the card’s detail container)

**Status codes:**

* `200 OK` if found.
* `404 Not Found` if artist id does not exist.
* `502/503` (or `200` with error fragment) if upstream API fails; must not crash.

**Data source:**

* Find artist in cached list (your sample loops cached artists to find by ID)
* Fetch relation from upstream: `/relation/{id}` (your sample does this)
* Optionally add a small relation cache per ID.

### **GET /healthz**

**Purpose:** Simple health check.  
**Response:** `200 OK` text/plain `"ok"`.

## 3\) Partial Template Responsibilities (What Each Template Owns)

Your sample currently renders cards in `index.html` and uses JS to fetch JSON and build the concerts HTML into a `div id="result-ID"`. With HTMX, the server directly returns those HTML snippets.

### `templates/layout.html` (Full page wrapper)

**Responsibilities:**

* Document `<html>`, `<head>`, global CSS.
* Include HTMX script (CDN).
* Provide a top-level content block where pages render.

**Must NOT:**

* Contain artist grid markup (that belongs to partials).
* Contain per-page search/filter forms (belongs to `pages/index.html`).

### `templates/pages/index.html` (Home page body)

**Responsibilities:**

* Display title/header (e.g. “Bands \& Artists” as in sample)
* Provide search input + filter UI (form elements).
* Provide a results container:

  * `div#artist-grid` (swap target)
* Wire up HTMX triggers:

  * Search input triggers `hx-get="/artists"` with query string values
  * Filters trigger `hx-get="/artists"`
  * On load, fetch initial grid (`hx-trigger="load"`)

**Must NOT:**

* Render the artist list itself inline (optional, but recommended to keep a single source of truth for grid rendering: `/artists`).



### `templates/partials/artist\_grid.html` (HTMX swap target)

**Responsibilities:**

* Render **only** the list/grid of artists given a `GridVM`:

  * count summary (“N results”)
  * loop cards
  * handle empty state (or include `empty\_state.html`)

**Inputs (suggested `GridVM`):**

* `Artists \[]ArtistCardVM`
* `Query FilterVM` (optional for showing “active filters” chips)
* `Error string` (optional; if set, show error banner and maybe no cards)

**Must NOT:**

* Include `<html>`, `<head>`, page wrappers.
* Include HTMX script tags.

### `templates/partials/artist\_card.html` (Optional include)

**Responsibilities:**

* Render a single artist card (image, name, members count, dates).
* Include the “Show concerts” button wired to HTMX:

  * `hx-get="/artists/{{.ID}}"`
  * `hx-target="#details-{{.ID}}"`
  * `hx-swap="innerHTML"`
* Provide a placeholder `<div id="details-{{.ID}}"></div>` to receive the details fragment.

This directly replaces your sample’s `onclick="loadDetails({{.ID}})"` + `result-ID` approach.

### `templates/partials/artist\_details.html` (Concert details fragment)

**Responsibilities:**

* Render details for one artist:

  * Members list
  * Concerts grouped by location with dates (based on `Relation.DatesLocations`)

**Inputs (suggested `DetailsVM`):**

* `Artist` (name, members, creation date, first album)
* `Concerts \[]LocationConcertVM` where each item:

  * `Location string`
  * `Dates \[]string`

**Must NOT:**

* Query upstream API directly (logic belongs in handler/service).
* Include layout wrappers.

### `templates/partials/error\_banner.html` (Optional)

**Responsibilities:**

* Render a small banner with a friendly message, e.g. “Failed to load artists. Please try again.”

Supports the “must not crash” requirement by gracefully surfacing failures.

### `templates/partials/empty\_state.html` (Optional)

**Responsibilities:**

* Render “No results found” state for searches/filters.

## 4\) Handler Responsibilities (Clean Separation)

You already have:

* cache: `Cache.Artists(ctx)` with TTL logic
* API client methods `Artists(ctx)` and `Relation(ctx,id)`

We keep those, and layer on:

### `internal/web/handlers.go`

* `Index(w,r)`:

  * Render full page only.
  * Do **not** fetch artists here (optional), or fetch only for server-side pre-render. Prefer letting `/artists` handle grid population so filters/search always use the same path.
* `Artists(w,r)`:

  * Parse filters/search: `internal/filters/query.go`
  * Fetch cached artists list: `cache.Artists(ctx)`
  * Apply filter: `internal/filters/apply.go`
  * Render:

    * if HTMX request → `partials/artist\_grid.html`
    * else → full page including grid
* `ArtistDetails(w,r)`:

  * Parse `{id}`
  * Find artist in cached list (your sample does this loop)
  * Fetch relation via API client: `api.Relation(ctx,id)`
  * Transform relation map into slice for stable iteration ordering
  * Render `partials/artist\_details.html`

### `internal/web/render.go`

* Centralize template parsing/caching.
* Provide helpers:

  * `RenderPage(w, "index", vm)`
  * `RenderPartial(w, "artist\_grid", vm)`

### `internal/web/viewmodels.go`

Define template-focused viewmodels to avoid leaking raw API structs everywhere:

* `ArtistCardVM`
* `GridVM`
* `DetailsVM`
* `FilterVM` (parsed/validated query state)

This makes the project easier to extend without rewriting templates.



## 5\) Exact Request/Response Examples (Concrete Contracts)

### Example: Search typing (HTMX)

**Client (index page):**

* `<input name="q" hx-get="/artists" hx-trigger="keyup changed delay:300ms" hx-target="#artist-grid" hx-push-url="true">`

**Request:**

```
GET /artists?q=metallica
HX-Request: true
```

**Response:**

* HTML fragment: `<div class="grid">…filtered cards…</div>`

### Example: Show concerts (event/action requirement)

This satisfies “client call to server → receive info” requirement  without custom JS.

**Request:**

```
GET /artists/12
HX-Request: true
```

**Response:**

* HTML fragment with concerts grouped by location/dates derived from relation data

## 6\) Small but Important Extendability Decisions

### Filtering on location/date (where data lives)

* Artist list endpoint `/artists` only has artist data (name, members, etc.)
* Location/date filtering requires relation data which is per-artist (`/relation/{id}`)

**Extendable approach:**

* Phase 1: filters that only need artist fields (fast).
* Phase 2: when user applies `location` or `date` filters, you can:

  * lazily fetch relations for candidates (cache results), or
  * provide a separate “Concert Search” view/endpoints.

This keeps the base app responsive and simple.

