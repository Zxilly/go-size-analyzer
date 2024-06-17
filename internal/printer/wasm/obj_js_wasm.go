//go:build js && wasm

package wasm

import (
	"syscall/js"

	"github.com/Zxilly/go-size-analyzer/internal/result"
)

func JavaScript(r *result.Result) js.Value {
	ret := r.MarshalJavaScript()

	return js.ValueOf(ret)
}
