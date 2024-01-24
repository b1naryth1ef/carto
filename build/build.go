package build

import (
	"compress/gzip"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/Tnze/go-mc/save"
	"github.com/b1naryth1ef/carto"
	"github.com/b1naryth1ef/carto/dl"
	"github.com/b1naryth1ef/carto/web"
)

type BuildOpts struct {
	ForceClean bool
}

func ensureDirectory(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = os.Mkdir(path, os.ModePerm)
		if err != nil {
			return err
		}
	} else {
		return err
	}
	return nil
}

func writeDirectory(path string, fs embed.FS, dir string) error {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			err = os.Mkdir(filepath.Join(path, entry.Name()), os.ModePerm)
			if err != nil && !os.IsExist(err) {
				return err
			}
			err = writeDirectory(filepath.Join(path, entry.Name()), fs, filepath.Join(dir, entry.Name()))
			if err != nil {
				return err
			}
		} else {
			contents, err := fs.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				return err
			}

			err = os.WriteFile(filepath.Join(path, entry.Name()), contents, os.ModePerm)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func writeStatic(path string, maps []string) error {
	err := os.WriteFile(filepath.Join(path, "index.html"), []byte(web.GetIndexHTML()), os.ModePerm)
	if err != nil {
		return err
	}

	err = ensureDirectory(filepath.Join(path, "static"))
	if err != nil {
		return err
	}

	fs := web.GetStaticContent()
	err = writeDirectory(filepath.Join(path, "static"), fs, ".")
	if err != nil {
		return err
	}

	data, err := fs.ReadFile("js/map.js.tmpl")
	if err != nil {
		return err
	}

	fd, err := os.Create(filepath.Join(path, "static", "js", "map.js"))
	if err != nil {
		return err
	}
	defer fd.Close()

	tmpl := template.Must(template.New("map.js").Parse(string(data)))
	err = tmpl.Execute(fd, map[string]interface{}{"Maps": maps, "Default": maps[0]})
	if err != nil {
		return err
	}

	return nil
}

func downloadClientJar(v string, path string) error {
	version, err := dl.GetVersionManifest()
	if err != nil {
		return err
	}

	var release *dl.Version
	if v == "" {
		release = version.GetLatestRelease()
	} else {
		release = version.GetRelease(v)
	}

	meta, err := release.GetMetadata()
	if err != nil {
		return err
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}

	return meta.Downloads["client"].Get(out)
}

func Build(config *carto.Config, opts BuildOpts) error {
	outputs := map[string]string{}
	outputMaps := map[string][]string{}
	for _, output := range config.Outputs {
		err := ensureDirectory(output.Path)
		if err != nil {
			return err
		}

		err = ensureDirectory(filepath.Join(output.Path, "tiles"))
		if err != nil {
			return err
		}

		err = ensureDirectory(filepath.Join(output.Path, "res"))
		if err != nil {
			return err
		}

		outputs[output.Name] = output.Path
		outputMaps[output.Name] = []string{}
	}

	layers := map[string]*carto.LayerConfigBlock{}
	for _, layer := range config.Layers {
		layers[layer.Name] = layer
	}

	for _, mapCfg := range config.Maps {
		outputPath := outputs[mapCfg.Output]

		tilePath := filepath.Join(outputPath, "tiles", mapCfg.Name)
		err := ensureDirectory(tilePath)
		if err != nil {
			return err
		}

		version := mapCfg.Version
		if version == "" {
			levelPath := filepath.Join(mapCfg.Path, "..", "level.dat")

			fd, err := os.Open(levelPath)
			if err != nil {
				return err
			}

			r, err := gzip.NewReader(fd)
			if err != nil {
				return err
			}

			level, err := save.ReadLevel(r)
			fd.Close()
			if err != nil {
				return err
			}

			version = level.Data.Version.Name
		}

		clientJarPath := filepath.Join(outputPath, "res", fmt.Sprintf("client-%s.jar", version))
		if _, err := os.Stat(clientJarPath); os.IsNotExist(err) {
			err = downloadClientJar(version, clientJarPath)
			if err != nil {
				return err
			}
		}

		assetLoader, err := carto.NewAssetLoaderFromClientJAR(clientJarPath)
		if err != nil {
			return err
		}
		defer assetLoader.Close()

		for _, layerName := range mapCfg.Layers {
			err := ensureDirectory(filepath.Join(tilePath, layerName))
			if err != nil {
				return err
			}

			renderOpts := carto.WorldRenderOpts{
				Concurrency: config.Concurrency,
			}

			var buildMeta carto.RenderMeta
			buildMetaPath := filepath.Join(tilePath, "build.json")

			if !opts.ForceClean {
				if _, err := os.Stat(filepath.Join(tilePath, "build.json")); err == nil {
					data, err := os.ReadFile(buildMetaPath)
					if err != nil {
						return err
					}

					err = json.Unmarshal(data, &buildMeta)
					if err != nil {
						return err
					}

					renderOpts.RegionTimestamps = buildMeta.RegionTimestamps
				}
			}

			var chunkRenderer carto.ChunkRenderer
			if layers[layerName].Render == "pixel" {
				renderOpts := carto.ChunkPixelRendererOpts{
					Shading: true,
				}
				chunkRenderer = carto.NewChunkPixelRenderer(renderOpts, assetLoader)
			} else if layers[layerName].Render == "biome" {
				chunkRenderer = carto.NewBiomeRenderer(assetLoader)
			} else {
				log.Panicf("Unsupported renderer '%s'", layers[layerName].Render)
			}

			renderer := carto.NewRenderer(chunkRenderer)

			start := time.Now()
			result, err := renderer.RenderWorld(mapCfg.Path, filepath.Join(tilePath, layerName), renderOpts)
			if err != nil {
				return err
			}

			buildMeta.RegionTimestamps = result.RegionTimestamps
			data, err := json.Marshal(buildMeta)
			if err != nil {
				return err
			}

			os.WriteFile(buildMetaPath, data, os.ModePerm)

			log.Printf("Finished rendering %s/%s in %dms (%d chunks)", mapCfg.Name, layerName, time.Since(start).Milliseconds(), result.RenderedChunks)
			outputMaps[mapCfg.Output] = append(outputMaps[mapCfg.Output], path.Join(mapCfg.Name, layerName))
		}
	}

	for _, output := range config.Outputs {
		if output.IncludeStatic {
			err := writeStatic(output.Path, outputMaps[output.Name])
			if err != nil {
				return err
			}
		}
	}

	return nil
}
