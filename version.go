package gsv

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

const (
	unknownVersion  = "(devel)"
	unknownProperty = "N/A"
)

var (
	name       = unknownProperty
	version    = unknownVersion
	commit     = unknownProperty
	buildDate  = unknownProperty
	commitDate = unknownProperty
	dirtyBuild = unknownProperty
)

func SprintVersion() string {
	info, ok := debug.ReadBuildInfo()
	if ok {
		if version == unknownVersion && info.Main.Version != "" {
			version = info.Main.Version
		}

		for _, kv := range info.Settings {
			switch kv.Key {
			case "vcs.revision":
				if commit == unknownProperty && kv.Value != "" {
					commit = kv.Value
				}
			case "vcs.time":
				if commitDate == unknownProperty && kv.Value != "" {
					commitDate = kv.Value
				}
			case "vcs.modified":
				if dirtyBuild == unknownProperty && kv.Value != "" {
					dirtyBuild = kv.Value
				}
			}
		}
	}

	formattedBool := func(b string) string {
		switch b {
		case "true":
			return "yes"
		case "false":
			return "no"
		default:
			return b
		}
	}

	const layout = "2006-01-02 15:04:05"
	buildDateTime, err := time.Parse(time.RFC3339, buildDate)
	if err == nil {
		buildDate = buildDateTime.Format(layout) + " (" + humanize.Time(buildDateTime) + ")"
	}
	commitDateTime, err := time.Parse(time.RFC3339, commitDate)
	if err == nil {
		commitDate = commitDateTime.Format(layout) + " (" + humanize.Time(commitDateTime) + ")"
	}

	s := new(strings.Builder)

	s.WriteString("▓▓▓ gsa\n\n")
	values := map[string]string{
		"Version":     version,
		"Git Commit":  commit,
		"Build Date":  buildDate,
		"Commit Date": commitDate,
		"Dirty Build": formattedBool(dirtyBuild),
		"Go Version":  runtime.Version(),
		"Platform":    fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
	keys := []string{"Version", "Git Commit", "Build Date", "Commit Date", "Dirty Build", "Go Version", "Platform"}

	for _, k := range keys {
		s.WriteString(fmt.Sprintf("  %-11s      %s\n", k, values[k]))
	}

	return s.String()
}
