package carto

import "sync"

type RenderMeta struct {
	RegionTimestamps map[string]int32
}

type WorldRenderOpts struct {
	Concurrency      int
	RegionTimestamps map[string]int32
}

type WorldRenderResult struct {
	sync.Mutex

	RenderedChunks   uint32
	RegionTimestamps map[string]int32
}
