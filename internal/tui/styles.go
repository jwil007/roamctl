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

func apTableStyle() table.Styles {
	s := table.DefaultStyles()
	s.Cell.Align(lipgloss.Center)
	s.Header.Align(lipgloss.Center).Bold(true)
	s.Selected.Align(lipgloss.Center)
	s.Selected = lipgloss.NewStyle()
	return s
}

func (m model) roamTableStyle() table.Styles {
	s := table.DefaultStyles()
	s.Cell.Align(lipgloss.Center)
	s.Header = s.Header.Bold(true).Align(lipgloss.Center)
	s.Selected = lipgloss.NewStyle()
	return s
}

func greenText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Bold(true)
}

func redText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")).Bold(true)
}

func currentAPStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#00aaff"))
}

func (m model) alignCenter() lipgloss.Style {
	return lipgloss.NewStyle().Width(tuiWidth).Align(lipgloss.Center)
}

func (m model) alignLeft() lipgloss.Style {
	return lipgloss.NewStyle().Width(tuiWidth).Align(lipgloss.Left)
}
func (m model) alignRight() lipgloss.Style {
	return lipgloss.NewStyle().Width(tuiWidth).Align(lipgloss.Right)
}

func containerStyle(color color.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(color).Padding(0, 1)
}

func (m model) borderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(),
			false, false, true, false).Width(tuiWidth)
}

var headerStyle = lipgloss.NewStyle().Bold(true)

var titleStyle = lipgloss.NewStyle().Faint(true)
