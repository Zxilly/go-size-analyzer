package tool

import (
	"github.com/schollz/progressbar/v3"
	"os"
	"time"
)

func NewPb(max int64, desc string) *progressbar.ProgressBar {
	return progressbar.NewOptions64(
		max,
		progressbar.OptionSetDescription(desc),
		progressbar.OptionSetWriter(os.Stdout),
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
