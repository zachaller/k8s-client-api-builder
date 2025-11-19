package hydrator

import (
	"fmt"
	"strings"
)

// DependencyGraph represents resource dependencies
type DependencyGraph map[string][]string

// buildDependencyGraph builds a dependency graph from resources
func (h *Hydrator) buildDependencyGraph(resources []map[string]interface{}) (DependencyGraph, error) {
	graph := make(DependencyGraph)

	for _, resource := range resources {
		key, err := getResourceKey(resource)
		if err != nil {
			continue // Skip resources without proper metadata
		}

		// Find all resource references in this resource
		refs := h.findResourceReferences(resource)
		graph[key] = refs
	}

	return graph, nil
}

// findResourceReferences finds all resource references in a value
func (h *Hydrator) findResourceReferences(value interface{}) []string {
	refs := []string{}

	switch v := value.(type) {
	case string:
		// Check if string contains resource() calls
		extracted := extractResourceRefsFromString(v)
		refs = append(refs, extracted...)

	case map[string]interface{}:
		// Recursively search map
		for _, val := range v {
			childRefs := h.findResourceReferences(val)
			refs = append(refs, childRefs...)
		}

	case []interface{}:
		// Recursively search slice
		for _, item := range v {
			childRefs := h.findResourceReferences(item)
			refs = append(refs, childRefs...)
		}
	}

	return refs
}

// extractResourceRefsFromString extracts resource references from a string
func extractResourceRefsFromString(s string) []string {
	refs := []string{}

	// Find all resource() calls
	for {
		start := strings.Index(s, "resource(")
		if start == -1 {
			break
		}

		// Find matching closing parenthesis
		depth := 0
		end := -1
		for i := start + len("resource"); i < len(s); i++ {
			if s[i] == '(' {
				depth++
			} else if s[i] == ')' {
				depth--
				if depth == 0 {
					end = i
					break
				}
			}
		}

		if end == -1 {
			break
		}

		// Extract the resource reference
		refStr := s[start : end+1]

		// Parse to get apiVersion, kind, name
		// Simple extraction - just get the key
		if key := parseResourceKey(refStr); key != "" {
			refs = append(refs, key)
		}

		// Continue searching after this reference
		s = s[end+1:]
	}

	return refs
}

// parseResourceKey extracts the resource key from a resource() call
func parseResourceKey(refStr string) string {
	// Extract arguments from resource("apiVersion", "kind", "name")
	start := strings.Index(refStr, "(")
	end := strings.LastIndex(refStr, ")")
	if start == -1 || end == -1 {
		return ""
	}

	argsStr := refStr[start+1 : end]

	// Simple parsing - split by comma and extract quoted strings
	parts := strings.Split(argsStr, ",")
	if len(parts) < 3 {
		return ""
	}

	apiVersion := strings.Trim(strings.TrimSpace(parts[0]), "\"")
	kind := strings.Trim(strings.TrimSpace(parts[1]), "\"")

	// Name might be an expression, try to extract if it's a simple string
	namePart := strings.TrimSpace(parts[2])
	if strings.HasPrefix(namePart, "\"") && strings.HasSuffix(namePart, "\"") {
		name := strings.Trim(namePart, "\"")
		return fmt.Sprintf("%s/%s/%s", apiVersion, kind, name)
	}

	// If name is an expression, we can't determine the key statically
	// Return a partial key for now
	return fmt.Sprintf("%s/%s/*", apiVersion, kind)
}

// detectCircularReferences detects circular dependencies in the graph
func detectCircularReferences(graph DependencyGraph) []string {
	cycles := []string{}
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var detectCycle func(node string, path []string) bool
	detectCycle = func(node string, path []string) bool {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range graph[node] {
			// Skip wildcard references (can't determine statically)
			if strings.HasSuffix(neighbor, "/*") {
				continue
			}

			if !visited[neighbor] {
				if detectCycle(neighbor, path) {
					return true
				}
			} else if recStack[neighbor] {
				// Found a cycle
				cycleStart := -1
				for i, n := range path {
					if n == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart != -1 {
					cyclePath := append(path[cycleStart:], neighbor)
					cycles = append(cycles, strings.Join(cyclePath, " -> "))
				}
				return true
			}
		}

		recStack[node] = false
		return false
	}

	// Check each node
	for node := range graph {
		if !visited[node] {
			detectCycle(node, []string{})
		}
	}

	return cycles
}

// getResourceKey builds a key from a resource
func getResourceKey(resource map[string]interface{}) (string, error) {
	apiVersion, ok := resource["apiVersion"].(string)
	if !ok {
		return "", fmt.Errorf("missing apiVersion")
	}

	kind, ok := resource["kind"].(string)
	if !ok {
		return "", fmt.Errorf("missing kind")
	}

	metadata, ok := resource["metadata"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("missing metadata")
	}

	name, ok := metadata["name"].(string)
	if !ok {
		return "", fmt.Errorf("missing name")
	}

	return fmt.Sprintf("%s/%s/%s", apiVersion, kind, name), nil
}
