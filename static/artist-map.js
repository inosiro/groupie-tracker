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

    // console.log(geoData);
    // console.log(geoData.features?.length);

    map.on('load', async () => {
        // const image = await map.loadImage('./static/marker2.png');
        // map.addImage('custom-marker', image.data);

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

        // add layer to draw a dashed line connecting concerts
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

        const bounds = new maplibregl.LngLatBounds();
        coords.forEach(c => bounds.extend(c))
        // geoData.features.forEach(f => bounds.extend(f.geometry.coordinates));
        if (!bounds.isEmpty()) map.fitBounds(bounds, { padding: 50 });
    });
}