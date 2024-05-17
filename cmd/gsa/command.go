package main

import (
	"github.com/alecthomas/kong"

	gsv "github.com/Zxilly/go-size-analyzer"
	"github.com/Zxilly/go-size-analyzer/internal/webui"
)

var Options struct {
	Verbose bool   `help:"Verbose output"`
	Format  string `short:"f" enum:"text,json,html,svg" default:"text" help:"Output format, possible values: ${enum}"`

	NoDisasm bool `help:"Skip disassembly pass"`
	NoSymbol bool `help:"Skip symbol pass"`

	HideSections bool `help:"Hide sections" group:"text"`
	HideMain     bool `help:"Hide main package" group:"text"`
	HideStd      bool `help:"Hide standard library" group:"text"`

	Indent  *int `help:"Indentation for json output" group:"json"`
	Compact bool `help:"Hide function details, replacement with size" group:"json"`

	Width       int `help:"Width of the svg treemap" default:"1028" group:"svg"`
	Height      int `help:"Height of the svg treemap" default:"640" group:"svg"`
	MarginBox   int `help:"Margin between boxes" default:"4" group:"svg"`
	PaddingBox  int `help:"Padding between box border and content" default:"4" group:"svg"`
	PaddingRoot int `help:"Padding around root content" default:"32" group:"svg"`

	Web         bool                  `long:"web" help:"use web interface to explore the details" group:"web"`
	Listen      string                `long:"listen" help:"listen address" default:":8080" group:"web"`
	Open        bool                  `long:"open" help:"Open browser" group:"web"`
	UpdateCache webui.UpdateCacheFlag `long:"update-cache" help:"Update the cache file for the web UI" group:"web"`

	Tui bool `long:"tui" help:"use terminal interface to explore the details" group:"tui"`

	Output string `short:"o" help:"Write to file"`

	Version kong.VersionFlag `help:"Show version"`

	Binary string `arg:"" name:"file" required:"" help:"Binary file to analyze" type:"existingfile"`
}

func init() {
	kong.Parse(&Options,
		kong.Name("gsa"),
		kong.Description("A tool for analyzing the size of dependencies in compiled Golang binaries, "+
			"providing insight into their impact on the final build."),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Summary: true,
		}),
		kong.ExplicitGroups([]kong.Group{
			{
				Key:   "text",
				Title: "Text output options",
			},
			{
				Key:   "json",
				Title: "Json output options",
			},
			{
				Key:   "web",
				Title: "Web interface options",
			},
			{
				Key:   "svg",
				Title: "Svg output options",
			},
			{
				Key:   "tui",
				Title: "Terminal interface options",
			},
		}),
		kong.Vars{
			"version": gsv.SprintVersion(),
		},
		kong.PostBuild(func(k *kong.Kong) error {
			_, showCache := any(webui.UpdateCacheFlag(true)).(interface {
				BeforeReset(*kong.Kong, kong.Vars) error
			})
			for _, f := range k.Model.Flags {
				if f.Name == "update-cache" {
					f.Hidden = !showCache
				}
			}
			return nil
		}),
	)
}
