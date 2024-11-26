package tree32

import (
	"fmt"
	"sync"

	"github.com/henderiw/idxtable/pkg/tree"
	"github.com/henderiw/idxtable/pkg/tree/gtree"
	"github.com/henderiw/idxtable/pkg/tree/id32"
	"k8s.io/apimachinery/pkg/labels"
)

func New(name string, length uint8) (gtree.GTree, error) {
	if length > id32.IDBitSize {
		return nil, fmt.Errorf("cannot create a tree which bitlength > %d, got: %d", id32.IDBitSize, length)
	}
	return &tree32{
		m:      new(sync.RWMutex),
		tree:   tree.NewTree[tree.Entry](name, id32.IsLeftBitSet, id32.IDBitSize),
		size:   1<<length - 1,
		length: length,
	}, nil
}

type tree32 struct {
	m      *sync.RWMutex
	tree   *tree.Tree[tree.Entry]
	size   uint32
	length uint8
}

func (r *tree32) Clone() gtree.GTree {
	return &tree32{
		m:      new(sync.RWMutex),
		tree:   r.tree.Clone(),
		size:   r.size,
		length: r.length,
	}
}

func (r *tree32) Get(id tree.ID) (tree.Entry, error) {
	r.m.RLock()
	defer r.m.RUnlock()

	iter := r.Iterate()
	for iter.Next() {
		if iter.Entry().ID().ID() == id.ID() &&
			iter.Entry().ID().Length() == id.Length() {
			return iter.Entry(), nil
		}
	}
	return nil, fmt.Errorf("entry %d not found", id)
}

func (r *tree32) Update(id tree.ID, labels labels.Set) error {
	if err := r.validate(id); err != nil {
		return err
	}
	treeEntry := tree.NewEntry(id.Copy(), labels)

	r.m.Lock()
	defer r.m.Unlock()
	return r.set(id, treeEntry)
}

func (r *tree32) ClaimID(id tree.ID, labels labels.Set) error {
	if err := r.validate(id); err != nil {
		return err
	}
	treeEntry := tree.NewEntry(id.Copy(), labels)

	r.m.Lock()
	defer r.m.Unlock()
	return r.set(id, treeEntry)
}

func (r *tree32) ClaimFree(labels labels.Set) (tree.Entry, error) {

	id, err := r.findFree()
	if err != nil {
		return nil, fmt.Errorf("no free ids available, err: %s", err.Error())
	}

	treeId := id32.NewID(uint32(id), id32.IDBitSize)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)
	r.m.Lock()
	defer r.m.Unlock()
	if err := r.set(treeId, treeEntry); err != nil {
		return nil, err
	}
	return treeEntry, nil
}

func (r *tree32) ClaimRange(s string, labels labels.Set) error {
	vlanRange, err := id32.ParseRange(s)
	if err != nil {
		return err
	}
	// TODO check if free

	// get each entry and validate owner

	for _, treeId := range vlanRange.IDs() {
		treeEntry := tree.NewEntry(treeId.Copy(), labels)
		if err := r.set(treeId, treeEntry); err != nil {
			return err
		}
	}
	return nil
}

func (r *tree32) set(id tree.ID, e tree.Entry) error {
	r.tree.Set(id, e)
	return nil
}

func (r *tree32) findFree() (uint32, error) {
	rootID := id32.NewID(0, r.length)
	var bldr id32.IDSetBuilder
	bldr.AddId(rootID)

	for _, e := range r.Children(rootID) {
		bldr.RemoveId(e.ID())
	}
	ipset, err := bldr.IPSet()
	if err != nil {
		return 0, err
	}

	availableID, _, _ := ipset.RemoveFreePrefix(id32.IDBitSize)
	if availableID == nil {
		return 0, fmt.Errorf("no free id available")
	}
	if err := r.validate(availableID); err != nil {
		return 0, err
	}
	return uint32(availableID.ID()), nil
}

func (r *tree32) ReleaseID(id tree.ID) error {
	if err := r.validate(id); err != nil {
		return err
	}
	e, err := r.Get(id)
	if err != nil {
		return nil
	}
	r.m.Lock()
	defer r.m.Unlock()
	return r.del(id, e)
}

func (r *tree32) ReleaseByLabel(selector labels.Selector) error {
	entries := r.GetByLabel(selector)

	r.m.Lock()
	defer r.m.Unlock()

	for _, e := range entries {
		if err := r.del(e.ID().Copy(), e); err != nil {
			return err
		}
	}
	return nil
}

func (r *tree32) del(id tree.ID, e tree.Entry) error {
	matchFunc := func(e1, e2 tree.Entry) bool {
		return e1.Equal(e2)
	}
	r.tree.Delete(id, matchFunc, e)
	return nil
}

func (r *tree32) Children(id tree.ID) tree.Entries {
	entries := tree.Entries{}
	r.m.RLock()
	defer r.m.RUnlock()

	iter := r.Iterate()

	for iter.Next() {
		entry := iter.Entry()
		if entry.ID().Overlaps(id) && entry.ID().Length() > id.Length() {
			entries = append(entries, iter.Entry())
		}

	}
	return entries
}

func (r *tree32) Parents(id tree.ID) tree.Entries {
	entries := tree.Entries{}
	r.m.RLock()
	defer r.m.RUnlock()

	iter := r.Iterate()
	for iter.Next() {
		entry := iter.Entry()
		if entry.ID().Overlaps(id) && entry.ID().Length() < id32.IDBitSize {
			entries = append(entries, iter.Entry())
		}
	}
	return entries
}

func (r *tree32) GetByLabel(selector labels.Selector) tree.Entries {
	entries := tree.Entries{}

	iter := r.Iterate()
	for iter.Next() {
		if selector.Matches(iter.Entry().Labels()) {
			entries = append(entries, iter.Entry())
		}
	}

	return entries
}

func (r *tree32) GetAll() tree.Entries {
	entries := tree.Entries{}

	iter := r.Iterate()
	for iter.Next() {
		entries = append(entries, iter.Entry())
	}

	return entries
}

func (r *tree32) Size() int {
	var size int

	iter := r.Iterate()
	for iter.Next() {
		size++
	}

	return size
}

func (r *tree32) Iterate() *gtree.GTreeIterator {
	r.m.RLock()
	defer r.m.RUnlock()

	return &gtree.GTreeIterator{
		Iter: r.tree.Iterate(),
	}
}

func (r *tree32) validate(id tree.ID) error {
	if id.ID() > uint64(r.size) {
		return fmt.Errorf("max id allowed is %d, got %d", r.size, id.ID())
	}
	if id.Length() < uint8(0) {
		return fmt.Errorf("min allowed length is %d, got %d", r.length, id.Length())
	}
	return nil
}

func (r *tree32) PrintNodes() {
	r.tree.PrintNodes(0)
}

func (r *tree32) PrintValues() {
	r.tree.PrintValues()
}