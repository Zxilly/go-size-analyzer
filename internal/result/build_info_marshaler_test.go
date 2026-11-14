//go:build js && wasm

package result

import (
	"encoding/json/v2"
	"runtime/debug"
	"syscall/js"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildInfoJavaScriptPreservesBuildContext(t *testing.T) {
	for _, info := range []*debug.BuildInfo{
		{},
		{GoVersion: "go1.27.0", Path: "example/main", Main: debug.Module{Path: "example", Version: "(devel)"},
			Deps:     []*debug.Module{{Path: "dep", Version: "v1.0.0", Sum: "h1:test", Replace: &debug.Module{Path: "local"}}},
			Settings: []debug.BuildSetting{{Key: "GOARCH", Value: "amd64"}, {Key: "-tags", Value: "embed"}}},
	} {
		native, err := json.Marshal(info)
		require.NoError(t, err)
		browser := js.Global().Get("JSON").Call("stringify", buildInfoJavaScript(info)).String()
		require.JSONEq(t, string(native), browser)
	}
}
