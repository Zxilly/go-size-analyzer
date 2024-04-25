package utils

import (
	"github.com/schollz/progressbar/v3"
	"io"
	"time"
)

func NewPb(max int64, desc string, disable bool) *progressbar.ProgressBar {
	var out io.Writer = Stdout
	if disable {
		out = io.Discard
	}

	return progressbar.NewOptions64(
		max,
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetWriter(out),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetItsString("functions"),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)
}
