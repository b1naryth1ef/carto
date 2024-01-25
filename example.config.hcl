concurrency = 8

output "web" {
  path           = "/mnt/bigdata/mc"
  include_static = true
}

layer "normal" {
  render = "pixel"
}

layer "biome" {
  render  = "biome"
  opacity = 0.5
}

layer "light" {
  render = "light"
}

map "overworld" {
  output = "web"
  path   = "/home/andrei/mc/world/region"
  layers = ["normal", "biome", "light"]
}

map "hermitcraft9" {
  output  = "web"
  path    = "/mnt/bigdata/mc/tmp/region"
  layers  = ["normal", "biome", "light"]
  version = "1.20.1"
}