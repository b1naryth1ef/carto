package web

type FrontendData struct {
	Maps []MapData `json:"maps"`
}

type MapData struct {
	Name   string      `json:"name"`
	Layers []LayerData `json:"layers"`
}

type LayerData struct {
	Name     string  `json:"name"`
	TileSize int     `json:"tileSize"`
	Opacity  float64 `json:"opacity"`
}
