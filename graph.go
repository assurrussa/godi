package godi

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"go.uber.org/dig"
)

type Graph struct {
	Providers []ProviderNode
	Edges     []ProviderEdge
}

type ProviderNode struct {
	ID          string
	Key         string
	Type        string
	Name        string
	Group       string
	Kind        string
	Constructor string
	File        string
	Line        int
	Provides    []GraphToken
	Requires    []GraphToken
}

type GraphToken struct {
	typ      reflect.Type
	Type     string
	Name     string
	Group    string
	Optional bool
}

type ProviderEdge struct {
	From     string
	To       string
	Type     string
	Name     string
	Group    string
	Optional bool
	Missing  bool
}

type tokenKey struct {
	t     reflect.Type
	name  string
	group string
}

// Graph builds a dependency graph for the container root scope (resolved providers only).
func (c *Container) Graph() Graph {
	graphs := c.GraphModules()
	if graph, ok := graphs["root"]; ok {
		return graph
	}
	return BuildGraph(CollectDependencies(c.dependencies...))
}

// GraphDOT renders the container graph in DOT (Graphviz) format.
func (c *Container) GraphDOT() string {
	return c.Graph().DOT()
}

// GraphModules builds dependency graphs for the root and each module scope.
// Module graphs include the root providers plus module-private providers.
func (c *Container) GraphModules() map[string]Graph {
	rootEntries := buildRootEntries(c.dependencies)
	moduleResolutions, err := buildModuleResolutions(c.modules)
	if err != nil {
		return map[string]Graph{"root": BuildGraph(CollectDependencies(c.dependencies...))}
	}

	globalEntries := buildGlobalEntries(rootEntries, moduleResolutions)
	globalResolution, err := resolveEntries(globalEntries)
	if err != nil {
		return map[string]Graph{"root": BuildGraph(CollectDependencies(c.dependencies...))}
	}

	graphs := map[string]Graph{}
	graphs["root"] = buildGraphFromEntries(globalResolution.providers, globalResolution.decorators)

	for moduleName, res := range moduleResolutions {
		entries := moduleGraphEntries(globalResolution.providers, res.providers)
		resolved, err := resolveEntries(entries)
		decorators := append([]depEntry{}, globalResolution.decorators...)
		decorators = append(decorators, res.decorators...)
		if err != nil {
			providers, extraDecorators := splitEntries(entries)
			graphs[moduleName] = buildGraphFromEntries(providers, append(decorators, extraDecorators...))
			continue
		}
		graphs[moduleName] = buildGraphFromEntries(resolved.providers, decorators)
	}

	return graphs
}

// GraphDOTModules returns DOT graphs for the root and each module scope.
func (c *Container) GraphDOTModules() map[string]string {
	graphs := c.GraphModules()
	dots := map[string]string{}
	for name, graph := range graphs {
		dots[name] = graph.DOT()
	}
	return dots
}

// BuildGraph builds a dependency graph for the provided dependencies (resolved providers only).
func BuildGraph(deps Dependencies) Graph {
	list := deps.List()
	entries := make([]depEntry, 0, len(list))
	for i, dep := range list {
		entries = append(entries, depEntry{dep: dep, idx: i})
	}

	resolved, err := resolveEntries(entries)
	if err != nil {
		providers, decorators := splitEntries(entries)
		return buildGraphFromEntries(providers, decorators)
	}

	return buildGraphFromEntries(resolved.providers, resolved.decorators)
}

func buildGraphFromEntries(providerEntries []depEntry, decoratorEntries []depEntry) Graph {
	providerNodes, baseByToken := buildProviderNodes(providerEntries)
	decoratorNodes, decoratorBySlot := buildDecoratorNodes(decoratorEntries)

	providerNodes = append(providerNodes, decoratorNodes...)

	finalByToken, decoratorPrev := buildDecorationIndex(baseByToken, decoratorBySlot)
	edges := buildGraphEdges(providerNodes, finalByToken, decoratorPrev)

	return Graph{Providers: providerNodes, Edges: edges}
}

func buildProviderNodes(entries []depEntry) ([]ProviderNode, map[tokenKey][]string) {
	nodes := make([]ProviderNode, 0, len(entries))
	baseByToken := map[tokenKey][]string{}

	for _, entry := range entries {
		dep := entry.dep
		if dep.kind == dependencyKindDecorate {
			continue
		}

		node, id := buildNodeFromEntry(entry)
		node.Provides = buildProvideTokens(dep)
		node.Requires = buildRequireTokens(dep)
		nodes = append(nodes, node)

		for _, token := range node.Provides {
			if token.typ == nil {
				continue
			}
			key := tokenKey{t: token.typ, name: token.Name, group: token.Group}
			baseByToken[key] = append(baseByToken[key], id)
		}
	}

	return nodes, baseByToken
}

func buildDecoratorNodes(entries []depEntry) ([]ProviderNode, map[slotKey][]string) {
	nodes := make([]ProviderNode, 0, len(entries))
	decoratorBySlot := map[slotKey][]string{}

	for _, entry := range entries {
		dep := entry.dep
		if dep.kind != dependencyKindDecorate {
			continue
		}

		node, id := buildNodeFromEntry(entry)
		node.Provides = buildDecoratorProvideTokens(dep)
		node.Requires = buildRequireTokens(dep)
		nodes = append(nodes, node)

		slots, err := decoratorSlots(dep)
		if err != nil {
			continue
		}
		for _, slot := range slots {
			decoratorBySlot[slot] = append(decoratorBySlot[slot], id)
		}
	}

	return nodes, decoratorBySlot
}

func buildNodeFromEntry(entry depEntry) (ProviderNode, string) {
	dep := entry.dep
	info := describeProvider(dep, entry.idx)
	id := buildProviderID(dep, info, entry.idx)
	node := ProviderNode{
		ID:          id,
		Key:         derefString(dep.key),
		Type:        depTypeString(dep),
		Name:        derefString(dep.name),
		Group:       depGroup(dep),
		Kind:        dependencyKindString(dep.kind),
		Constructor: info.Constructor,
		File:        info.File,
		Line:        info.Line,
	}
	return node, id
}

func buildDecorationIndex(
	baseByToken map[tokenKey][]string,
	decoratorBySlot map[slotKey][]string,
) (map[tokenKey][]string, map[string]map[slotKey][]string) {
	finalByToken := cloneTokenIndex(baseByToken)
	decoratorPrev := map[string]map[slotKey][]string{}

	for slot, chain := range decoratorBySlot {
		if slot.group != "" {
			continue
		}
		key := tokenKey(slot)
		prev := baseByToken[key]
		for _, decoratorID := range chain {
			if decoratorPrev[decoratorID] == nil {
				decoratorPrev[decoratorID] = map[slotKey][]string{}
			}
			decoratorPrev[decoratorID][slot] = prev
			prev = []string{decoratorID}
		}
		if len(chain) > 0 {
			finalByToken[key] = prev
		}
	}

	return finalByToken, decoratorPrev
}

func cloneTokenIndex(in map[tokenKey][]string) map[tokenKey][]string {
	out := map[tokenKey][]string{}
	for key, ids := range in {
		out[key] = append([]string{}, ids...)
	}
	return out
}

func buildGraphEdges(
	nodes []ProviderNode,
	finalByToken map[tokenKey][]string,
	decoratorPrev map[string]map[slotKey][]string,
) []ProviderEdge {
	edges := make([]ProviderEdge, 0)
	for _, node := range nodes {
		for _, token := range node.Requires {
			if token.typ == nil {
				continue
			}

			targets := resolveTargets(node, token, finalByToken, decoratorPrev)
			if len(targets) == 0 {
				edges = append(edges, ProviderEdge{
					From:     node.ID,
					To:       "",
					Type:     token.Type,
					Name:     token.Name,
					Group:    token.Group,
					Optional: token.Optional,
					Missing:  true,
				})
				continue
			}

			for _, target := range targets {
				edges = append(edges, ProviderEdge{
					From:     node.ID,
					To:       target,
					Type:     token.Type,
					Name:     token.Name,
					Group:    token.Group,
					Optional: token.Optional,
				})
			}
		}
	}
	return edges
}

func resolveTargets(
	node ProviderNode,
	token GraphToken,
	finalByToken map[tokenKey][]string,
	decoratorPrev map[string]map[slotKey][]string,
) []string {
	key := tokenKey{t: token.typ, name: token.Name, group: token.Group}
	targets := finalByToken[key]
	if node.Kind != dependencyKindString(dependencyKindDecorate) {
		return targets
	}

	prev := decoratorPrev[node.ID]
	if prev == nil {
		return targets
	}

	slot := slotKey{t: token.typ, name: token.Name, group: token.Group}
	mapped, ok := prev[slot]
	if !ok {
		return targets
	}
	return mapped
}

func moduleGraphEntries(globalProviders, moduleProviders []depEntry) []depEntry {
	entries := make([]depEntry, 0, len(globalProviders)+len(moduleProviders))
	entries = append(entries, globalProviders...)
	for _, entry := range moduleProviders {
		if entry.dep.private {
			dep := entry.dep
			dep.kind = dependencyKindReplace
			entry.dep = dep
			entries = append(entries, entry)
		}
	}
	return entries
}

// DOT renders a dependency graph as DOT (Graphviz) format.
func (g Graph) DOT() string {
	var b strings.Builder
	_, _ = b.WriteString("digraph DI {\n")
	_, _ = b.WriteString("  rankdir=LR;\n")
	_, _ = b.WriteString("  node [fontname=\"Helvetica\"];\n")

	for _, node := range g.Providers {
		label := buildProviderLabel(node)
		shape := "box"
		style := "solid"
		if node.Kind == dependencyKindString(dependencyKindDecorate) {
			shape = "ellipse"
			style = "dashed"
		} else if node.Kind == dependencyKindString(dependencyKindReplace) {
			style = "bold"
		}
		_, _ = b.WriteString(fmt.Sprintf(
			"  \"%s\" [shape=%s style=%s label=\"%s\"];\n",
			escapeDOT(node.ID),
			shape,
			style,
			escapeDOT(label),
		))
	}

	missingNodes := map[string]string{}
	for _, edge := range g.Edges {
		if !edge.Missing {
			continue
		}
		id := missingNodeID(edge)
		if _, exists := missingNodes[id]; exists {
			continue
		}
		label := buildTokenLabel(GraphToken{
			Type:     edge.Type,
			Name:     edge.Name,
			Group:    edge.Group,
			Optional: edge.Optional,
		})
		missingNodes[id] = label
	}
	for id, label := range missingNodes {
		_, _ = b.WriteString(fmt.Sprintf(
			"  \"%s\" [shape=diamond style=dashed label=\"%s\"];\n",
			escapeDOT(id),
			escapeDOT(label),
		))
	}

	for _, edge := range g.Edges {
		label := buildTokenLabel(GraphToken{
			Type:     edge.Type,
			Name:     edge.Name,
			Group:    edge.Group,
			Optional: edge.Optional,
		})
		target := edge.To
		if edge.Missing {
			target = missingNodeID(edge)
		}
		style := "solid"
		if edge.Optional {
			style = "dashed"
		}
		_, _ = b.WriteString(fmt.Sprintf(
			"  \"%s\" -> \"%s\" [label=\"%s\" style=%s];\n",
			escapeDOT(edge.From),
			escapeDOT(target),
			escapeDOT(label),
			style,
		))
	}

	_, _ = b.WriteString("}\n")
	return b.String()
}

func buildProviderID(dep Dependency, info ProviderInfo, index int) string {
	if dep.key != nil {
		// Keys are often reused across Provide/Replace to keep diagnostics stable,
		// so the ID must include the index to remain unique within the graph.
		return fmt.Sprintf("key:%s#%d", *dep.key, index)
	}
	if info.Constructor != "" {
		return fmt.Sprintf("ctor:%s#%d", info.Constructor, index)
	}
	return fmt.Sprintf("dep:%d", index)
}

func depTypeString(dep Dependency) string {
	if t := dep.Type(); t != nil {
		return t.String()
	}
	return ""
}

func buildProviderLabel(node ProviderNode) string {
	parts := []string{}
	if node.Key != "" {
		parts = append(parts, node.Key)
	}
	if node.Type != "" {
		parts = append(parts, node.Type)
	}
	if node.Constructor != "" {
		parts = append(parts, node.Constructor)
	}
	if node.Name != "" {
		parts = append(parts, "name:"+node.Name)
	}
	if node.Group != "" {
		parts = append(parts, "group:"+node.Group)
	}
	if node.Kind != "" && node.Kind != dependencyKindString(dependencyKindProvide) {
		parts = append(parts, "kind:"+node.Kind)
	}
	if node.File != "" && node.Line > 0 {
		parts = append(parts, fmt.Sprintf("%s:%d", node.File, node.Line))
	}
	return strings.Join(parts, "\\n")
}

func buildTokenLabel(token GraphToken) string {
	parts := []string{}
	if token.Type != "" {
		parts = append(parts, token.Type)
	}
	if token.Name != "" {
		parts = append(parts, "name:"+token.Name)
	}
	if token.Group != "" {
		parts = append(parts, "group:"+token.Group)
	}
	if token.Optional {
		parts = append(parts, "optional")
	}
	return strings.Join(parts, " ")
}

func missingNodeID(edge ProviderEdge) string {
	return fmt.Sprintf("missing:%s|%s|%s", edge.Type, edge.Name, edge.Group)
}

func escapeDOT(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func depGroup(dep Dependency) string {
	if dep.group != nil {
		return *dep.group
	}
	if dep.IsRunnable() {
		return runnableGroup
	}
	return ""
}

func dependencyKindString(kind dependencyKind) string {
	switch kind {
	case dependencyKindReplace:
		return "replace"
	case dependencyKindDecorate:
		return "decorate"
	default:
		return "provide"
	}
}

func splitEntries(entries []depEntry) (providers []depEntry, decorators []depEntry) {
	providers = make([]depEntry, 0, len(entries))
	decorators = make([]depEntry, 0)
	for _, entry := range entries {
		if entry.dep.kind == dependencyKindDecorate {
			decorators = append(decorators, entry)
			continue
		}
		providers = append(providers, entry)
	}
	return providers, decorators
}

func buildProvideTokens(dep Dependency) []GraphToken {
	slots, err := dependencySlots(dep)
	if err != nil {
		return buildProvideTokensFallback(dep)
	}
	if len(slots) == 0 {
		return nil
	}
	result := make([]GraphToken, 0, len(slots))
	for _, slot := range slots {
		if slot.t == nil {
			continue
		}
		result = append(result, GraphToken{
			typ:   slot.t,
			Type:  slot.t.String(),
			Name:  slot.name,
			Group: slot.group,
		})
	}
	return result
}

func buildProvideTokensFallback(dep Dependency) []GraphToken {
	group := depGroup(dep)
	name := derefString(dep.name)
	if group != "" {
		name = ""
	}

	types := dep.ExposedTypes()
	if len(types) == 0 {
		return nil
	}
	result := make([]GraphToken, 0, len(types))
	for _, t := range types {
		if t == nil {
			continue
		}
		result = append(result, GraphToken{
			typ:   t,
			Type:  t.String(),
			Name:  name,
			Group: group,
		})
	}
	return result
}

func buildDecoratorProvideTokens(dep Dependency) []GraphToken {
	slots, err := decoratorSlots(dep)
	if err != nil {
		return nil
	}
	result := make([]GraphToken, 0, len(slots))
	for _, slot := range slots {
		if slot.t == nil {
			continue
		}
		result = append(result, GraphToken{
			typ:   slot.t,
			Type:  slot.t.String(),
			Name:  slot.name,
			Group: slot.group,
		})
	}
	return result
}

func buildRequireTokens(dep Dependency) []GraphToken {
	if dep.constructor == nil {
		return nil
	}
	return parseConstructorInputs(dep.constructor)
}

func parseConstructorInputs(constructor any) []GraphToken {
	fnType := reflect.TypeOf(constructor)
	if fnType == nil || fnType.Kind() != reflect.Func {
		return nil
	}
	var result []GraphToken
	for i := 0; i < fnType.NumIn(); i++ {
		param := fnType.In(i)
		if isDigInStruct(param) {
			result = append(result, parseDigInFields(param)...)
			continue
		}
		result = append(result, GraphToken{typ: param, Type: param.String()})
	}
	return result
}

func isDigInStruct(t reflect.Type) bool {
	if t == nil {
		return false
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	digIn := reflect.TypeOf(dig.In{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous && field.Type == digIn {
			return true
		}
	}
	return false
}

func parseDigInFields(t reflect.Type) []GraphToken {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	result := []GraphToken{}
	digIn := reflect.TypeOf(dig.In{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous && field.Type == digIn {
			continue
		}
		if field.PkgPath != "" {
			continue
		}

		name := field.Tag.Get("name")
		group := field.Tag.Get("group")
		optional := field.Tag.Get("optional") == "true"
		if group != "" {
			name = ""
		}

		fieldType := field.Type
		if group != "" && fieldType.Kind() == reflect.Slice {
			fieldType = fieldType.Elem()
		}

		result = append(result, GraphToken{
			typ:      fieldType,
			Type:     fieldType.String(),
			Name:     name,
			Group:    group,
			Optional: optional,
		})
	}
	return result
}

func (g Graph) ProviderIDs() []string {
	ids := make([]string, 0, len(g.Providers))
	for _, p := range g.Providers {
		ids = append(ids, p.ID)
	}
	sort.Strings(ids)
	return ids
}
