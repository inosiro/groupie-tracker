# Groupie Tracker (Go + HTMX)

## 1. Overview
Groupie Tracker is a web-based artist exploration tool that consumes a specialized external API to visualize band information, including members, concert locations, dates, and relationships. 

The application is built with a focus on high performance and clean architecture, utilizing Go's standard library for the backend and HTMX for a dynamic, reactive user interface without the need for complex frontend frameworks.

## 2. Core Features

- **Artist Discovery**: A responsive grid of artist cards displaying core metadata such as formation year and first album release.
- **Dynamic Detail Views**: Interactive buttons on each artist card allow users to load specific data fragments on-demand:
    - **Members**: A list of current and past band members.
    - **Concerts**: Relation data mapping specific locations to their respective performance dates.
    - **Locations**: A focused list of concert venues and cities.
    - **Dates**: A chronological list of performance dates.
- **Interactive Geolocalization**: A dynamic mapping feature that visualizes concert locations on an interactive map. When triggered, the artist card performs a 3D rotation animation using CSS and HTMX to reveal a MapLibre-powered map populated with geocoded concert markers.
- **Advanced Filtering**: Narrow down artists by formation year, first album release, number of members, or specific concert locations and dates using high-performance server-side logic.
- **Intelligent Caching**: A TTL (Time-To-Live) based in-memory cache reduces external API load and ensures sub-millisecond response times for frequently accessed data.
- **Graceful Degradation**: Robust error handling ensures the server remains responsive even if upstream API services experience downtime.

## 3. Concurrency and Asynchronicity

The application leverages Go's powerful concurrency model and HTMX's asynchronous nature to provide a seamless user experience.

### Server-Side Concurrency (Go)
- **Implicit Concurrency**: The Go `net/http` package automatically spawns a new **goroutine** for every incoming HTTP request. This allows the server to handle multiple users and data fetches simultaneously without blocking.
- **Thread Safety**: Access to the shared artist cache is protected by a `sync.Mutex`, preventing data races during concurrent cache updates.
- **Timeout Management**: All external API calls utilize `context.Context` with a 5-second timeout, ensuring that hanging upstream requests do not exhaust server resources.

### Client-Side Asynchronicity (HTMX)
- **AJAX Swaps**: UI actions (like clicking "Concerts" or "Members") trigger asynchronous AJAX requests via HTMX. 
- **Partial Rendering**: Instead of reloading the entire page, the server returns HTML fragments which are swapped into the DOM. This provides a "Single Page Application" feel while maintaining the simplicity of server-side templates.
- **Animated Transitions**: HTMX handles the content swap for the map view, while CSS transitions manage the 3D "flip" animation of the artist card to transition between the profile and the map.

## 4. Technical Stack

- **Backend**: Go 1.22+ (Strictly standard library: `net/http`, `html/template`, `sync`, `context`).
- **Frontend**: HTMX (for event-driven DOM updates), MapLibre GL JS (for interactive maps), and CSS3 (for animations).
- **Data Source**: External Groupie Trackers API.

## 5. API Endpoints

| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/` | GET | Renders the initial home page shell. |
| `/artists` | GET | Returns the artist grid (Full page or HTMX fragment). |
| `/artists/{id}` | GET | Returns the concert relations fragment for a specific artist. |
| `/members/{id}` | GET | Returns the members list fragment. |
| `/locations/{id}` | GET | Returns the concert locations fragment. |
| `/dates/{id}` | GET | Returns the concert dates fragment. |
| `/artists/{id}/concerts.geojson` | GET | Returns the geoJSON of concerts for a specific artist
| `/healthz` | GET | Health check endpoint for monitoring. |
| `/static/` | GET | Serves CSS and the HTMX library. |

## 6. Setup and Installation

1. Ensure you have Go 1.22 or later installed.
2. Clone the repository to your local machine.
3. Start the server:
   ```bash
   go run main.go
   ```
4. Open your browser and navigate to `http://localhost:8080`.

## 7. Architecture Note
This project follows a "View Model" pattern to separate API data structures from template requirements:
1. **Models**: Map directly to API JSON responses.
2. **Cache**: Manages data persistence and TTL.
3. **Handlers**: Orchestrate data fetching and view model transformation.
4. **Templates**: Pure HTML fragments using Go's `html/template` engine for safe, auto-escaped rendering.

## 8. Robustness
As per project requirements, the server is designed to be "crash-proof":
- **Input Validation**: Path and query parameters are strictly validated before processing.
- **API Resilience**: Failures in upstream API calls for details (Locations/Dates) are captured, and the user is presented with a friendly error banner rather than a broken page or server crash.

##  License

These projects were completed as part of the Zone01 Athens curriculum.
