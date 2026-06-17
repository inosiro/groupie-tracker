# Testing

Run all automated tests with:

```sh
go test ./...
```

## Automated Audit Coverage

The Go test suite covers:

- Project has no non-standard module dependencies in `go.mod`.
- The filter UI template contains range inputs.
- The filter UI template contains checkbox inputs.
- Creation date range filtering.
- First album year range filtering.
- Member checkbox filtering.
- Concert location filtering, including normalized multi-token input.
- Combined filters for creation date, first album year, member count, and concert location.
- Reset/show-all behavior when filters are empty.

## Manual Browser Checklist

Use this checklist for behavior that is best confirmed in the browser:

- Sliders trigger asynchronous filtering without pressing "Apply Filters".
- Number inputs and sliders stay visually in sync for Creation Year and First Album Year.
- Reset All clears the UI controls and shows all artists/bands.
- Region checkboxes still disable the location text input as before.
