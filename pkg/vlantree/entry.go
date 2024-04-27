package vlantree

import "k8s.io/apimachinery/pkg/labels"

type Enries []Entry

type Entry interface {
	ID() uint16
	Labels() labels.Set
}

func NewEntry(id uint16, labels labels.Set) Entry {
	return entry{
		id:     id,
		labels: labels,
	}
}

type entry struct {
	id     uint16
	labels labels.Set
}
type Entries []Entry

func (r entry) ID() uint16         { return r.id }
func (r entry) Labels() labels.Set { return r.labels }
