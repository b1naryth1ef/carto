package carto

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

type Config struct {
	Concurrency int                  `hcl:"concurrency,optional"`
	Outputs     []*OutputConfigBlock `hcl:"output,block"`
	Layers      []*LayerConfigBlock  `hcl:"layer,block"`
	Maps        []*MapConfigBlock    `hcl:"map,block"`
}

type OutputConfigBlock struct {
	Name          string `hcl:"name,label"`
	Path          string `hcl:"path"`
	IncludeStatic bool   `hcl:"include_static,optional"`
}

type LayerConfigBlock struct {
	Name    string  `hcl:"name,label"`
	Render  string  `hcl:"render"`
	Opacity float64 `hcl:"opacity,optional"`
}

type MapConfigBlock struct {
	Name    string   `hcl:"name,label"`
	Output  string   `hcl:"output"`
	Path    string   `hcl:"path"`
	Layers  []string `hcl:"layers"`
	Version string   `hcl:"version,optional"`
}

func newHCLEvalContext() *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{},
		Functions: map[string]function.Function{},
	}
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config
	evalCtx := newHCLEvalContext()
	err := hclsimple.DecodeFile(path, evalCtx, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
