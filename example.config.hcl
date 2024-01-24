concurrency = 8

server {
  bind = "localhost:9874"
}

output "web" {
  path           = "/mnt/bigdata/mc"
  include_static = true
}

layer "normal" {
  render = "pixel"

  opts {
    shading = true
  }
}

map "overworld" {
  output = "web"
  path   = "/home/andrei/mc/world/region"
  layers = ["normal"]
}

map "hermitcraft9" {
  path    = "/mnt/bigdata/mc/tmp/region"
  layers  = ["normal"]
  version = "1.20.1"
}