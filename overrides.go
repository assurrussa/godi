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
// It is independent of list order. Invalid dependency sets return no report;
// use NewContainer or Provide to obtain their configuration errors.
func DetectOverrides(deps Dependencies) []OverrideInfo {
	entries := buildRootEntries(deps.List())
	resolved, err := resolveEntries(entries)
	if err != nil {
		return nil
	}
	var overrides []OverrideInfo
	for _, entry := range entries {
		if entry.dep.kind != dependencyKindProvide {
			continue
		}
		slots, err := dependencySlots(entry.dep)
		if err != nil {
			continue
		}
		for _, slot := range slots {
			winner, ok := resolved.slots[slot]
			if !ok || winner.dep.kind != dependencyKindReplace {
				continue
			}
			overrides = append(overrides, OverrideInfo{
				Key:      slotLabel(slot),
				Previous: describeProvider(entry.dep, entry.idx),
				Next:     describeProvider(winner.dep, winner.idx),
			})
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
