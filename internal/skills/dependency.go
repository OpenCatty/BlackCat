package skills

import (
	"fmt"
	"log"
	"sort"
)

// ResolveDependencies returns skills in topological order using Kahn's algorithm.
// Skills whose DependsOn entries reference unknown skills are silently removed
// (a warning is logged). Circular dependencies cause an error with the cycle path.
func ResolveDependencies(skills []Skill) ([]Skill, error) {
	if len(skills) == 0 {
		return []Skill{}, nil
	}

	// 1. Build skillMap for O(1) lookup
	skillMap := make(map[string]Skill, len(skills))
	for _, s := range skills {
		skillMap[s.Name] = s
	}

	// 2. Remove skills with missing dependencies (graceful degradation)
	changed := true
	for changed {
		changed = false
		for name, s := range skillMap {
			for _, dep := range s.DependsOn {
				if _, ok := skillMap[dep]; !ok {
					log.Printf("WARNING: skill %q depends on %q which does not exist; removing %q", name, dep, name)
					delete(skillMap, name)
					changed = true
					break
				}
			}
		}
	}

	if len(skillMap) == 0 {
		return []Skill{}, nil
	}

	// 3. Build in-degree map and adjacency list
	// Edge: dependency -> dependent (dep must come before dependent)
	inDegree := make(map[string]int, len(skillMap))
	adj := make(map[string][]string, len(skillMap))

	for name := range skillMap {
		inDegree[name] = 0
	}

	for name, s := range skillMap {
		for _, dep := range s.DependsOn {
			adj[dep] = append(adj[dep], name)
			inDegree[name]++
		}
	}

	// 4. Kahn's algorithm: enqueue zero-in-degree nodes (sorted for determinism)
	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue)

	var sorted []Skill
	for len(queue) > 0 {
		// Pop first element
		node := queue[0]
		queue = queue[1:]

		sorted = append(sorted, skillMap[node])

		// Gather neighbors that become zero-in-degree
		var newZero []string
		for _, neighbor := range adj[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				newZero = append(newZero, neighbor)
			}
		}

		// Sort new zero-in-degree nodes and merge into queue in sorted order
		if len(newZero) > 0 {
			sort.Strings(newZero)
			queue = mergeSorted(queue, newZero)
		}
	}

	// 5. If not all skills processed, cycle detected
	if len(sorted) != len(skillMap) {
		// Collect remaining skills (those still with in-degree > 0)
		var remaining []Skill
		for name, deg := range inDegree {
			if deg > 0 {
				remaining = append(remaining, skillMap[name])
			}
		}
		cycle := findCycle(remaining)
		return nil, fmt.Errorf("circular dependency detected: %v", cycle)
	}

	return sorted, nil
}

// mergeSorted merges two sorted string slices into one sorted slice.
func mergeSorted(a, b []string) []string {
	result := make([]string, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			result = append(result, a[i])
			i++
		} else {
			result = append(result, b[j])
			j++
		}
	}
	result = append(result, a[i:]...)
	result = append(result, b[j:]...)
	return result
}

// findCycle uses DFS with an inStack tracker to find a cycle among skills.
// Returns the cycle path as a slice of skill names (e.g. ["A", "B", "C", "A"]).
// Self-dependency (A depends on A) is detected as ["A", "A"].
func findCycle(skills []Skill) []string {
	skillMap := make(map[string]Skill, len(skills))
	for _, s := range skills {
		skillMap[s.Name] = s
	}

	visited := make(map[string]bool, len(skills))
	inStack := make(map[string]bool, len(skills))
	parent := make(map[string]string, len(skills))

	// Process in sorted order for determinism
	names := make([]string, 0, len(skills))
	for _, s := range skills {
		names = append(names, s.Name)
	}
	sort.Strings(names)

	var cyclePath []string

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		inStack[node] = true

		s := skillMap[node]
		deps := make([]string, len(s.DependsOn))
		copy(deps, s.DependsOn)
		sort.Strings(deps)

		for _, dep := range deps {
			if _, ok := skillMap[dep]; !ok {
				continue // skip deps not in our set
			}

			if !visited[dep] {
				parent[dep] = node
				if dfs(dep) {
					return true
				}
			} else if inStack[dep] {
				// Found cycle — reconstruct path
				cyclePath = reconstructCycle(parent, node, dep)
				return true
			}
		}

		inStack[node] = false
		return false
	}

	for _, name := range names {
		if !visited[name] {
			if dfs(name) {
				return cyclePath
			}
		}
	}

	return nil
}

// reconstructCycle builds the cycle path from the parent map.
// current is the node that has an edge back to cycleStart.
func reconstructCycle(parent map[string]string, current, cycleStart string) []string {
	// Self-dependency case
	if current == cycleStart {
		return []string{cycleStart, cycleStart}
	}

	// Build path from cycleStart to current by walking parent map backward
	var path []string
	node := current
	for node != cycleStart {
		path = append(path, node)
		node = parent[node]
	}
	path = append(path, cycleStart)

	// Reverse to get cycleStart -> ... -> current
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	// Close the cycle
	path = append(path, cycleStart)
	return path
}
