package carto

import (
	"archive/zip"
	"fmt"
	"image"
	"image/png"
	"io"
	"strings"
)

type AssetLoader struct {
	Files map[string]*zip.File

	reader *zip.ReadCloser
}

func NewAssetLoaderFromClientJAR(path string) (*AssetLoader, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}

	files := make(map[string]*zip.File)
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "assets/") && !strings.HasPrefix(f.Name, "data/") {
			continue
		}
		files[f.Name] = f
	}

	return &AssetLoader{
		Files:  files,
		reader: r,
	}, nil
}

func (a *AssetLoader) LoadPNG(name string) (image.Image, error) {
	file, ok := a.Files[name]
	if !ok {
		return nil, fmt.Errorf("file %s does not exist", name)
	}

	fd, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	return png.Decode(fd)
}

func (a *AssetLoader) LoadRaw(name string) ([]byte, error) {
	file, ok := a.Files[name]
	if !ok {
		return nil, fmt.Errorf("file %s does not exist", name)
	}

	fd, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	return io.ReadAll(fd)
}

func (a *AssetLoader) Close() {
	a.reader.Close()
}
