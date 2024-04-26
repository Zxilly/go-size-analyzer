package main

import (
	"github.com/alecthomas/kong"
)

var Options struct {
	Verbose      bool   `help:"Verbose output"`
	Format       string `enum:"text,json,html" default:"text" help:"Output format, possible values: ${enum}"`
	HideProgress bool   `help:"Hide progress bar for disassembly"`

	HideSections bool `help:"Hide sections" group:"text"`
	HideMain     bool `help:"Hide main package" group:"text"`
	HideStd      bool `help:"Hide standard library" group:"text"`

	Indent *int `help:"Indentation for json output" group:"json"`

	Web    bool   `long:"web" help:"use web interface to explore the details" group:"web"`
	Listen string `long:"listen" help:"listen address" default:":8080" group:"web"`
	Open   bool   `long:"open" help:"Open browser" group:"web"`

	Output  string `help:"Write to file"`
	Version bool   `help:"Show version"`

	Binary string `arg:"" name:"file" required:"" help:"Binary file to analyze" type:"existingfile"`
}

var cli = kong.Parse(&Options,
	kong.Name("gsa"),
	kong.Description("A tool for analysing the size of dependencies in compiled Golang binaries, "+
		"providing insight into their impact on the final build."),
	kong.UsageOnError(),
	kong.ConfigureHelp(kong.HelpOptions{
		Compact: true,
		Summary: true,
		Tree:    true,
	}),
	kong.ExplicitGroups([]kong.Group{
		{
			Key:         "text",
			Title:       "Text output options",
			Description: "Options for text output",
		},
		{
			Key:         "json",
			Title:       "Json output options",
			Description: "Options for json output",
		},
		{
			Key:         "web",
			Title:       "Web output options",
			Description: "Options for web output",
		},
	}),
)
