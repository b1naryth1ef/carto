package carto

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/bits"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Tnze/go-mc/level"
	"github.com/Tnze/go-mc/save"
	"github.com/muesli/gamut"
)

type Biome struct {
	Temperature float64 `json:"temperature"`
	Downfall    float64 `json:"downfall"`
}

func (b *Biome) ColorMapCoords() (int, int) {
	r := clamp(b.Downfall, 0, 1) * clamp(b.Temperature, 0, 1)
	x := int(math.Ceil(255 - (clamp(b.Temperature, 0, 1) * 255)))
	y := int(math.Ceil(255 - (r * 255)))
	return x, y
}

type BiomeRenderer struct {
	biomes map[string]color.Color
}

func NewBiomeRenderer(loader *AssetLoader) *BiomeRenderer {
	biomes := make(map[string]color.Color)

	biomeNames := []string{}
	for path := range loader.Files {
		if strings.HasPrefix(path, "data/minecraft/worldgen/biome/") {
			name := strings.TrimSuffix(filepath.Base(path), ".json")
			biomeNames = append(biomeNames, fmt.Sprintf("minecraft:%s", name))
		}
	}

	colors, err := gamut.Generate(len(biomeNames), gamut.PastelGenerator{})
	if err != nil {
		log.Panicf("Failed to generate color palette for biomes: %v", err)
	}

	sort.Strings(biomeNames)
	for idx, biome := range biomeNames {
		biomes[biome] = colors[idx]
	}

	return &BiomeRenderer{
		biomes: biomes,
	}
}

func (c *BiomeRenderer) Finalize(path string) error {
	return nil
}

func (c *BiomeRenderer) ImageSize() (int, int) {
	return 16, 16
}

func (c *BiomeRenderer) RenderChunk(chunk *save.Chunk) (image.Image, error) {
	img := image.NewRGBA64(image.Rect(0, 0, 16, 16))
	bitsForHeight := bits.Len(uint(len(chunk.Sections))*16 + 1)
	motionBlocking := level.NewBitStorage(bitsForHeight, 16*16, chunk.Heightmaps["MOTION_BLOCKING"])

	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			heightmapIndex := ((z) * 16) + x
			yStart := motionBlocking.Get(heightmapIndex)
			sectionY := yStart / 16

			section := chunk.Sections[sectionY]
			if len(section.Biomes.Palette) == 0 {
				continue
			}

			v := calcBitsPerValue(16*16*16, len(section.Biomes.Data))
			biomes := level.NewBitStorage(v, 16*16*16, section.Biomes.Data)

			blockIndex := ((((sectionY) * 16) + z) * 16) + x
			biomeState := section.Biomes.Palette[biomes.Get(blockIndex)]

			color := c.biomes[string(biomeState)]
			if color != nil {
				img.Set(x, z, color)
			} else {
				log.Printf("unmapped biome %v", biomeState)
			}
		}
	}

	return img, nil
}
