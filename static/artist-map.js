window.initArtistMap = (geoJSON, id) => {
    const card = document.getElementById(`card-${id}`);
    card.classList.add('flipped');

    const container = document.getElementById(`map-${id}`);
    // Avoid re-initializing if already done
    if (container.dataset.loaded === 'true') return;
    container.dataset.loaded = 'true';

    const map = new maplibregl.Map({
        container: container,
        // style: 'https://tiles.openfreemap.org/styles/bright',
        style: 'https://demotiles.maplibre.org/style.json',
        center: [0, 20],
        zoom: 1
    });

    const geoData = typeof geoJSON === 'string' ? JSON.parse(geoJSON) : geoJSON;

    console.log(geoData);
    // console.log(geoData.features?.length);

    map.on('load', async () => {

        // add concerts points for markers
        map.addSource('concerts', { type: 'geojson', data: geoData });

        // extract coords to build a LineString route
        const coords = geoData.features.map(f => f.geometry.coordinates)
        map.addSource('route', {
            'type': 'geojson',
            'data': {
                'type': 'Feature',
                'properties': {},
                'geometry': {
                    'type': 'LineString',
                    'coordinates': coords,
                }
            }
        });

        // Visible dashed line for the tour route
        map.addLayer({
            'id': 'route-line',
            'type': 'line',
            'source': 'route',
            'layout': {
                'line-join': 'round',
                'line-cap': 'round',
            },
            'paint': {
                'line-color': '#ff5a5f',
                'line-width': 3,
                'line-dasharray': [2, 2],
            }
        });

        // Hidden solid line for arrow placement.
        // symbol-placement:'line' does NOT work on dashed lines —
        // MapLibre cannot resolve symbol positions on dash-segmented geometry.
        // This invisible solid line gives the symbol layer a continuous path to place arrows on.
        map.addLayer({
            'id': 'route-line-solid',
            'type': 'line',
            'source': 'route',
            'layout': {
                'line-join': 'round',
                'line-cap': 'round',
            },
            'paint': {
                'line-color': 'rgba(0,0,0,0)',
                'line-width': 1,
            }
        });

        // Generate a small arrow icon dynamically via canvas
        const arrowSize = 16;
        const canvas = document.createElement('canvas');
        canvas.width = arrowSize;
        canvas.height = arrowSize;
        const ctx = canvas.getContext('2d');
        ctx.fillStyle = '#ff5a5f';
        ctx.beginPath();
        ctx.moveTo(2, 3);
        ctx.lineTo(14, 8);
        ctx.lineTo(2, 13);
        ctx.closePath();
        ctx.fill();
        const imageData = ctx.getImageData(0, 0, arrowSize, arrowSize);
        map.addImage('arrow-icon', {
            width: arrowSize,
            height: arrowSize,
            data: new Uint8Array(imageData.data.buffer)
        });

        // Place arrows along the hidden solid line
        map.addLayer({
            'id': 'route-arrows',
            'type': 'symbol',
            'source': 'route',
            'layout': {
                'symbol-placement': 'line',
                'symbol-spacing': 80,
                'icon-image': 'arrow-icon',
                'icon-size': 1,
                'icon-rotate': 0,
                'icon-rotation-alignment': 'map',
                'icon-allow-overlap': true,
                'icon-ignore-placement': true
            }
        });

        // add layer for concerts markers
        map.addLayer({
            'id': 'concerts',
            'type': 'circle',
            'source': 'concerts',
            'paint': {
                'circle-radius': 8,
                'circle-color': '#0a6f77',
                'circle-stroke-width': 2,
                'circle-stroke-color': '#fff',
            }
        });

        // add labels for the markers
        map.addLayer({
            'id': 'concert-labels',
            'type': 'symbol',
            'source': 'concerts',
            'layout': {
                'text-font': ['Noto Sans Regular'],
                'text-field': ['concat', ['get', 'sequence_index'], ' ', ['get', 'location_label']],
                'text-size': 14,
                'text-offset': [0, 1.25],
                'text-anchor': 'top',
            },
            'paint': {
                'text-color': '#111',
                'text-halo-color': '#fff',
                'text-halo-width': 3,
            }
        });


        const bounds = new maplibregl.LngLatBounds();
        coords.forEach(c => bounds.extend(c))
        // geoData.features.forEach(f => bounds.extend(f.geometry.coordinates));
        if (!bounds.isEmpty()) map.fitBounds(bounds, { padding: 50 });
    });
}