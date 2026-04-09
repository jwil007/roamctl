package tui

import (
	"image/color"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

func makeRSSIColors(min int, max int) []colorStop {
	rnge := max - min
	brk := rnge / 16
	var c colorStop
	var cs []colorStop
	var rssiGradient = []string{
		"#ff0000", // 0  - worst
		"#ff2200",
		"#ff4400",
		"#ff6600",
		"#ff8800",
		"#ffaa00",
		"#ffcc00",
		"#ffee00",
		"#ddff00",
		"#bbff00",
		"#99ff00",
		"#77ff00",
		"#55ff00",
		"#33ff00",
		"#11ff00",
		"#00ff00", // 15 - best
	}
	for i := 0; i < 16; i++ {
		if i == 15 {
			c.threshold = max
		} else {
			c.threshold = min + (i * brk)
		}
		c.color = lipgloss.Color(rssiGradient[i])
		cs = append(cs, c)
	}
	return cs
}

func apTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.Bold(true).Foreground(lipgloss.Color("252"))
	s.Selected = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	return s
}

func currentAPStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffe0"))
}

func (m model) allignCenter() lipgloss.Style {
	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center)
}

func containerStyle(color color.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).Padding(0, 1).Foreground(lipgloss.Color("252"))
}

func (m model) borderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(),
			false, false, true, false).Width(m.width)
}

var baseStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
