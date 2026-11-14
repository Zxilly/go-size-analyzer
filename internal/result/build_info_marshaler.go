//go:build js && wasm

package result

import "runtime/debug"

func moduleJavaScript(m *debug.Module) any {
	if m == nil {
		return nil
	}
	value := map[string]any{}
	putString(value, "Path", m.Path)
	putString(value, "Version", m.Version)
	putString(value, "Sum", m.Sum)
	if m.Replace != nil {
		value["Replace"] = moduleJavaScript(m.Replace)
	}
	return value
}

func buildInfoJavaScript(b *debug.BuildInfo) any {
	value := map[string]any{"Main": moduleJavaScript(&b.Main)}
	putString(value, "GoVersion", b.GoVersion)
	putString(value, "Path", b.Path)
	deps := make([]any, 0, len(b.Deps))
	for _, m := range b.Deps {
		deps = append(deps, moduleJavaScript(m))
	}
	settings := make([]any, 0, len(b.Settings))
	for _, s := range b.Settings {
		setting := map[string]any{}
		putString(setting, "Key", s.Key)
		putString(setting, "Value", s.Value)
		settings = append(settings, setting)
	}
	if len(deps) > 0 {
		value["Deps"] = deps
	}
	if len(settings) > 0 {
		value["Settings"] = settings
	}
	return value
}

func putString(value map[string]any, key, text string) {
	if text != "" {
		value[key] = text
	}
}
