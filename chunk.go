package carto

import (
	"image"

	"github.com/Tnze/go-mc/save"
)

type ChunkRenderer interface {
	ImageSize() (int, int)
	RenderChunk(*save.Chunk) (image.Image, error)
	Finalize(path string) error
}

type ChunkRenderOpts struct {
	data map[string]string
}

func NewChunkRenderOpts(data map[string]string) *ChunkRenderOpts {
	return &ChunkRenderOpts{data: data}
}

func (c *ChunkRenderOpts) GetBool(key string, def bool) bool {
	v, ok := c.data[key]
	if !ok {
		return def
	}

	if v == "true" || v == "1" {
		return true
	}

	return false
}
