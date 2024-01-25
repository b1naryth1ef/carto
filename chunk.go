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
