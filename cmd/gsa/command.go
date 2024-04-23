package main

import (
	"github.com/ZxillyFork/go-flags"
)

type Options struct {
	Verbose bool   `long:"verbose" description:"Verbose output"`
	Format  string `short:"f" long:"format" description:"Output format" choice:"text" choice:"json" choice:"html"`

	TextOptions struct {
		HideSections bool `long:"hide-sections" description:"Hide sections"`
		HideMain     bool `long:"hide-main" description:"Hide main package"`
		HideStd      bool `long:"hide-std" description:"Hide standard library"`
	} `group:"Text Options"`

	JsonOptions struct {
		Indent *int `long:"indent" description:"Indentation for json output"`
	} `group:"Json Options"`

	HtmlOptions struct {
		Web    bool   `long:"web" description:"Start web server for html output, this option will override format to html and ignore output option"`
		Listen string `long:"listen" description:"Listen address" default:":8080"`
		Open   bool   `long:"open" description:"Open browser"`
	} `group:"Html Options"`

	Output  string `short:"o" long:"output" description:"Write to file"`
	Version bool   `long:"version" description:"Show version"`

	Arg struct {
		Binary string `positional-arg-name:"file" description:"Binary file to analyze"`
	} `positional-args:"yes"`
}

var options Options
var parser = flags.NewParser(&options, flags.Default)
