var MapSelector = L.Control.extend({
	initialize(layers, options) {
		L.Util.setOptions(this, options);

		this._layers = layers;
		this._currentLayer = null;
		this._firstLayer = null;
	},

	onAdd: function (map) {
		let obj = L.DomUtil.create('div', 'leaflet-control-layers leaflet-control-layers-expanded');
		L.DomEvent.disableClickPropagation(obj);
		L.DomEvent.disableScrollPropagation(obj);

		for (const layer of Object.values(this._layers)) {
			obj.appendChild(this.makeItem(map, layer, this._firstLayer === null));

			if (this._firstLayer === null) {
				this._firstLayer = layer.name
			}
		}

		return obj;
	},

	addTo(map) {
		L.Control.prototype.addTo.call(this, map);
		this.setCurrentLayer(map, this._firstLayer);
		return this;
	},

	setCurrentLayer: function (map, name) {
		if (this._currentLayer !== null && this._layers[this._currentLayer].control !== undefined) {
			map.removeControl(this._layers[this._currentLayer].control);
		}

		map.eachLayer(function (layer) {
			map.removeLayer(layer);
		});

		this._layers[name].layer.addTo(map);
		if (this._layers[name].control !== undefined) {
			this._layers[name].control.addTo(map);
		}
		this._currentLayer = name;

		map.setView([0, 0], 3);
	},

	makeItem: function (map, layer, checked) {
		const label = document.createElement("label");

		label.innerHTML = `<span><input type="radio" class="leaflet-control-layers-selector" name="leaflet-base-layers_27" ${checked ? 'checked="checked"' : ''}><span> ${layer.name}</span></span>`
		label.firstChild.addEventListener('change', (e) => {
			this.setCurrentLayer(map, layer.name);
		});

		return label;
	}
});

var CoordViewer = L.Control.extend({
	onAdd: (map) => {
		var container = L.DomUtil.create('div');
		var gauge = L.DomUtil.create('coords');
		container.style.background = 'rgba(255,255,255,1)';
		container.style.textAlign = 'right';

		map.on('mousemove', event => {
			var coords = coord = map.project(event.latlng, 3);
			gauge.innerHTML = 'Coords: ' + Math.round(coords.x) + ", " + Math.round(coords.y);
		})
		container.appendChild(gauge);

		return container;
	}
});

function init(data) {
	const map = L.map('map', {
		crs: L.CRS.Simple,
		zoomDelta: 0.25,
		zoomSnap: 0,
		noWrap: true,
	});

	let maps = {};

	for (const mapData of data.maps) {
		let layers = {};
		let mainLayer;

		for (const layer of mapData.layers) {
			let thisLayer = L.tileLayer(`/tiles/${mapData.name}/${layer.name}/r.{x}.{y}.png`, {
				attribution: 'carto',
				minNativeZoom: 3,
				maxNativeZoom: 3,
				minZoom: 0,
				maxZoom: 5,
				tileSize: layer.tileSize,
				noWrap: true,
				opacity: layer.opacity === 0 ? 1 : layer.opacity
			});

			if (mainLayer === undefined) {
				mainLayer = thisLayer;
			} else {
				layers[layer.name] = thisLayer;
			}
		}

		maps[mapData.name] = {
			name: mapData.name,
			layer: mainLayer,
		};

		if (Object.keys(layers).length > 0) {
			maps[mapData.name].control = L.control.layers({}, layers, { collapsed: false });
		}
	}

	(new CoordViewer).addTo(map);
	(new MapSelector(maps)).addTo(map);
}

