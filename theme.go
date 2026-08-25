package main

import "github.com/gdamore/tcell/v3"

var (
	colorFileBanner       = tcell.NewRGBColor(57, 75, 101)
	colorFileBannerActive = tcell.NewRGBColor(76, 101, 137)
	colorSidebarActive    = tcell.NewRGBColor(47, 63, 85)

	styleAddition = tcell.StyleDefault.
			Foreground(tcell.NewRGBColor(184, 231, 198)).
			Background(tcell.NewRGBColor(21, 59, 39))
	styleDeletion = tcell.StyleDefault.
			Foreground(tcell.NewRGBColor(242, 195, 198)).
			Background(tcell.NewRGBColor(72, 37, 42))
	styleEmpty = tcell.StyleDefault.
			Background(tcell.NewRGBColor(34, 37, 43))
	stylePaneHeader = tcell.StyleDefault.
			Foreground(tcell.NewRGBColor(150, 158, 171)).
			Background(tcell.NewRGBColor(30, 36, 45))
	styleBeforeLabel  = stylePaneHeader.Foreground(tcell.NewRGBColor(231, 160, 164)).Bold(true)
	styleAfterLabel   = stylePaneHeader.Foreground(tcell.NewRGBColor(143, 213, 165)).Bold(true)
	styleUnifiedLabel = stylePaneHeader.Foreground(tcell.NewRGBColor(174, 198, 235)).Bold(true)
	styleSingleFile   = stylePaneHeader.Foreground(tcell.NewRGBColor(174, 198, 235)).Bold(true)
	styleFileBanner   = tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(226, 231, 239)).
				Bold(true)
	styleFileEyebrow = tcell.StyleDefault.Foreground(tcell.NewRGBColor(166, 175, 188)).Bold(true)
	styleFileDetail  = tcell.StyleDefault.Foreground(tcell.NewRGBColor(166, 175, 188))
	styleHunk        = tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(169, 192, 229)).
				Background(tcell.NewRGBColor(32, 42, 58))
	styleMeta = tcell.StyleDefault.
			Foreground(tcell.NewRGBColor(165, 173, 186)).
			Background(tcell.NewRGBColor(29, 31, 36))
	styleMuted                = tcell.StyleDefault.Foreground(tcell.NewRGBColor(140, 145, 155))
	styleDivider              = tcell.StyleDefault.Foreground(tcell.NewRGBColor(75, 80, 88))
	styleLineNumberBackground = tcell.NewRGBColor(28, 30, 34)
	styleLineNumberForeground = tcell.NewRGBColor(120, 125, 135)
	styleStatus               = tcell.StyleDefault.
					Foreground(tcell.NewRGBColor(210, 215, 225)).
					Background(tcell.NewRGBColor(43, 47, 54))
	styleStatusError = tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(255, 220, 220)).
				Background(tcell.NewRGBColor(115, 35, 40))
	styleSearchMatch = tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(24, 27, 32)).
				Background(tcell.NewRGBColor(255, 211, 92)).
				Bold(true)
	styleSidebar = tcell.StyleDefault.
			Foreground(tcell.NewRGBColor(190, 196, 208)).
			Background(tcell.NewRGBColor(25, 28, 34))
	styleSidebarHeader = styleSidebar.
				Foreground(tcell.NewRGBColor(232, 236, 245)).
				Background(tcell.NewRGBColor(37, 42, 51)).
				Bold(true)
	styleSidebarDirectory = styleSidebar.Foreground(tcell.NewRGBColor(135, 165, 215))
	styleSidebarActive    = styleSidebar.
				Foreground(tcell.NewRGBColor(255, 255, 255)).
				Background(colorSidebarActive).
				Bold(true)
	styleSidebarViewed   = styleSidebar.Foreground(tcell.NewRGBColor(133, 220, 158)).Bold(true)
	styleSidebarUnviewed = styleSidebar.Foreground(tcell.NewRGBColor(132, 139, 151))
	styleMenu            = tcell.StyleDefault.
				Foreground(tcell.NewRGBColor(210, 215, 225)).
				Background(tcell.NewRGBColor(30, 34, 41))
	styleMenuTitle  = styleMenu.Foreground(tcell.NewRGBColor(238, 242, 255)).Bold(true)
	styleMenuKey    = styleMenu.Foreground(tcell.NewRGBColor(152, 190, 255)).Bold(true)
	styleMenuBorder = styleMenu.Foreground(tcell.NewRGBColor(92, 110, 140))
)

func styleForCell(kind cellKind) tcell.Style {
	switch kind {
	case cellDeletion:
		return styleDeletion
	case cellAddition:
		return styleAddition
	case cellEmpty:
		return styleEmpty
	default:
		return tcell.StyleDefault
	}
}

func fileBannerStyle(active bool) (tcell.Style, tcell.Color) {
	background := colorFileBanner
	if active {
		background = colorFileBannerActive
	}
	return styleFileBanner.Background(background), background
}

func reviewStyle(viewed bool, background tcell.Color) tcell.Style {
	style := tcell.StyleDefault.Foreground(tcell.NewRGBColor(137, 146, 159)).Background(background)
	if viewed {
		style = style.Foreground(tcell.NewRGBColor(133, 211, 157)).Bold(true)
	}
	return style
}

func fileStatusBadgeStyle(status string) tcell.Style {
	style := tcell.StyleDefault.Foreground(tcell.NewRGBColor(24, 28, 34)).Bold(true)
	switch status {
	case "A":
		return style.Background(tcell.NewRGBColor(133, 211, 157))
	case "D":
		return style.Background(tcell.NewRGBColor(231, 139, 145))
	case "R":
		return style.Background(tcell.NewRGBColor(140, 179, 232))
	default:
		return style.Background(tcell.NewRGBColor(226, 190, 113))
	}
}

func sidebarViewedStyle(viewed, active bool) tcell.Style {
	style := styleSidebarUnviewed
	if viewed {
		style = styleSidebarViewed
	}
	if active {
		style = style.Background(colorSidebarActive)
	}
	return style
}

func sidebarStatusStyle(status string, active bool) tcell.Style {
	style := styleSidebar
	if active {
		style = styleSidebarActive
	}
	switch status {
	case "A":
		return style.Foreground(tcell.NewRGBColor(133, 220, 158))
	case "D":
		return style.Foreground(tcell.NewRGBColor(245, 139, 146))
	case "R":
		return style.Foreground(tcell.NewRGBColor(145, 185, 245))
	default:
		return style.Foreground(tcell.NewRGBColor(238, 200, 116))
	}
}
