package template

import "embed"

//go:embed service-kit/** service-kit/**/*
var EmbeddedTemplates embed.FS
