package vlantable

import (
	"fmt"
	"sync"

	"github.com/henderiw/idxtable/pkg/tree"
	"github.com/henderiw/idxtable/pkg/tree/id32"
	"k8s.io/apimachinery/pkg/labels"
)

const addressbitsize = 32

type VLANTable struct {
	m    *sync.RWMutex
	tree *tree.Tree[tree.Entry]
	size int
}

type VLANTableIterator struct {
	iter *tree.TreeIterator[tree.Entry]
}

func New() *VLANTable {
	return &VLANTable{
		m:    new(sync.RWMutex),
		tree: tree.NewTree[tree.Entry](id32.IsLeftBitSet),
		size: 4096,
	}
}

func (r *VLANTable) Clone() *VLANTable {
	return &VLANTable{
		m:   new(sync.RWMutex),
		tree: r.tree.Clone(),
		size: 4096,
	}
}

func (r *VLANTable) Get(id uint16) (Entry, error) {
	// returns true if exact match is found.
	r.m.RLock()
	defer r.m.RUnlock()

	iter := r.Iterate()
	for iter.Next() {
		if uint16(iter.Entry().ID().ID()) == id &&
			iter.Entry().ID().Length() == addressbitsize {
			return NewEntry(id, iter.Entry().Labels()), nil
		}
	}
	return nil, fmt.Errorf("entry %d not found", id)
}

func (r *VLANTable) Update(id uint16, labels labels.Set) error {
	if err := r.validate(id); err != nil {
		return err
	}
	treeId := id32.NewID(uint32(id), addressbitsize)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)

	r.m.Lock()
	defer r.m.Unlock()
	return r.set(treeId, treeEntry)
}

func (r *VLANTable) Claim(id uint16, labels labels.Set) error {
	if err := r.validate(id); err != nil {
		return err
	}
	treeId := id32.NewID(uint32(id), addressbitsize)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)

	r.m.Lock()
	defer r.m.Unlock()
	return r.set(treeId, treeEntry)
}

func (r *VLANTable) ClaimFree(labels labels.Set) (Entry, error) {

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
	return NewEntry(id, labels), nil
}

func (r *VLANTable) ClaimRange(s string, labels labels.Set) error {
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

func (r *VLANTable) set(id tree.ID, e tree.Entry) error {
	r.tree.Set(id, e)
	return nil
}

func (r *VLANTable) findFree() (uint16, error) {
	rootID := id32.NewID(0, 20)
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
	if err := r.validate(uint16(availableID.ID())); err != nil {
		return 0, err
	}
	return uint16(availableID.ID()), nil
}

func (r *VLANTable) Release(id uint16) error {
	if err := r.validate(id); err != nil {
		return err
	}
	e, err := r.Get(id)
	if err != nil {
		return nil
	}
	treeId := id32.NewID(uint32(id), addressbitsize)
	treeEntry := tree.NewEntry(treeId.Copy(), e.Labels())
	r.m.Lock()
	defer r.m.Unlock()
	return r.del(treeId, treeEntry)
}

func (r *VLANTable) ReleaseByLabel(selector labels.Selector) error {
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

func (r *VLANTable) del(id tree.ID, e tree.Entry) error {
	matchFunc := func(e1, e2 tree.Entry) bool {
		return e1.Equal(e2)
	}
	r.tree.Delete(id, matchFunc, e)
	return nil
}

func (r *VLANTable) Children(id tree.ID) tree.Entries {
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

func (r *VLANTable) Parents(id uint16) tree.Entries {
	entries := tree.Entries{}
	r.m.RLock()
	defer r.m.RUnlock()

	treeid := id32.NewID(uint32(id), addressbitsize)

	iter := r.Iterate()
	for iter.Next() {
		entry := iter.Entry()
		if entry.ID().Overlaps(treeid) && entry.ID().Length() < addressbitsize {
			entries = append(entries, iter.Entry())
		}
	}
	return entries
}

func (r *VLANTable) GetByLabel(selector labels.Selector) tree.Entries {
	entries := tree.Entries{}

	iter := r.Iterate()
	for iter.Next() {
		if selector.Matches(iter.Entry().Labels()) {
			entries = append(entries, iter.Entry())
		}
	}

	return entries
}

func (r *VLANTable) GetAll() Entries {
	entries := Entries{}

	iter := r.Iterate()
	for iter.Next() {
		entries = append(entries, NewEntry(uint16(iter.Entry().ID().ID()), iter.Entry().Labels()))
	}

	return entries
}

func (r *VLANTable) Iterate() *VLANTableIterator {
	r.m.RLock()
	defer r.m.RUnlock()

	return &VLANTableIterator{
		iter: r.tree.Iterate(),
	}
}

func (r *VLANTable) validate(id uint16) error {
	if id > uint16(r.size-1) {
		return fmt.Errorf("max id allowed is %d, got %d", r.size-1, id)
	}
	return nil
}

func (r *VLANTable) PrintNodes() {
	r.tree.PrintNodes(0)
}

func (i *VLANTableIterator) Next() bool {
	return i.iter.Next()
}

func (i *VLANTableIterator) Entry() tree.Entry {
	l := i.iter.Vals()
	// we store only 1 entry
	return l[0]
}