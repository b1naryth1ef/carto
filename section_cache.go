package carto

import (
	"github.com/Tnze/go-mc/level"
	"github.com/Tnze/go-mc/save"
)

type sectionCache struct {
	palette *Palette
	chunk   *save.Chunk
	cache   map[int]*sectionCacheItem
}

type sectionCacheItem struct {
	section save.Section
	storage *level.BitStorage
	biomes  *level.BitStorage
}

func newSectionCache(palette *Palette, chunk *save.Chunk) *sectionCache {
	return &sectionCache{
		palette: palette,
		chunk:   chunk,
		cache:   make(map[int]*sectionCacheItem),
	}
}

func (c *sectionCache) get(index int) *sectionCacheItem {
	sc, ok := c.cache[index]
	if !ok {
		if len(c.chunk.Sections) <= index {
			return nil
		}

		section := c.chunk.Sections[index]

		// prepare the palette for this section so we can lookup metadata for blockstates
		c.palette.Prepare(section)

		v := calcBitsPerValue(16*16*16, len(section.BlockStates.Data))
		storage := level.NewBitStorage(v, 16*16*16, section.BlockStates.Data)

		v = calcBitsPerValue(16*16*16, len(section.Biomes.Data))
		biomes := level.NewBitStorage(v, 16*16*16, section.Biomes.Data)
		sc = &sectionCacheItem{
			section: section,
			storage: storage,
			biomes:  biomes,
		}

		c.cache[index] = sc
	}
	return sc
}
