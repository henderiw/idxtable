package table32

import (
	"fmt"

	"github.com/henderiw/idxtable/pkg/idxtable"
	"github.com/henderiw/idxtable/pkg/tree"
	"github.com/henderiw/idxtable/pkg/tree/id32"
	"k8s.io/apimachinery/pkg/labels"
)

func New(start, end uint32) *Table32 {
	return &Table32{
		table: idxtable.NewTable[tree.Entry](
			int64(end - start + 1),
		),
		start: start,
		end:   end,
	}
}

type Table32 struct {
	table idxtable.Table[tree.Entry]
	start uint32
	end   uint32
}

func (r *Table32) Get(id uint32) (tree.Entry, error) {
	var entry tree.Entry
	// Validate input
	if err := r.validateID(id); err != nil {
		return entry, err
	}
	newid := calculateIndex(id, r.start)
	e, err := r.table.Get(newid)
	if err != nil {
		return entry, err
	}
	return e.Data(), nil
}

func (r *Table32) Claim(id uint32, labels labels.Set) error {
	// Validate input
	if err := r.validateID(id); err != nil {
		return err
	}
	newid := calculateIndex(id, r.start)
	if !r.table.IsFree(newid) {
		return fmt.Errorf("claim failed id %d already claimed", calculateIDFromIndex(r.start, newid))
	}

	treeId := id32.NewID(uint32(newid), 32)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)
	return r.table.Claim(newid, treeEntry)
}

func (r *Table32) ClaimFree(labels labels.Set) (tree.Entry, error) {
	// Validate input

	id, err := r.FindFree()
	if err != nil {
		return nil, err
	}
	if err := r.Claim(id, labels); err != nil {
		return nil, err
	}
	treeId := id32.NewID(uint32(id), 32)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)
	return treeEntry, nil
}

func (r *Table32) Release(id uint32) error {
	// Validate input
	if err := r.validateID(id); err != nil {
		return err
	}
	newid := calculateIndex(id, r.start)
	return r.table.Release(newid)
}

func (r *Table32) Update(id uint32, labels labels.Set) error {
	// Validate input
	if err := r.validateID(id); err != nil {
		return err
	}
	newid := calculateIndex(id, r.start)
	treeId := id32.NewID(uint32(newid), 32)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)
	return r.table.Update(newid, treeEntry)
}

func (r *Table32) Size() int {
	return r.table.Size()
}

func (r *Table32) Has(id uint32) bool {
	// Validate IP address
	if err := r.validateID(id); err != nil {
		return false
	}
	newid := calculateIndex(id, r.start)
	return r.table.Has(newid)
}

func (r *Table32) IsFree(id uint32) bool {
	// Validate IP address
	if err := r.validateID(id); err != nil {
		return false
	}
	newid := calculateIndex(id, r.start)
	return r.table.IsFree(newid)
}

func (r *Table32) FindFree() (uint32, error) {
	id, err := r.table.FindFree()
	if err != nil {
		return 0, err
	}
	return calculateIDFromIndex(r.start, id), nil
}

func (r *Table32) GetAll() tree.Entries {
	entries := make(tree.Entries, 0, r.table.Size())
	for _, entry := range r.table.GetAll() {
		// need to remap the id for the outside world
		entry := tree.NewEntry(id32.NewID(uint32(calculateIDFromIndex(r.start, entry.ID())), 32), entry.Data().Labels())

		entries = append(entries, entry)
	}
	return entries
}

func (r *Table32) GetByLabel(selector labels.Selector) tree.Entries {
	entries := make(tree.Entries, 0, r.table.Size())

	iter := r.table.Iterate()

	for iter.Next() {
		entry := iter.Value().Data()
		if selector.Matches(entry.Labels()) {
			// need to remap the id for the outside world
			entry := tree.NewEntry(id32.NewID(uint32(calculateIDFromIndex(r.start, int64(entry.ID().ID()))), 32), entry.Labels())
			entries = append(entries, entry)
		}
	}
	return entries
}

func (r *Table32) validateID(id uint32) error {

	if id < r.start {
		return fmt.Errorf("id %d, does not fit in the range from %d to %d", id, r.start, r.end)
	}
	if id > r.end {
		return fmt.Errorf("id %d, does not fit in the range from %d to %d", id, r.start, r.end)
	}
	return nil
}

func calculateIndex(id, start uint32) int64 {
	// Calculate the index in the bitmap
	return int64(id - start)
}

func calculateIDFromIndex(start uint32, id int64) uint32 {
	return start + uint32(id)
}
