package godi

import (
	"reflect"
	"runtime"
)

type ProviderInfo struct {
	Index       int
	Constructor string
	File        string
	Line        int
	Type        string
	Name        string
	Group       string
}

type OverrideInfo struct {
	Key      string
	Previous ProviderInfo
	Next     ProviderInfo
}

// DetectOverrides reports explicit replacements (godi.Replace) by slot.
func DetectOverrides(deps Dependencies) []OverrideInfo {
	seen := map[slotKey]ProviderInfo{}
	overrides := make([]OverrideInfo, 0)

	for i, dep := range deps.List() {
		if dep.kind == dependencyKindDecorate {
			continue
		}

		slots, err := dependencySlots(dep)
		if err != nil || len(slots) == 0 {
			continue
		}

		info := describeProvider(dep, i)
		for _, slot := range slots {
			if slot.group != "" {
				continue
			}
			if dep.kind == dependencyKindReplace {
				if prev, ok := seen[slot]; ok {
					overrides = append(overrides, OverrideInfo{
						Key:      slotLabel(slot),
						Previous: prev,
						Next:     info,
					})
				}
				seen[slot] = info
				continue
			}
			if _, ok := seen[slot]; !ok {
				seen[slot] = info
			}
		}
	}

	return overrides
}

func describeProvider(dep Dependency, index int) ProviderInfo {
	info := ProviderInfo{Index: index}

	if t := dep.Type(); t != nil {
		info.Type = t.String()
	}

	if dep.name != nil {
		info.Name = *dep.name
	}

	if dep.group != nil {
		info.Group = *dep.group
	}

	if dep.constructor != nil {
		describeEnrichFunc(dep, &info)
	}

	return info
}

func describeEnrichFunc(dep Dependency, info *ProviderInfo) {
	val := reflect.ValueOf(dep.constructor)
	if val.Kind() == reflect.Func {
		pc := val.Pointer()
		if pc != 0 {
			if fn := runtime.FuncForPC(pc); fn != nil {
				info.Constructor = fn.Name()
				file, line := fn.FileLine(pc)
				info.File = file
				info.Line = line
			}
		}
	}
}
