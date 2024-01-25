package carto

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"strings"
	"sync"

	"github.com/Tnze/go-mc/nbt"
	"github.com/Tnze/go-mc/save"
)

var airBlocks = map[string]struct{}{
	"minecraft:air":         {},
	"minecraft:cave_air":    {},
	"minecraft:dead_bush":   {},
	"minecraft:short_grass": {},
	"minecraft:lily_pad":    {},
	"minecraft:torch":       {},
	"minecraft:wall_torch":  {},
}

func isAirBlock(block string) bool {
	_, ok := airBlocks[block]
	return ok
}

var grassBlocks = map[string]struct{}{
	"minecraft:grass":       {},
	"minecraft:grass_block": {},
	"minecraft:tall_grass":  {},
	"minecraft:vine":        {},
	"minecraft:fern":        {},
	"minecraft:large_fern":  {},
}

func isGrassBlock(block string) bool {
	_, ok := grassBlocks[block]
	return ok
}

var foliageBlocks = map[string]struct{}{
	"minecraft:oak_leaves":      {},
	"minecraft:jungle_leaves":   {},
	"minecraft:acacia_leaves":   {},
	"minecraft:dark_oak_leaves": {},
	"minecraft:mangrove_leaves": {},
	"minecraft:azalea_leaves":   {},
	"minecraft:cherry_leaves":   {},
}

func isFoliageBlock(block string) bool {
	_, ok := foliageBlocks[block]
	return ok
}

type BlockStateMultipart struct {
	Apply json.RawMessage `json:"apply"`
	When  json.RawMessage `json:"when"`
}

type BlockStateMultipartApply struct {
	Model string `json:"model"`
}

type BlockStateMultipartWhen map[string]string
type BlockStateMultipartWhenOr struct {
	Or []BlockStateMultipartWhen `json:"OR"`
}

type BlockStateVariant struct {
	Model string `json:"model"`
}

type BlockStateInfo struct {
	Variants  map[string]json.RawMessage `json:"variants"`
	Multipart []BlockStateMultipart      `json:"multipart"`
}

type ModelInfo struct {
	Textures map[string]string `json:"textures"`
}

type Palette struct {
	sync.RWMutex

	loader *AssetLoader

	biomeLock  sync.RWMutex
	biomeCache map[save.BiomeState]*Biome

	modelCache      map[string]ModelInfo
	blockStateCache map[string]BlockStateInfo
	textureCache    map[string]image.Image

	blockStateTextures map[string]string
	blockStateColors   map[string]color.Color

	grassColorMap   image.Image
	foliageColorMap image.Image
}

func NewPalette(loader *AssetLoader) *Palette {
	grassColorMap, err := loader.LoadPNG("assets/minecraft/textures/colormap/grass.png")
	if err != nil {
		log.Panicf("Failed to load grass colormap: %v", err)
	}
	foliageColorMap, err := loader.LoadPNG("assets/minecraft/textures/colormap/foliage.png")
	if err != nil {
		log.Panicf("Failed to load foliage colormap: %v", err)
	}
	return &Palette{
		loader:             loader,
		biomeCache:         make(map[save.BiomeState]*Biome),
		modelCache:         make(map[string]ModelInfo),
		blockStateCache:    make(map[string]BlockStateInfo),
		textureCache:       make(map[string]image.Image),
		blockStateTextures: make(map[string]string),
		blockStateColors:   make(map[string]color.Color),
		grassColorMap:      grassColorMap,
		foliageColorMap:    foliageColorMap,
	}
}

func (p *Palette) Prepare(section save.Section) {
	p.Lock()
	defer p.Unlock()

	for _, state := range section.BlockStates.Palette {
		stateStr := state.Name + "/" + state.Properties.String()

		if _, ok := p.blockStateColors[stateStr]; ok {
			continue
		}

		if isAirBlock(state.Name) {
			continue
		}
		p.prepareBlockState(state)
	}
}

func (p *Palette) getBiome(state save.BiomeState) *Biome {
	p.biomeLock.RLock()
	if res, ok := p.biomeCache[state]; ok {
		p.biomeLock.RUnlock()
		return res
	}
	p.biomeLock.RUnlock()

	p.biomeLock.Lock()
	defer p.biomeLock.Unlock()

	path := fmt.Sprintf("data/minecraft/worldgen/biome/%s.json", strings.Split(string(state), ":")[1])
	data, err := p.loader.LoadRaw(path)
	if err != nil {
		log.Panicf("failed to load biome %s: %s", state, err)
	}

	var biome Biome
	err = json.Unmarshal(data, &biome)
	if err != nil {
		log.Panicf("failed to unmarshal biome %s: %s", state, err)
	}

	p.biomeCache[state] = &biome
	return &biome
}

func (p *Palette) GetTexture(state save.BlockState) image.Image {
	p.RLock()
	defer p.RUnlock()
	stateStr := state.Name + "/" + state.Properties.String()
	texName, ok := p.blockStateTextures[stateStr]
	if !ok {
		return nil
	}
	return p.textureCache[texName]
}

func (p *Palette) GetColor(state save.BlockState, biome save.BiomeState) color.Color {
	p.RLock()
	defer p.RUnlock()
	stateStr := state.Name + "/" + state.Properties.String()

	color := p.blockStateColors[stateStr]

	if color != nil {
		return p.fixColor(state, color, biome)
	}
	return color
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	} else if v > max {
		return max
	} else {
		return v
	}
}

func (p *Palette) fixColor(state save.BlockState, clr color.Color, biome save.BiomeState) color.Color {
	if isGrassBlock(state.Name) {
		x, y := p.getBiome(biome).ColorMapCoords()
		return p.grassColorMap.At(x, y)
	} else if isFoliageBlock(state.Name) {
		x, y := p.getBiome(biome).ColorMapCoords()
		return p.foliageColorMap.At(x, y)
	} else if state.Name == "minecraft:birch_leaves" {
		return color.RGBA{
			R: 0x80,
			G: 0xa7,
			B: 0x55,
			A: 255,
		}
	} else if state.Name == "minecraft:spruce_leaves" {
		return color.RGBA{
			R: 0x61,
			G: 0x99,
			B: 0x61,
			A: 255,
		}
	} else if state.Name == "minecraft:water" {
		switch biome {
		case "minecraft:swamp":
			return color.RGBA{R: 0x61, G: 0x7B, B: 0x64, A: 255}
		case "minecraft:river":
			return color.RGBA{R: 0x3F, G: 0x76, B: 0xE4, A: 255}
		case "minecraft:ocean":
			return color.RGBA{R: 0x3F, G: 0x76, B: 0xE4, A: 255}
		case "minecraft:lukewarm_ocean":
			return color.RGBA{R: 0x45, G: 0xAD, B: 0xF2, A: 255}
		case "minecraft:warm_ocean":
			return color.RGBA{R: 0x43, G: 0xD5, B: 0xEE, A: 255}
		case "minecraft:cold_ocean":
			return color.RGBA{R: 0x3D, G: 0x57, B: 0xD6, A: 255}
		case "minecraft:frozen_river":
			return color.RGBA{R: 0x39, G: 0x38, B: 0xC9, A: 255}
		case "minecraft:frozen_ocean":
			return color.RGBA{R: 0x39, G: 0x38, B: 0xC9, A: 255}
		default:
			return color.RGBA{R: 0x3f, G: 0x76, B: 0xe4, A: 255}
		}
	}
	return clr
}

func (p *Palette) prepareBlockState(state save.BlockState) {
	blockStateInfo, ok := p.blockStateCache[state.Name]
	if !ok {
		rawName := strings.Split(state.Name, ":")[1]
		file, ok := p.loader.Files[fmt.Sprintf("assets/minecraft/blockstates/%s.json", rawName)]
		if !ok {
			log.Panicf("does not exist: %v", rawName)
		}

		fd, err := file.Open()
		if err != nil {
			log.Panicf("failed to open file %s: %v", rawName, err)
		}
		defer fd.Close()

		err = json.NewDecoder(fd).Decode(&blockStateInfo)
		p.blockStateCache[state.Name] = blockStateInfo
		if err != nil {
			log.Panicf("failed to decode json file %s: %v", rawName, err)
		}
	}

	propsMap := makeStatePropertiesMap(state.Properties)

	var modelName string

	if blockStateInfo.Multipart != nil {
		modelName = findMultipartModel(propsMap, blockStateInfo.Multipart)
	} else if len(blockStateInfo.Variants) == 1 {
		// TODO: does indexing this even matter?
		modelName = decodeVariants(firstVariant(blockStateInfo.Variants))[0].Model
	} else {
		modelName = findVariants(makeStatePropertiesMap(state.Properties), blockStateInfo.Variants)[0].Model
	}

	modelInfo, ok := p.modelCache[modelName]
	if !ok {
		rawName := strings.Split(modelName, ":")[1]
		file, ok := p.loader.Files[fmt.Sprintf("assets/minecraft/models/%s.json", rawName)]
		if !ok {
			log.Panicf("does not exist: %v (%v)", rawName, modelName)
		}

		fd, err := file.Open()
		if err != nil {
			log.Panicf("failed to open file %s: %v", rawName, err)
		}
		defer fd.Close()

		err = json.NewDecoder(fd).Decode(&modelInfo)
		p.modelCache[state.Name] = modelInfo
		if err != nil {
			log.Panicf("failed to decode json file %s: %v", rawName, err)
		}

	}

	var textureName string
	if len(modelInfo.Textures) == 1 {
		for _, v := range modelInfo.Textures {
			textureName = v
		}
	} else if top, ok := modelInfo.Textures["top"]; ok {
		textureName = top
	} else if all, ok := modelInfo.Textures["all"]; ok {
		textureName = all
	} else if all, ok := modelInfo.Textures["texture"]; ok {
		textureName = all
	} else {
		// TODO: random
		for _, v := range modelInfo.Textures {
			textureName = v
			break
		}
	}
	if textureName == "" || textureName == "#texture" {
		log.Panicf("failed to load for %v / %v", state.Name, modelName)
	}

	if strings.Contains(textureName, ":") {
		textureName = strings.Split(textureName, ":")[1]
	}

	stateStr := state.Name + "/" + state.Properties.String()
	p.blockStateTextures[stateStr] = textureName

	texture, ok := p.textureCache[textureName]
	if !ok {
		image, err := p.loader.LoadPNG(fmt.Sprintf("assets/minecraft/textures/%s.png", textureName))
		if err != nil {
			log.Panicf("failed to load texture image %s: %v", textureName, err)
		}

		texture = image
		p.textureCache[textureName] = image
	}

	p.blockStateColors[stateStr] = generateBlockStateColor(texture)
}

func generateBlockStateColor(texture image.Image) color.Color {
	bounds := texture.Bounds()
	var rr, gg, bb, aa, count float64
	for i := bounds.Min.X; i < bounds.Max.X; i++ {
		for j := bounds.Min.Y; j < bounds.Max.Y; j++ {
			col := texture.At(i, j)
			rrr, ggg, bbb, aaa := col.RGBA()
			rr += float64(rrr) * float64(aaa)
			gg += float64(ggg) * float64(aaa)
			bb += float64(bbb) * float64(aaa)
			aa += float64(aaa)
			count++
		}
	}
	return &color.RGBA64{
		R: uint16(rr / aa),
		G: uint16(gg / aa),
		B: uint16(bb / aa),
		A: uint16(aa / count),
	}
}

func makeStatePropertiesMap(msg nbt.RawMessage) map[string]string {
	test := map[string]string{}
	if msg.Type == nbt.TagEnd {
		return test
	}

	err := msg.Unmarshal(&test)
	if err != nil {
		panic(err)
	}
	return test
}

func firstVariant(variants map[string]json.RawMessage) json.RawMessage {
	for _, v := range variants {
		return v
	}
	return nil
}

func decodeVariants(raw json.RawMessage) []BlockStateVariant {
	var variants []BlockStateVariant
	err := json.Unmarshal(raw, &variants)
	if err == nil {
		return variants
	} else {
		var v BlockStateVariant
		err = json.Unmarshal(raw, &v)
		if err == nil {
			variants = append(variants, v)
			return variants
		}
	}
	return nil
}

func parseVariantProperties(raw string) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(raw, ",")
	for _, part := range parts {
		partSplit := strings.Split(part, "=")
		result[partSplit[0]] = partSplit[1]
	}
	return result
}

func findVariants(properties map[string]string, raw map[string]json.RawMessage) []BlockStateVariant {
	for k, v := range raw {
		p := parseVariantProperties(k)

		matches := true
		for k, v := range p {
			if properties[k] != v {
				matches = false
				continue
			}
		}
		if !matches {
			continue
		}

		return decodeVariants(v)
	}

	return nil
}

func decodeMultipart(raw BlockStateMultipart) ([]BlockStateMultipartApply, []BlockStateMultipartWhen) {
	applies := []BlockStateMultipartApply{}
	whens := []BlockStateMultipartWhen{}

	var apply BlockStateMultipartApply
	err := json.Unmarshal(raw.Apply, &apply)
	if err == nil {
		applies = append(applies, apply)
	} else {
		err = json.Unmarshal(raw.Apply, &applies)
		if err != nil {
			log.Panicf("FAILED APPLY: %v", string(raw.Apply))
		}
	}

	if len(raw.When) > 0 {
		var when BlockStateMultipartWhen
		err = json.Unmarshal(raw.When, &when)
		if err == nil {
			whens = append(whens, when)
		} else {
			err = json.Unmarshal(raw.When, &whens)
			if err != nil {

				var whenOr BlockStateMultipartWhenOr
				err = json.Unmarshal(raw.When, &whenOr)
				if err == nil {
					whens = append(whens, whenOr.Or...)
				} else {
					log.Panicf("FAILED WHEN: %v / %v", string(raw.When), string(raw.Apply))
				}
			}
		}
	}

	return applies, whens
}

func findMultipartModel(properties map[string]string, raw []BlockStateMultipart) string {
	for _, rawmp := range raw {
		applies, _ := decodeMultipart(rawmp)
		// no clue if this is actually valid, how do we interact with the whens array?
		return applies[0].Model

	}
	return ""
}
