package tree12

import (
	"fmt"
	"sync"

	"github.com/henderiw/idxtable/pkg/tree"
	"github.com/henderiw/idxtable/pkg/tree/id32"
	"k8s.io/apimachinery/pkg/labels"
)

const addressbitsize = 32

type Tree12 struct {
	m      *sync.RWMutex
	tree   *tree.Tree[tree.Entry]
	size   int
	length uint8
}

type Tree12Iterator struct {
	iter *tree.TreeIterator[tree.Entry]
}

func New(length uint8) *Tree12 {
	return &Tree12{
		m:      new(sync.RWMutex),
		tree:   tree.NewTree[tree.Entry](id32.IsLeftBitSet, addressbitsize),
		size:   1<<length - 1,
		length: length,
	}
}

func (r *Tree12) Clone() *Tree12 {
	return &Tree12{
		m:      new(sync.RWMutex),
		tree:   r.tree.Clone(),
		size:   r.size,
		length: r.length,
	}
}

func (r *Tree12) Get(id tree.ID) (tree.Entry, error) {
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

func (r *Tree12) Update(id tree.ID, labels labels.Set) error {
	if err := r.validate(id); err != nil {
		return err
	}
	treeEntry := tree.NewEntry(id.Copy(), labels)

	r.m.Lock()
	defer r.m.Unlock()
	return r.set(id, treeEntry)
}

func (r *Tree12) Claim(id tree.ID, labels labels.Set) error {
	if err := r.validate(id); err != nil {
		return err
	}
	treeEntry := tree.NewEntry(id.Copy(), labels)

	r.m.Lock()
	defer r.m.Unlock()
	return r.set(id, treeEntry)
}

func (r *Tree12) ClaimFree(labels labels.Set) (tree.Entry, error) {

	id, err := r.findFree()
	if err != nil {
		return nil, fmt.Errorf("no free ids available, err: %s", err.Error())
	}

	treeId := id32.NewID(uint32(id), addressbitsize)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)
	r.m.Lock()
	defer r.m.Unlock()
	if err := r.set(treeId, treeEntry); err != nil {
		return nil, err
	}
	return treeEntry, nil
}

func (r *Tree12) ClaimRange(s string, labels labels.Set) error {
	range12, err := id32.ParseRange(s)
	if err != nil {
		return err
	}
	// TODO check if free

	// get each entry and validate owner

	for _, treeId := range range12.IDs() {
		treeEntry := tree.NewEntry(treeId.Copy(), labels)
		if err := r.set(treeId, treeEntry); err != nil {
			return err
		}
	}
	return nil
}

func (r *Tree12) set(id tree.ID, e tree.Entry) error {
	r.tree.Set(id, e)
	return nil
}

func (r *Tree12) findFree() (uint32, error) {
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

	availableID, _, _ := ipset.RemoveFreePrefix(addressbitsize)
	if availableID == nil {
		return 0, fmt.Errorf("no free id available")
	}
	if err := r.validate(availableID); err != nil {
		return 0, err
	}
	return uint32(availableID.ID()), nil
}

func (r *Tree12) ReleaseID(id tree.ID) error {
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

func (r *Tree12) ReleaseByLabel(selector labels.Selector) error {
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

func (r *Tree12) del(id tree.ID, e tree.Entry) error {
	matchFunc := func(e1, e2 tree.Entry) bool {
		return e1.Equal(e2)
	}
	r.tree.Delete(id, matchFunc, e)
	return nil
}

func (r *Tree12) Children(id tree.ID) tree.Entries {
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

func (r *Tree12) Parents(id tree.ID) tree.Entries {
	entries := tree.Entries{}
	r.m.RLock()
	defer r.m.RUnlock()

	iter := r.Iterate()
	for iter.Next() {
		entry := iter.Entry()
		if entry.ID().Overlaps(id) && entry.ID().Length() < addressbitsize {
			entries = append(entries, iter.Entry())
		}
	}
	return entries
}

func (r *Tree12) GetByLabel(selector labels.Selector) tree.Entries {
	entries := tree.Entries{}

	iter := r.Iterate()
	for iter.Next() {
		if selector.Matches(iter.Entry().Labels()) {
			entries = append(entries, iter.Entry())
		}
	}

	return entries
}

func (r *Tree12) GetAll() tree.Entries {
	entries := tree.Entries{}

	iter := r.Iterate()
	for iter.Next() {
		entries = append(entries, iter.Entry())
	}

	return entries
}

func (r *Tree12) Size() int {
	var size int

	iter := r.Iterate()
	for iter.Next() {
		size++
	}

	return size
}

func (r *Tree12) Iterate() *Tree12Iterator {
	r.m.RLock()
	defer r.m.RUnlock()

	return &Tree12Iterator{
		iter: r.tree.Iterate(),
	}
}

func (r *Tree12) validate(id tree.ID) error {
	if id.ID() > uint64(r.size) {
		return fmt.Errorf("max id allowed is %d, got %d", r.size, id.ID())
	}
	if id.Length() < uint8(r.length) {
		return fmt.Errorf("min allowed length is %d, got %d", r.length, id.Length())
	}
	return nil
}

func (r *Tree12) PrintNodes() {
	r.tree.PrintNodes(0)
}

func (i *Tree12Iterator) Next() bool {
	return i.iter.Next()
}

func (i *Tree12Iterator) Entry() tree.Entry {
	l := i.iter.Vals()
	// we store only 1 entry
	return l[0]
}
