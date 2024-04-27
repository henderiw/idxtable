package tree

import (
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
)

type Entry interface {
	ID() ID
	Labels() labels.Set
	String() string
	Equal(e2 Entry) bool
}

type entry struct {
	id     ID
	labels labels.Set
}
type Entries []Entry

func (r entry) ID() ID             { return r.id }
func (r entry) Labels() labels.Set { return r.labels }
func (r entry) String() string     { return fmt.Sprintf("id: %d, labels: %s", r.id, r.labels.String()) }
func (r entry) Equal(e2 Entry) bool {
	if r.ID().ID() == e2.ID().ID() &&
		r.ID().Length() == e2.ID().Length() &&
		r.labels.String() == e2.Labels().String() {
		return true
	}
	return false
}

func NewEntry(id ID, labels labels.Set) Entry {
	return entry{
		id:     id.Copy(),
		labels: labels,
	}
}

type Enries[T1 any] []Entry
