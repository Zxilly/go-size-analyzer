package diff

import (
	"runtime/debug"
	"slices"
)

type buildChange struct {
	Field string `json:"field"`
	From  string `json:"from"`
	To    string `json:"to"`
}

func compareContext(old, next *commonResult, notes []diagnostic) ([]buildChange, []diagnostic) {
	var changes []buildChange
	if old.BuildInfo == nil || next.BuildInfo == nil {
		notes = append(notes, diagnostic{Code: "build_info_unavailable", Message: "Build information is missing from at least one report; target toolchain and build settings cannot be fully compared."})
	} else {
		if old.BuildInfo.GoVersion == "" || next.BuildInfo.GoVersion == "" {
			notes = append(notes, diagnostic{Code: "build_info_incomplete", Message: "Build information is present but at least one target Go version is unknown; matching recorded fields does not establish toolchain comparability."})
		}
		a, b := buildFields(old.BuildInfo), buildFields(next.BuildInfo)
		keys := map[string]bool{}
		for k := range a {
			keys[k] = true
		}
		for k := range b {
			keys[k] = true
		}
		ordered := make([]string, 0, len(keys))
		for k := range keys {
			ordered = append(ordered, k)
		}
		slices.Sort(ordered)
		for _, k := range ordered {
			from, oldOK := a[k]
			to, newOK := b[k]
			if from == to && oldOK == newOK {
				continue
			}
			if !oldOK {
				from = "<not recorded>"
			}
			if !newOK {
				to = "<not recorded>"
			}
			changes = append(changes, buildChange{Field: k, From: from, To: to})
		}
		if len(changes) > 0 {
			notes = append(notes, diagnostic{Code: "build_context_changed", Message: "Target build information changed; size differences may reflect toolchain, configuration, dependency or source changes. See build_changes."})
		}
	}
	if !requireAnalyzeModeSame(old, next) {
		notes = append(notes, diagnostic{Code: "analyzers_changed", Message: "Available analyzers differ between reports; attribution changes may reflect analysis coverage."})
	}
	if old.Coverage == nil || next.Coverage == nil {
		notes = append(notes, diagnostic{Code: "coverage_version_unavailable", Message: "Coverage accounting metadata is missing from at least one report."})
	} else if old.Coverage.Version != next.Coverage.Version {
		notes = append(notes, diagnostic{Code: "coverage_version_changed", Message: "Coverage accounting versions differ; attributed sizes may use different accounting rules."})
	}
	return changes, notes
}

func buildFields(info *debug.BuildInfo) map[string]string {
	fields := map[string]string{"go_version": info.GoVersion, "main_path": info.Path, "main_module": moduleIdentity(&info.Main)}
	for _, setting := range info.Settings {
		fields["setting:"+setting.Key] = setting.Value
	}
	for _, dep := range info.Deps {
		if dep != nil {
			fields["module:"+dep.Path] = moduleIdentity(dep)
		}
	}
	return fields
}

func moduleIdentity(module *debug.Module) string {
	value := module.Path + "@" + module.Version
	if module.Sum != "" {
		value += " " + module.Sum
	}
	if module.Replace != nil {
		value += " => " + moduleIdentity(module.Replace)
	}
	return value
}
