package carto

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/Tnze/go-mc/level"
)

type coord struct {
	X int
	Z int
}

// ChunkPixelShader handles shading for the ChunkPixelRenderer
type ChunkPixelShader struct {
	sync.RWMutex

	regions    map[coord]struct{}
	heightmaps map[coord]*level.BitStorage
}

func NewChunkPixelShader() *ChunkPixelShader {
	return &ChunkPixelShader{
		regions:    make(map[coord]struct{}),
		heightmaps: make(map[coord]*level.BitStorage),
	}
}

// add tracks a chunks heightmap for later-use in shading
func (c *ChunkPixelShader) add(x, z int, hm *level.BitStorage) {
	c.Lock()
	defer c.Unlock()

	regionX, regionZ := int(math.Floor(float64(x)/32.0)), int(math.Floor(float64(z)/32.0))
	c.regions[coord{X: regionX, Z: regionZ}] = struct{}{}

	crd := coord{X: x, Z: z}
	c.heightmaps[crd] = hm
}

// get returns the height map for a given chunk
func (c *ChunkPixelShader) get(crd coord) *level.BitStorage {
	c.RLock()
	defer c.RUnlock()
	return c.heightmaps[crd]
}

// Render generates and overlays shading for all regions
func (c *ChunkPixelShader) Render(path string) error {
	errors := make(chan error)

	var wg sync.WaitGroup
	for crd := range c.regions {
		wg.Add(1)
		go func(crd coord) {
			defer wg.Done()
			err := c.renderRegion(path, crd)
			if err != nil {
				errors <- err
			}
		}(crd)
	}

	go func() {
		wg.Wait()
		close(errors)
	}()

	for err := range errors {
		return err
	}

	return nil
}

// renderRegion handles rendering, merging, and saving the shading for a region image
func (c *ChunkPixelShader) renderRegion(path string, crd coord) error {
	shadeImg := image.NewRGBA64(image.Rect(0, 0, 32*16, 32*16))
	for x := 0; x < 32; x++ {
		for z := 0; z < 32; z++ {
			chunkCrd := coord{X: (crd.X * 32) + x, Z: (crd.Z * 32) + z}
			hm := c.get(chunkCrd)
			if hm == nil {
				continue
			}

			chunkImg, err := c.renderChunk(chunkCrd)
			if err != nil {
				return fmt.Errorf("failed to render region shading for (%v, %v): %v", crd.X, crd.Z, err)
			}
			pnt := image.Point{
				x * 16,
				z * 16,
			}
			draw.Draw(shadeImg, chunkImg.Bounds().Add(pnt), chunkImg, image.Point{0, 0}, draw.Src)
		}
	}

	fd, err := os.Open(filepath.Join(path, fmt.Sprintf("r.%d.%d.png", crd.X, crd.Z)))
	if err != nil {
		return fmt.Errorf("failed to open region final image (%v, %v): %v", crd.X, crd.Z, err)
	}

	srcImg, err := png.Decode(fd)
	if err != nil {
		return fmt.Errorf("failed to decode region final image (%v, %v): %v", crd.X, crd.Z, err)
	}
	fd.Close()

	finalImg := image.NewRGBA64(srcImg.Bounds())

	draw.Draw(finalImg, srcImg.Bounds(), srcImg, image.Point{0, 0}, draw.Src)
	draw.Draw(finalImg, shadeImg.Bounds(), shadeImg, image.Point{0, 0}, draw.Over)

	fd, err = os.Create(filepath.Join(path, fmt.Sprintf("r.%d.%d.png", crd.X, crd.Z)))
	if err != nil {
		return fmt.Errorf("failed to open region final image (%v, %v): %v", crd.X, crd.Z, err)
	}
	defer fd.Close()

	err = png.Encode(fd, finalImg)
	if err != nil {
		return fmt.Errorf("failed to encode final image (%v, %v): %v", crd.X, crd.Z, err)
	}

	return nil
}

// renderChunk handles generating a shaded overlay image for a single chunk
func (c *ChunkPixelShader) renderChunk(crd coord) (image.Image, error) {
	img := image.NewRGBA64(image.Rect(0, 0, 16, 16))
	hm := c.get(crd)

	for x := 0; x < 16; x++ {
		for z := 0; z < 16; z++ {
			var topHeight int
			var leftHeight int

			heightmapIndex := ((z) * 16) + x
			height := hm.Get(heightmapIndex)

			// if x == 0 we must fetch the left height from another chunk
			if x == 0 {
				leftHeightMap := c.get(coord{X: crd.X - 1, Z: crd.Z})
				if leftHeightMap != nil {
					heightmapIndex := ((z) * 16) + 15
					leftHeight = leftHeightMap.Get(heightmapIndex)
				} else {
					leftHeight = height
				}
			} else {
				heightmapIndex := ((z) * 16) + (x - 1)
				leftHeight = hm.Get(heightmapIndex)
			}

			// if z == 0 we must fetch the top height from another chunk
			if z == 0 {
				topHeightMap := c.get(coord{X: crd.X, Z: crd.Z - 1})
				if topHeightMap != nil {
					heightmapIndex := ((15) * 16) + x
					topHeight = topHeightMap.Get(heightmapIndex)
				} else {
					topHeight = height
				}
			} else {
				heightmapIndex := ((z - 1) * 16) + x
				topHeight = hm.Get(heightmapIndex)
			}

			var d int
			if topHeight > height {
				d = (topHeight - height) * 16
			}
			if leftHeight > height {
				d += (leftHeight - height) * 16
			}
			if d > 64 {
				d = 64
			}

			img.Set(x, z, color.RGBA{
				R: 0,
				G: 0,
				B: 0,
				A: uint8(d),
			})
		}
	}

	return img, nil
}
