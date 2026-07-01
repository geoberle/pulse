package workitem

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// Flatten converts a tree of WorkItems into a flat slice. Each item is
// deep-copied so the result shares no references with the input.
// OwnerReferences are set to the parent's identity, Children are cleared.
// Order is depth-first pre-order.
func Flatten(roots []*WorkItem) []WorkItem {
	var out []WorkItem
	var walk func(items []*WorkItem, owner *metav1.OwnerReference)
	walk = func(items []*WorkItem, owner *metav1.OwnerReference) {
		for _, item := range items {
			cp := item.DeepCopy()
			if owner != nil {
				cp.OwnerReferences = []metav1.OwnerReference{*owner}
			} else {
				cp.OwnerReferences = nil
			}
			cp.Children = nil
			out = append(out, *cp)
			walk(item.Children, &metav1.OwnerReference{
				APIVersion: APIVersion,
				Kind:       item.Kind,
				Name:       item.Name,
			})
		}
	}
	walk(roots, nil)
	return out
}

// BuildTree reconstructs a tree from a flat slice of WorkItems using
// OwnerReferences. Items with no OwnerReferences become roots. Items
// whose parent is not in the slice are also treated as roots.
// Output items have Children populated and OwnerReferences cleared —
// tree shape is expressed via Children, OwnerReferences are only for
// flat representations in the informer cache.
func BuildTree(items []*WorkItem) []*WorkItem {
	byName := make(map[string]*WorkItem, len(items))
	for _, item := range items {
		cp := item.DeepCopy()
		cp.Children = nil
		cp.OwnerReferences = nil
		byName[item.Name] = cp
	}

	var roots []*WorkItem
	for _, item := range items {
		node := byName[item.Name]
		parentName := item.ParentName()
		if parentName == "" {
			roots = append(roots, node)
			continue
		}
		parent, ok := byName[parentName]
		if !ok {
			roots = append(roots, node)
			continue
		}
		parent.Children = append(parent.Children, node)
	}
	return roots
}
