package web

import (
	"embed"
)

//go:embed index.html.tmpl
var indexHTML string

//go:embed js/*
var staticContent embed.FS

func GetIndexHTML() string {
	return indexHTML
}

func GetStaticContent() embed.FS {
	return staticContent
}
