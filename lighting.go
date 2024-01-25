package carto

import (
	"image"
	"image/color"
	"math/bits"

	"github.com/Tnze/go-mc/level"
	"github.com/Tnze/go-mc/save"
)

type LightingRenderer struct {
}

func NewLightingRenderer() *LightingRenderer {
	return &LightingRenderer{}
}

func (c *LightingRenderer) Finalize(path string) error {
	return nil
}

func (c *LightingRenderer) ImageSize() (int, int) {
	return 16, 16
}

func (c *LightingRenderer) RenderChunk(chunk *save.Chunk) (image.Image, error) {
	img := image.NewRGBA64(image.Rect(0, 0, 16, 16))
	bitsForHeight := bits.Len(uint(len(chunk.Sections))*16 + 1)
	motionBlocking := level.NewBitStorage(bitsForHeight, 16*16, chunk.Heightmaps["MOTION_BLOCKING"])

	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			heightmapIndex := ((z) * 16) + x
			yStart := motionBlocking.Get(heightmapIndex)
			sectionY := yStart / 16

			section := chunk.Sections[sectionY]

			blockLight := byte(0)
			if len(section.BlockLight) > 0 {
				blockLightIndex := x + ((sectionY & 0x0f) << 8) + (z << 4)
				blockLightRaw := section.BlockLight[blockLightIndex/2]

				if blockLightIndex&1 > 0 {
					blockLight = (blockLightRaw >> 4) & 0x0F
				} else {
					blockLight = (blockLightRaw & 0x0F)
				}

			}

			a := 192 - ((blockLight + 1) * 12)
			img.Set(x, z, color.RGBA{
				R: 0,
				G: 0,
				B: 0,
				A: a,
			})
		}
	}

	return img, nil
}
