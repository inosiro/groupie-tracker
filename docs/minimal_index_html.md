Below is a **minimal `index.html`** that uses **HTMX** to (1) load the artist grid, (2) support **search + basic filters** with zero custom JS, and (3) leave clean hooks for **location/date filtering** (explained right after).

This matches your project constraints: the UI triggers server requests (“event/action”) and the server returns updated HTML fragments, instead of JSON + client-side rendering as in your current sample.

## Minimal `templates/pages/index.html` (HTMX)

```
{{define "content"}}
<main class="container">
  <header>
    <h1>Bands &amp; Artists</h1>
    <p class="muted">
      Search and filter artists. Click an artist to load concerts.
    </p>
  </header>

  <!-- Filters: single GET form. HTMX submits it and swaps the grid fragment. -->
  <form id="filters"
        class="filters"
        hx-get="/artists"
        hx-target="#artist-grid"
        hx-swap="innerHTML"
        hx-push-url="true"
        hx-trigger="submit, change delay:150ms, keyup changed delay:300ms from:input[name='q'], keyup changed delay:400ms from:input[name='location'], keyup changed delay:400ms from:input[name='date']">

    <div class="row">
      <label>
        Search
        <input type="search" name="q" placeholder="band name or member…" autocomplete="off">
      </label>

      <label>
        Members min
        <input type="number" name="members_min" min="1" placeholder="e.g. 3">
      </label>

      <label>
        Members max
        <input type="number" name="members_max" min="1" placeholder="e.g. 6">
      </label>
    </div>

    <div class="row">
      <label>
        Creation from
        <input type="number" name="creation_from" min="1900" max="2100" placeholder="e.g. 1980">
      </label>

      <label>
        Creation to
        <input type="number" name="creation_to" min="1900" max="2100" placeholder="e.g. 2005">
      </label>

      <label>
        Sort
        <select name="sort">
          <option value="">Relevance</option>
          <option value="name">Name</option>
          <option value="creation">Creation year</option>
          <option value="members">Members</option>
        </select>
      </label>
    </div>

    <!-- Geo/Concert-ready hooks -->
    <details class="advanced">
      <summary>Concert filters (location/date)</summary>
      <p class="muted">
        These may trigger extra server work because concert data lives in relations.
      </p>
      <div class="row">
        <label>
          Location contains
          <input type="text" name="location" placeholder="e.g. athens, france…">
        </label>

        <label>
          Date contains
          <input type="text" name="date" placeholder="e.g. 2012-10 or 12-04-2018…">
        </label>

        <!-- Optional: instruct server to apply expensive relation filtering -->
        <label class="checkbox">
          <input type="checkbox" name="include_relations" value="1">
          Apply concert filters (slower)
        </label>
      </div>
    </details>

    <div class="row actions">
      <button type="submit">Apply</button>
      /Reset</a>

      <!-- HTMX indicator -->
      <span id="loading" class="muted" aria-live="polite"></span>
    </div>
  </form>

  <!-- Grid container: loads initial results on page load -->
  <section id="artist-grid"
           hx-get="/artists"
           hx-trigger="load"
           hx-target="this"
           hx-swap="innerHTML"
           hx-indicator="#loading">
    <div class="muted">Loading artists…</div>
  </section>
</main>
{{end}}
```
### What this expects from your server

* `GET /` renders `layout.html` + this `pages/index.html` (full page).
* `GET /artists?...` returns an **HTML fragment** (e.g., `partials/artist_grid.html`) that contains cards.
* Each card’s “Show concerts” button uses HTMX to call `GET /artists/{id}` and swaps in `partials/artist_details.html` (replacing your current `/api/artist?id=` JSON + `app.js` flow).\
  This still satisfies the project requirement for an event/action involving client-server request/response.


# Clarifying filtering by **location/date** (important architectural point)

### Why location/date filtering is different

In the given API design:

* `/artists` contains the “artist profile” info (name, image, members, creationDate, firstAlbum).
* Concert **locations + dates** are not in `/artists`; they come from **relations** (mapping `location -> [dates]`).

So: **to filter by location or date**, you generally need relation data per artist (e.g., `/relation/{id}`), which can mean **many upstream calls** if you do it across the whole dataset.

## Recommended approach (clean + extendable)

### ✅ Option A (Best default): “Two-phase filtering” (fast + scalable)

1. **Always apply “cheap filters” first** on `/artists` data:
   * `q`, creation range, members range, sorting, pagination
2. Only if user enables concert filters (`include_relations=1`) *or* if results are already small:
   * Fetch relations for that reduced candidate set
   * Filter candidates by `location` / `date`

**Why it’s good:**

* Keeps the app responsive
* Minimal API calls most of the time
* Fits your “clean, simple, extendable” goal while still supporting the feature

**Server behavior example:**

* If `include_relations` is unchecked → ignore `location/date` and show a hint: “Enable ‘Apply concert filters’ to filter by concerts.”
* If checked → apply them, but with guardrails (see below).

### ✅ Option B: Separate “Concert Search” endpoint/page (clean separation)

Create an explicit mode:

* `GET /concerts?location=...&date=...` returns artists that match concert criteria

**Why it’s good:**

* Keeps `/artists` purely about artist metadata
* The intent is clear: “I’m searching concerts, not profiles”
* Easy to enhance later with geo “near me” without slowing down the artist browse view

## Guardrails you should implement if you support location/date filtering on `/artists`

Because the server “cannot crash” and must handle errors robustly, you should avoid worst-case API fan-out.
1. **Cache relations per artist ID (TTL)**\
   Your sample already caches artists with TTL; extend that pattern to relations.

2. **Cap the candidate set for relation lookups**\
   Example policy:
   * Apply cheap filters first
   * If candidates > 50 and `include_relations=1`, either:
     * return a message: “Too many results—narrow your search first”, or
     * automatically apply pagination before relation filtering

3. **Use timeouts + concurrency limits**
   * Use `context.WithTimeout` (your sample already uses 5s timeouts).
   * If you fetch many relations, use a small worker pool (e.g., 5–10 concurrent requests) to avoid overload.

4. **Degrade gracefully**
   * If upstream relation fetch fails for some artists, either:
     * exclude those from concert-filtered results and show “Some results omitted due to upstream errors”, or
     * include them but mark concerts as “unavailable”.


## Practical rule of thumb (what I’d implement for your PRD goals)

* **Default**: `/artists` supports search + artist-only filters (fast).
* **Advanced**: enable concert filters via `include_relations=1` and apply the two-phase approach with caching + caps.
* This keeps the project simple but unlocks the feature cleanly.
