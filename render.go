package carto

import (
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Tnze/go-mc/save"
	"github.com/Tnze/go-mc/save/region"
)

type Renderer struct {
	chunk ChunkRenderer
}

func NewRenderer(chunk ChunkRenderer) *Renderer {
	return &Renderer{
		chunk: chunk,
	}
}

func (r *Renderer) RenderWorld(src, dst string, opts WorldRenderOpts) (*WorldRenderResult, error) {
	entries, err := os.ReadDir(src)
	if err != nil {
		return nil, err
	}

	var renderedChunks atomic.Uint32

	result := WorldRenderResult{
		RegionTimestamps: make(map[string]int32),
	}

	concurrency := opts.Concurrency
	if concurrency == 0 {
		concurrency = runtime.GOMAXPROCS(0)
	}
	guard := make(chan struct{}, concurrency)

	var wg sync.WaitGroup
	for _, e := range entries {
		reg, err := region.Open(filepath.Join(src, e.Name()))
		if errors.Is(err, io.EOF) {
			continue
		}

		if err != nil {
			log.Printf("[renderer] failed to open region file %s: %v", filepath.Join(src, e.Name()), err)
			continue
		}

		guard <- struct{}{}
		wg.Add(1)
		go func(name string, reg *region.Region) {
			defer wg.Done()
			defer reg.Close()
			defer func() {
				<-guard
			}()

			regionName := strings.TrimSuffix(name, filepath.Ext(name))

			var previousMaxTimestamp int32
			if opts.RegionTimestamps != nil {
				previousMaxTimestamp = opts.RegionTimestamps[regionName]
			}

			img, maxTimestamp, renderedChunkCount, err := r.renderRegion(reg, previousMaxTimestamp)
			renderedChunks.Add(renderedChunkCount)
			if err != nil {
				log.Printf("[renderer] failed to render region file %s: %v", filepath.Join(src, name), err)
				return
			}

			if img == nil {
				result.Lock()
				result.RegionTimestamps[regionName] = previousMaxTimestamp
				result.Unlock()
				return
			}

			result.Lock()
			result.RegionTimestamps[regionName] = maxTimestamp
			result.Unlock()

			regionImageName := regionName + ".png"

			f, err := os.Create(filepath.Join(dst, regionImageName))
			if err != nil {
				log.Printf("[renderer] failed to render region file %s: %v", filepath.Join(src, name), err)
				return
			}
			defer f.Close()

			if err := png.Encode(f, img); err != nil {
				log.Printf("[renderer] failed to encode region png %s: %v", filepath.Join(dst, regionImageName), err)
			}
		}(e.Name(), reg)
	}
	wg.Wait()

	err = r.chunk.Finalize(dst)
	if err != nil {
		return nil, err
	}

	result.RenderedChunks = renderedChunks.Load()

	return &result, nil
}

type chunkImageResult struct {
	Timestamp int32
	X         int
	Z         int
	Image     image.Image
	Error     error
}

func (r *Renderer) renderRegion(reg *region.Region, previousMaxTimestamp int32) (image.Image, int32, uint32, error) {
	chunkImageHeight, chunkImageWidth := r.chunk.ImageSize()
	regionImageHeight := chunkImageHeight * 32
	regionImageWidth := chunkImageWidth * 32

	img := image.NewRGBA64(image.Rect(0, 0, regionImageHeight, regionImageWidth))

	chunkImages := make(chan chunkImageResult)

	needRender := false
	for x := 0; x < 32; x++ {
		for z := 0; z < 32; z++ {
			chunkTimestamp := reg.Timestamps[z][x]
			if chunkTimestamp > previousMaxTimestamp {
				needRender = true
			}
		}
	}

	if !needRender {
		return nil, 0, 0, nil
	}

	var wg sync.WaitGroup
	for x := 0; x < 32; x++ {
		for z := 0; z < 32; z++ {
			sector, err := reg.ReadSector(x, z)
			if errors.Is(err, region.ErrNoSector) {
				continue
			}

			if len(sector) == 0 {
				return nil, 0, 0, fmt.Errorf("sector is out of bounds")
			}

			chunkTimestamp := reg.Timestamps[z][x]

			wg.Add(1)
			go func(x, z int) {
				defer wg.Done()

				image, err := r.renderSector(sector)
				chunkImages <- chunkImageResult{
					Timestamp: chunkTimestamp,
					X:         x,
					Z:         z,
					Image:     image,
					Error:     err,
				}
			}(x, z)
		}
	}

	go func() {
		wg.Wait()
		close(chunkImages)
	}()

	var maxTimestamp int32
	chunkCount := uint32(0)
	for chunkImage := range chunkImages {
		if chunkImage.Error != nil {
			return nil, 0, 0, chunkImage.Error
		}

		if chunkImage.Image == nil {
			continue
		}

		if chunkImage.Timestamp > maxTimestamp {
			maxTimestamp = chunkImage.Timestamp
		}

		chunkCount += 1
		draw.Draw(img, chunkImage.Image.Bounds().Add(image.Point{
			chunkImage.X * chunkImageHeight,
			chunkImage.Z * chunkImageWidth,
		}), chunkImage.Image, image.Point{0, 0}, draw.Src)
	}

	return img, maxTimestamp, chunkCount, nil
}

func (r *Renderer) renderSector(sector []byte) (image.Image, error) {
	var chunk save.Chunk
	err := chunk.Load(sector)
	if err != nil {
		return nil, err
	}

	if chunk.Status != "minecraft:full" &&
		chunk.Status != "minecraft:spawn" &&
		chunk.Status != "minecraft:postprocessed" &&
		chunk.Status != "minecraft:fullchunk" {
		return nil, nil
	}

	return r.chunk.RenderChunk(&chunk)
}
