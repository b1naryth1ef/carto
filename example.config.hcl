concurrency = 8

output "web" {
  path           = "/mnt/bigdata/mc"
  include_static = true
}

layer "normal" {
  render  = "pixel"
  shading = true
}

layer "biome" {
  render = "biome"
}

map "overworld" {
  output = "web"
  path   = "/home/andrei/mc/world/region"
  layers = ["normal", "biome"]
}

map "hermitcraft9" {
  output  = "web"
  path    = "/mnt/bigdata/mc/tmp/region"
  layers  = ["normal", "biome"]
  version = "1.20.1"
}