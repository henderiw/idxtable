package tree32

import (
	"fmt"
	"sync"

	"github.com/henderiw/idxtable/pkg/tree"
	"github.com/henderiw/idxtable/pkg/tree/id32"
	"k8s.io/apimachinery/pkg/labels"
)

const addressbitsize = 32

type Tree32 struct {
	m      *sync.RWMutex
	tree   *tree.Tree[tree.Entry]
	size   int
	length uint8
}

type Tree32Iterator struct {
	iter *tree.TreeIterator[tree.Entry]
}

func New(size int, length uint8) *Tree32 {
	return &Tree32{
		m:      new(sync.RWMutex),
		tree:   tree.NewTree[tree.Entry](id32.IsLeftBitSet),
		size:   size,
		length: length,
	}
}

func (r *Tree32) Clone() *Tree32 {
	return &Tree32{
		m:      new(sync.RWMutex),
		tree:   r.tree.Clone(),
		size:   r.size,
		length: r.length,
	}
}

func (r *Tree32) Get(id tree.ID) (tree.Entry, error) {
	r.m.RLock()
	defer r.m.RUnlock()

	iter := r.Iterate()
	for iter.Next() {
		if uint16(iter.Entry().ID().ID()) == uint16(id.ID()) &&
			iter.Entry().ID().Length() == id.Length() {
			return iter.Entry(), nil
		}
	}
	return nil, fmt.Errorf("entry %d not found", id)
}

func (r *Tree32) Update(id tree.ID, labels labels.Set) error {
	if err := r.validate(id); err != nil {
		return err
	}
	treeEntry := tree.NewEntry(id.Copy(), labels)

	r.m.Lock()
	defer r.m.Unlock()
	return r.set(id, treeEntry)
}

func (r *Tree32) Claim(id tree.ID, labels labels.Set) error {
	if err := r.validate(id); err != nil {
		return err
	}
	treeEntry := tree.NewEntry(id.Copy(), labels)

	r.m.Lock()
	defer r.m.Unlock()
	return r.set(id, treeEntry)
}

func (r *Tree32) ClaimFree(labels labels.Set) (tree.Entry, error) {

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

func (r *Tree32) ClaimRange(s string, labels labels.Set) error {
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

func (r *Tree32) set(id tree.ID, e tree.Entry) error {
	r.tree.Set(id, e)
	return nil
}

func (r *Tree32) findFree() (uint16, error) {
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
	return uint16(availableID.ID()), nil
}

func (r *Tree32) ReleaseID(id tree.ID) error {
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

func (r *Tree32) ReleaseByLabel(selector labels.Selector) error {
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

func (r *Tree32) del(id tree.ID, e tree.Entry) error {
	matchFunc := func(e1, e2 tree.Entry) bool {
		return e1.Equal(e2)
	}
	r.tree.Delete(id, matchFunc, e)
	return nil
}

func (r *Tree32) Children(id tree.ID) tree.Entries {
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

func (r *Tree32) Parents(id tree.ID) tree.Entries {
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

func (r *Tree32) GetByLabel(selector labels.Selector) tree.Entries {
	entries := tree.Entries{}

	iter := r.Iterate()
	for iter.Next() {
		if selector.Matches(iter.Entry().Labels()) {
			entries = append(entries, iter.Entry())
		}
	}

	return entries
}

func (r *Tree32) GetAll() tree.Entries {
	entries := tree.Entries{}

	iter := r.Iterate()
	for iter.Next() {
		entries = append(entries, iter.Entry())
	}

	return entries
}

func (r *Tree32) Size() int {
	var size int

	iter := r.Iterate()
	for iter.Next() {
		size++
	}

	return size
}

func (r *Tree32) Iterate() *Tree32Iterator {
	r.m.RLock()
	defer r.m.RUnlock()

	return &Tree32Iterator{
		iter: r.tree.Iterate(),
	}
}

func (r *Tree32) validate(id tree.ID) error {
	if id.ID() > uint64(r.size-1) {
		return fmt.Errorf("max id allowed is %d, got %d", r.size-1, id.ID())
	}
	if id.Length() < uint8(0) {
		return fmt.Errorf("min allowed length is %d, got %d", r.length, id.Length())
	}
	return nil
}

func (r *Tree32) PrintNodes() {
	r.tree.PrintNodes(0)
}

func (i *Tree32Iterator) Next() bool {
	return i.iter.Next()
}

func (i *Tree32Iterator) Entry() tree.Entry {
	l := i.iter.Vals()
	// we store only 1 entry
	return l[0]
}
