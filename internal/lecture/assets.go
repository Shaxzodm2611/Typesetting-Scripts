package lecture

import (
	"embed"
	"text/template"
)

//go:embed assets/*
var assets embed.FS

var sourceTemplate = template.Must(template.ParseFS(assets, "assets/lecture.tex.tmpl"))
