package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// --- ТЕМА АПТЕЧНАЯ ЗЕЛЁНКА :| ---

type forestTheme struct{}

var _ fyne.Theme = (*forestTheme)(nil)

func (t *forestTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	darkGreen := color.NRGBA{R: 0x1e, G: 0x3b, B: 0x23, A: 0xff}
	lightGreen := color.NRGBA{R: 0xea, G: 0xf4, B: 0xea, A: 0xff}
	limeGreen := color.NRGBA{R: 0x32, G: 0xcd, B: 0x32, A: 0xff}
	panelGreen := color.NRGBA{R: 0x2a, G: 0x52, B: 0x33, A: 0xff}

	switch name {
	case theme.ColorNameBackground:
		return darkGreen
	case theme.ColorNameForeground:
		return lightGreen
	case theme.ColorNamePrimary:
		return limeGreen
	case theme.ColorNameInputBackground, theme.ColorNameMenuBackground, theme.ColorNameHeaderBackground:
		return panelGreen
	case theme.ColorNameButton:
		return panelGreen
	case theme.ColorNameDisabledButton:
		return color.NRGBA{R: 0x24, G: 0x47, B: 0x28, A: 0xff}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0x6c, G: 0x75, B: 0x7d, A: 0xff}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x3b, G: 0x68, B: 0x44, A: 0xff}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x8f, G: 0xbc, B: 0x8f, A: 0xff}
	case theme.ColorNameScrollBar:
		return panelGreen
	default:
		return theme.DarkTheme().Color(name, variant)
	}
}

func (t *forestTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DarkTheme().Icon(name)
}
func (t *forestTheme) Font(style fyne.TextStyle) fyne.Resource { return theme.DarkTheme().Font(style) }
func (t *forestTheme) Size(name fyne.ThemeSizeName) float32    { return theme.DarkTheme().Size(name) }
