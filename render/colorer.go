package render

import (
	"image/color"

	"github.com/MazenAlkhatib/treemap"
)

var (
	DarkTextColor  color.Color = color.Black
	LightTextColor color.Color = color.White
)

type NoneColorer struct{}

func (s NoneColorer) ColorBox(tree treemap.Tree, node string) color.Color {
	return color.Transparent
}

func (s NoneColorer) ColorText(tree treemap.Tree, node string) color.Color {
	return DarkTextColor
}
