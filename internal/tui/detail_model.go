package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
)

type detailModel struct {
	viewPort  viewport.Model
	renderer  *glamour.TermRenderer
	width     int
	dark      bool
	currentMD string
}

func newDetailModel() detailModel {
	// Default to dark while we wait for the terminal's BackgroundColorMsg
	// reply; SetDark will rebuild if the actual background is light.
	d := detailModel{viewPort: viewport.New(), dark: true}
	d.rebuildRenderer()
	return d
}

// markdownStyle picks the matching glamour style for the terminal background
// and re-tints inline code: the upstream Code defaults (red-on-gray in both
// dark and light themes) render the symbol names we show almost unreadable.
func markdownStyle(dark bool) ansi.StyleConfig {
	var cfg ansi.StyleConfig
	var color string
	if dark {
		cfg = styles.DarkStyleConfig
		color = "117" // light blue on dark
	} else {
		cfg = styles.LightStyleConfig
		color = "27" // deeper blue on light
	}
	cfg.Document.BlockPrefix = ""
	cfg.Document.BlockSuffix = ""
	cfg.Heading.BlockSuffix = ""
	cfg.H1.BlockSuffix = ""
	cfg.H2.BlockSuffix = ""
	cfg.H3.BlockSuffix = ""
	cfg.H4.BlockSuffix = ""
	cfg.H5.BlockSuffix = ""
	cfg.H6.BlockSuffix = ""
	cfg.Code = ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color: &color,
		},
	}
	return cfg
}

func (d *detailModel) rebuildRenderer() {
	wrap := d.width
	if wrap <= 0 {
		wrap = 80
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(markdownStyle(d.dark)),
		glamour.WithWordWrap(wrap),
	)
	if err == nil {
		d.renderer = r
	}
}

func (d *detailModel) SetDark(dark bool) {
	if d.dark == dark {
		return
	}
	d.dark = dark
	d.rebuildRenderer()
	if d.currentMD != "" {
		d.viewPort.SetContent(d.render(d.currentMD))
	}
}

func (d *detailModel) render(md string) string {
	if d.renderer == nil {
		return md
	}
	out, err := d.renderer.Render(md)
	if err != nil {
		return md
	}
	return out
}

func (d *detailModel) SetMarkdown(md string) {
	if md == d.currentMD {
		return
	}
	d.currentMD = md
	d.viewPort.SetContent(d.render(md))
	d.viewPort.GotoTop()
}

func (d *detailModel) SetWidth(w int) {
	if d.width == w {
		d.viewPort.SetWidth(w)
		return
	}
	d.width = w
	d.viewPort.SetWidth(w)
	d.rebuildRenderer()
	if d.currentMD != "" {
		d.viewPort.SetContent(d.render(d.currentMD))
	}
}

func (d *detailModel) SetHeight(h int) {
	d.viewPort.SetHeight(h)
}

func (d detailModel) Update(msg tea.Msg) (detailModel, tea.Cmd) {
	var cmd tea.Cmd
	d.viewPort, cmd = d.viewPort.Update(msg)
	return d, cmd
}

func (d detailModel) View() string {
	return d.viewPort.View()
}

func (d detailModel) KeyMap() []key.Binding {
	km := d.viewPort.KeyMap
	return []key.Binding{
		km.Up,
		km.Down,
		km.PageUp,
		km.PageDown,
		km.HalfPageUp,
		km.HalfPageDown,
	}
}
