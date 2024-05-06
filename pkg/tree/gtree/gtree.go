package gtree

import (
	"github.com/henderiw/idxtable/pkg/tree"
	"k8s.io/apimachinery/pkg/labels"
)



type GTree interface {
	Clone() GTree
	Get(id tree.ID) (tree.Entry, error)
	Update(id tree.ID, labels labels.Set) error
	ClaimID(id tree.ID, labels labels.Set) error
	ClaimFree(labels labels.Set) (tree.Entry, error)
	ClaimRange(s string, labels labels.Set) error
	ReleaseID(id tree.ID) error
	ReleaseByLabel(selector labels.Selector) error
	Children(id tree.ID) tree.Entries
	Parents(id tree.ID) tree.Entries
	GetByLabel(selector labels.Selector) tree.Entries
	GetAll() tree.Entries
	Size() int 
	Iterate() *GTreeIterator
	PrintNodes()
}

type GTreeIterator struct {
	Iter *tree.TreeIterator[tree.Entry]
}

func (i *GTreeIterator) Next() bool {
	return i.Iter.Next()
}

func (i *GTreeIterator) Entry() tree.Entry {
	l := i.Iter.Vals()
	// we store only 1 entry
	return l[0]
}