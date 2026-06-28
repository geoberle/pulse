package workitem

// Flatten converts a tree of WorkItems into a flat slice. Each item's
// ParentID is set to its parent's ID, and Children is cleared. The
// order is depth-first pre-order.
func Flatten(roots []*WorkItem) []WorkItem {
	var out []WorkItem
	var walk func(items []*WorkItem, parentID string)
	walk = func(items []*WorkItem, parentID string) {
		for _, item := range items {
			flat := WorkItem{
				TypeMeta:   item.TypeMeta,
				ObjectMeta: item.ObjectMeta,
				Spec:       item.Spec,
				ParsedSpec: item.ParsedSpec,
			}
			flat.ParentID = parentID
			out = append(out, flat)
			walk(item.Children, item.ID)
		}
	}
	walk(roots, "")
	return out
}

// BuildTree reconstructs a tree from a flat slice of WorkItems using
// ParentID. Items with empty ParentID become roots. Items whose parent
// is not in the slice are also treated as roots.
func BuildTree(items []*WorkItem) []*WorkItem {
	byID := make(map[string]*WorkItem, len(items))
	for _, item := range items {
		cp := *item
		cp.Children = nil
		cp.ParentID = ""
		byID[item.ID] = &cp
	}

	var roots []*WorkItem
	for _, item := range items {
		node := byID[item.ID]
		if item.ParentID == "" {
			roots = append(roots, node)
			continue
		}
		parent, ok := byID[item.ParentID]
		if !ok {
			roots = append(roots, node)
			continue
		}
		parent.Children = append(parent.Children, node)
	}
	return roots
}
