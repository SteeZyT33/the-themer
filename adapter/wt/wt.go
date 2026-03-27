// Package wt generates Windows Terminal color scheme JSON.
// The output is a standalone JSON object that can be added to the "schemes"
// array in Windows Terminal's settings.json.
package wt

import (
	"bytes"
	"text/template"

	"github.com/kylesnowschwartz/the-themer/adapter"
	"github.com/kylesnowschwartz/the-themer/palette"
)

func init() {
	adapter.Register(&wtAdapter{})
}

type wtAdapter struct{}

func (w *wtAdapter) Name() string                     { return "windows-terminal" }
func (w *wtAdapter) DirName() string                  { return "windows-terminal" }
func (w *wtAdapter) FileName(themeName string) string { return themeName + ".json" }

func (w *wtAdapter) Generate(cfg palette.Config) ([]byte, error) {
	var buf bytes.Buffer
	if err := wtTmpl.Execute(&buf, cfg); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// wtTmpl renders a Windows Terminal color scheme.
//
// ANSI color mapping:
//
//	color0  = black        color8  = brightBlack
//	color1  = red          color9  = brightRed
//	color2  = green        color10 = brightGreen
//	color3  = yellow       color11 = brightYellow
//	color4  = blue         color12 = brightBlue
//	color5  = purple       color13 = brightPurple
//	color6  = cyan         color14 = brightCyan
//	color7  = white        color15 = brightWhite
var wtTmpl = template.Must(template.New("wt").Parse(
	`{
    "name": "{{.Theme.Name}}",
    "background": "{{.Palette.BG}}",
    "foreground": "{{.Palette.FG}}",
    "cursorColor": "{{.Palette.Cursor}}",
    "selectionBackground": "{{.Palette.SelectionBG}}",
    "black": "{{.Palette.Color0}}",
    "red": "{{.Palette.Color1}}",
    "green": "{{.Palette.Color2}}",
    "yellow": "{{.Palette.Color3}}",
    "blue": "{{.Palette.Color4}}",
    "purple": "{{.Palette.Color5}}",
    "cyan": "{{.Palette.Color6}}",
    "white": "{{.Palette.Color7}}",
    "brightBlack": "{{.Palette.Color8}}",
    "brightRed": "{{.Palette.Color9}}",
    "brightGreen": "{{.Palette.Color10}}",
    "brightYellow": "{{.Palette.Color11}}",
    "brightBlue": "{{.Palette.Color12}}",
    "brightPurple": "{{.Palette.Color13}}",
    "brightCyan": "{{.Palette.Color14}}",
    "brightWhite": "{{.Palette.Color15}}"
}`))
