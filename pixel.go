package carto

import (
	"image"
	"image/color"
	"math/bits"
	"sync"

	"github.com/Tnze/go-mc/level"
	"github.com/Tnze/go-mc/save"
)

type ChunkPixelRenderer struct {
	sync.Mutex

	opts               *ChunkRenderOpts
	shader             *ChunkPixelShader
	palette            *Palette
	missingBlockStates map[string]struct{}

	stripCeiling bool
}

func NewChunkPixelRenderer(opts *ChunkRenderOpts, assetLoader *AssetLoader) *ChunkPixelRenderer {
	palette := NewPalette(assetLoader)
	shader := NewChunkPixelShader()
	return &ChunkPixelRenderer{
		opts:               opts,
		shader:             shader,
		palette:            palette,
		missingBlockStates: make(map[string]struct{}),
		stripCeiling:       opts.GetBool("strip-ceiling", false),
	}
}

func (c *ChunkPixelRenderer) Finalize(path string) error {
	if !c.opts.GetBool("shading", true) {
		return nil
	}

	return c.shader.Render(path)
}

func (c *ChunkPixelRenderer) ImageSize() (int, int) {
	return 16, 16
}

func (c *ChunkPixelRenderer) RenderChunk(chunk *save.Chunk) (image.Image, error) {
	bitsForHeight := bits.Len(uint(len(chunk.Sections))*16 + 1)
	motionBlocking := level.NewBitStorage(bitsForHeight, 16*16, chunk.Heightmaps["MOTION_BLOCKING"])
	oceanFloor := level.NewBitStorage(bitsForHeight, 16*16, chunk.Heightmaps["OCEAN_FLOOR"])

	if len(chunk.Sections) == 0 {
		return nil, nil
	}

	c.shader.add(int(chunk.XPos), int(chunk.ZPos), motionBlocking)

	img := image.NewRGBA64(image.Rect(0, 0, 16, 16))

	cache := newSectionCache(c.palette, chunk)

	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			heightmapIndex := ((z) * 16) + x
			yStart := motionBlocking.Get(heightmapIndex)
			underCeiling := false

			for y := yStart; y > 1; y-- {
				sectionIndex := y / 16
				sectionY := y % 16
				sc := cache.get(sectionIndex)
				if sc == nil {
					continue
				}

				if len(sc.section.BlockStates.Palette) == 0 {
					continue
				}

				blockIndex := ((((sectionY) * 16) + z) * 16) + x
				blockState := sc.section.BlockStates.Palette[sc.storage.Get(blockIndex)]
				biomeState := sc.section.Biomes.Palette[sc.biomes.Get(blockIndex)]

				// if we're stripping the ceiling we need to wait for the first airblock
				if c.stripCeiling && !underCeiling {
					if !isAirBlock(blockState.Name) || y == yStart {
						continue
					} else {
						underCeiling = true
					}
				}

				if isAirBlock(blockState.Name) {
					continue
				}

				clr := c.palette.GetColor(blockState, biomeState)
				if clr == nil {
					c.Lock()
					c.missingBlockStates[blockState.Name] = struct{}{}
					c.Unlock()
					continue
				}

				// for water we want to darken things based on the depth of the water
				if blockState.Name == "minecraft:water" {
					oceanFloorY := oceanFloor.Get(heightmapIndex)
					d := (y - oceanFloorY) * 8
					if d > 128 {
						d = 128
					}
					clr = combineColor(clr, color.RGBA{
						R: 0,
						G: 0,
						B: 0,
						A: uint8(d),
					})
				}

				img.Set(x, z, clr)
				break
			}
		}
	}

	return img, nil
}

func (c *ChunkPixelRenderer) GetMissingBlockStates() []string {
	result := []string{}
	for k := range c.missingBlockStates {
		result = append(result, k)
	}
	return result
}

func calcBitsPerValue(length, longs int) (bits int) {
	if longs == 0 || length == 0 {
		return 0
	}
	valuePerLong := (length + longs - 1) / longs
	return 64 / valuePerLong
}

func combineColor(c1, c2 color.Color) color.Color {
	r, g, b, a := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()

	return color.RGBA{
		uint8((r + r2) >> 9),
		uint8((g + g2) >> 9),
		uint8((b + b2) >> 9),
		uint8((a + a2) >> 9),
	}
}
