package godi

import (
	"errors"
	"fmt"
	"reflect"

	"go.uber.org/dig"
)

type depEntry struct {
	dep    Dependency
	idx    int
	module string
}

type slotKey struct {
	t     reflect.Type
	name  string
	group string
}

type entryKey struct {
	module string
	idx    int
}

type resolvedScope struct {
	providers  []depEntry
	decorators []depEntry
	slots      map[slotKey]depEntry
	groupSlots map[slotKey][]depEntry
}

type slotState struct {
	provide    depEntry
	hasProvide bool
	replace    depEntry
	hasReplace bool
	group      []depEntry
}

func resolveEntries(entries []depEntry) (resolvedScope, error) {
	states := map[slotKey]*slotState{}
	decorators := make([]depEntry, 0)

	for i := range entries {
		entry := entries[i]
		if err := classifyEntry(entry, states, &decorators); err != nil {
			return resolvedScope{}, err
		}
	}

	result, selected := buildResolvedScope(states)
	for _, entry := range entries {
		if selected[entryKeyFor(entry)] {
			result.providers = append(result.providers, entry)
		}
	}
	result.decorators = decorators

	return result, nil
}

func classifyEntry(entry depEntry, states map[slotKey]*slotState, decorators *[]depEntry) error {
	dep := entry.dep
	if dep.Error() != nil {
		return fmt.Errorf("dependency error: %w", dep.Error())
	}

	if dep.kind == dependencyKindDecorate {
		if err := validateDecorateDependency(dep); err != nil {
			return err
		}
		*decorators = append(*decorators, entry)
		return nil
	}

	if dependencyGroup(dep) != "" && dep.kind != dependencyKindProvide {
		return errors.New("replace/decorate is not supported for group dependency")
	}

	slots, err := dependencySlots(dep)
	if err != nil {
		return err
	}
	if len(slots) == 0 {
		return errors.New("dependency has no output types")
	}

	for _, slot := range slots {
		state := states[slot]
		if state == nil {
			state = &slotState{}
			states[slot] = state
		}
		if err := applyToState(state, slot, entry); err != nil {
			return err
		}
	}

	return nil
}

func applyToState(state *slotState, slot slotKey, entry depEntry) error {
	if slot.group != "" {
		state.group = append(state.group, entry)
		return nil
	}

	switch entry.dep.kind {
	case dependencyKindProvide:
		if state.hasProvide {
			return fmt.Errorf("duplicate provider for slot %s", slotLabel(slot))
		}
		state.provide = entry
		state.hasProvide = true
	case dependencyKindReplace:
		if state.hasReplace {
			return fmt.Errorf("duplicate replace for slot %s", slotLabel(slot))
		}
		state.replace = entry
		state.hasReplace = true
	default:
		return errors.New("unsupported dependency kind for slot resolution")
	}

	return nil
}

func buildResolvedScope(states map[slotKey]*slotState) (resolvedScope, map[entryKey]bool) {
	result := resolvedScope{
		slots:      map[slotKey]depEntry{},
		groupSlots: map[slotKey][]depEntry{},
	}
	selected := map[entryKey]bool{}

	for slot, state := range states {
		if slot.group != "" {
			for _, entry := range state.group {
				selected[entryKeyFor(entry)] = true
				result.groupSlots[slot] = append(result.groupSlots[slot], entry)
			}
			continue
		}

		var chosen depEntry
		hasChosen := false
		if state.hasReplace {
			chosen = state.replace
			hasChosen = true
		} else if state.hasProvide {
			chosen = state.provide
			hasChosen = true
		}

		if hasChosen {
			selected[entryKeyFor(chosen)] = true
			result.slots[slot] = chosen
		}
	}

	return result, selected
}

func dependencySlots(dep Dependency) ([]slotKey, error) {
	if dep.Error() != nil {
		return nil, dep.Error()
	}

	outSlots, ok, err := provideOutSlotsFromConstructor(dep.constructor)
	if err != nil {
		return nil, err
	}
	if ok {
		if dep.name != nil || dep.group != nil || len(dep.matchingInterfaces) > 0 {
			return nil, errors.New("dig.Out cannot be combined with WithName/WithGroup/WithMatch")
		}
		return dedupeSlots(outSlots), nil
	}

	group := dependencyGroup(dep)
	name := ""
	if dep.name != nil {
		name = *dep.name
	}
	if group != "" {
		name = ""
	}

	types := dep.ExposedTypes()
	if len(types) == 0 {
		return nil, nil
	}

	slots := make([]slotKey, 0, len(types))
	for _, t := range types {
		if t == nil {
			continue
		}
		slots = append(slots, slotKey{t: t, name: name, group: group})
	}

	return dedupeSlots(slots), nil
}

func provideOutSlotsFromConstructor(constructor any) ([]slotKey, bool, error) {
	fnType := reflect.TypeOf(constructor)
	if fnType == nil || fnType.Kind() != reflect.Func {
		return nil, false, nil
	}
	if fnType.NumOut() < 1 {
		return nil, false, nil
	}

	out := fnType.Out(0)
	return parseProvideOutStructSlots(out)
}

func parseProvideOutStructSlots(t reflect.Type) ([]slotKey, bool, error) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, false, nil
	}

	digOut := reflect.TypeOf(dig.Out{})
	hasOut := false
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous && field.Type == digOut {
			hasOut = true
			break
		}
	}
	if !hasOut {
		return nil, false, nil
	}

	slots := []slotKey{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous && field.Type == digOut {
			continue
		}
		if field.PkgPath != "" {
			continue
		}

		name := field.Tag.Get("name")
		group := field.Tag.Get("group")
		if group != "" {
			name = ""
		}

		fieldType := field.Type
		if group != "" && fieldType.Kind() == reflect.Slice {
			fieldType = fieldType.Elem()
		}

		slots = append(slots, slotKey{t: fieldType, name: name, group: group})
	}

	return slots, true, nil
}

func dedupeSlots(slots []slotKey) []slotKey {
	if len(slots) == 0 {
		return slots
	}

	seen := map[slotKey]struct{}{}
	result := make([]slotKey, 0, len(slots))
	for _, slot := range slots {
		if slot.t == nil {
			continue
		}
		if _, ok := seen[slot]; ok {
			continue
		}
		seen[slot] = struct{}{}
		result = append(result, slot)
	}
	return result
}

func decoratorSlots(dep Dependency) ([]slotKey, error) {
	if dep.kind != dependencyKindDecorate {
		return nil, nil
	}

	fnType := reflect.TypeOf(dep.constructor)
	if fnType == nil || fnType.Kind() != reflect.Func {
		return nil, errors.New("decorator must be a function")
	}

	if fnType.NumOut() < 1 || fnType.NumOut() > 2 {
		return nil, errors.New("decorator must return value or (value, error)")
	}
	if fnType.NumOut() == 2 && !fnType.Out(1).AssignableTo(reflect.TypeFor[error]()) {
		return nil, errors.New("decorator second return must be error")
	}

	out := fnType.Out(0)
	outSlots, ok, err := parseOutStructSlots(out)
	if err != nil {
		return nil, err
	}
	if ok {
		return outSlots, nil
	}

	return []slotKey{{t: out}}, nil
}

func parseOutStructSlots(t reflect.Type) ([]slotKey, bool, error) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, false, nil
	}

	digOut := reflect.TypeOf(dig.Out{})
	hasOut := false
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous && field.Type == digOut {
			hasOut = true
			break
		}
	}
	if !hasOut {
		return nil, false, nil
	}

	slots := []slotKey{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous && field.Type == digOut {
			continue
		}
		if field.PkgPath != "" {
			continue
		}
		name := field.Tag.Get("name")
		group := field.Tag.Get("group")
		if group != "" {
			return nil, true, errors.New("decorate does not support group outputs yet (use Provide for groups)")
		}
		slots = append(slots, slotKey{t: field.Type, name: name})
	}

	return slots, true, nil
}

func validateDecorateDependency(dep Dependency) error {
	if dep.kind != dependencyKindDecorate {
		return nil
	}
	if dep.name != nil || dep.group != nil || dep.IsRunnable() {
		return errors.New("decorate does not support WithName/WithGroup; use dig.Out in the decorator result")
	}
	if len(dep.matchingInterfaces) > 0 {
		return errors.New("decorate does not support WithMatch; use dig.Out in the decorator result")
	}
	return nil
}

func dependencyGroup(dep Dependency) string {
	if dep.IsRunnable() {
		return runnableGroup
	}
	if dep.group != nil {
		return *dep.group
	}
	return ""
}

func slotLabel(slot slotKey) string {
	if slot.group != "" {
		return fmt.Sprintf("%s[group=%s]", slot.t, slot.group)
	}
	if slot.name != "" {
		return fmt.Sprintf("%s[name=%s]", slot.t, slot.name)
	}
	return slot.t.String()
}

func sameEntry(a, b depEntry) bool {
	return a.module == b.module && a.idx == b.idx
}

func entryKeyFor(entry depEntry) entryKey {
	return entryKey{module: entry.module, idx: entry.idx}
}
