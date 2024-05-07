package table64

import (
	"fmt"

	"github.com/henderiw/idxtable/pkg/idxtable"
	"github.com/henderiw/idxtable/pkg/table"
	"github.com/henderiw/idxtable/pkg/tree"
	"github.com/henderiw/idxtable/pkg/tree/id64"
	"k8s.io/apimachinery/pkg/labels"
)

func New(start, end uint64) table.Table {
	return &table64{
		table: idxtable.NewTable[tree.Entry](
			uint64(end - start + 1),
		),
		start: start,
		end:   end,
	}
}

type table64 struct {
	table idxtable.Table[tree.Entry]
	start uint64
	end   uint64
}

func (r *table64) Get(id uint64) (tree.Entry, error) {
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

func (r *table64) Claim(id uint64, labels labels.Set) error {
	// Validate input
	if err := r.validateID(id); err != nil {
		return err
	}
	newid := calculateIndex(id, r.start)
	if !r.table.IsFree(newid) {
		return fmt.Errorf("claim failed id %d already claimed", calculateIDFromIndex(r.start, newid))
	}

	treeId := id64.NewID(uint64(newid), id64.IDBitSize)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)
	return r.table.Claim(newid, treeEntry)
}

func (r *table64) ClaimFree(labels labels.Set) (tree.Entry, error) {
	// Validate input

	id, err := r.FindFree()
	if err != nil {
		return nil, err
	}
	if err := r.Claim(id, labels); err != nil {
		return nil, err
	}
	treeId := id64.NewID(uint64(id), id64.IDBitSize)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)
	return treeEntry, nil
}

func (r *table64) Release(id uint64) error {
	// Validate input
	if err := r.validateID(id); err != nil {
		return err
	}
	newid := calculateIndex(id, r.start)
	return r.table.Release(newid)
}

func (r *table64) Update(id uint64, labels labels.Set) error {
	// Validate input
	if err := r.validateID(id); err != nil {
		return err
	}
	newid := calculateIndex(id, r.start)
	treeId := id64.NewID(uint64(newid), id64.IDBitSize)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)
	return r.table.Update(newid, treeEntry)
}

func (r *table64) Size() int {
	return r.table.Size()
}

func (r *table64) Has(id uint64) bool {
	// Validate IP address
	if err := r.validateID(id); err != nil {
		return false
	}
	newid := calculateIndex(id, r.start)
	return r.table.Has(newid)
}

func (r *table64) IsFree(id uint64) bool {
	// Validate IP address
	if err := r.validateID(id); err != nil {
		return false
	}
	newid := calculateIndex(id, r.start)
	return r.table.IsFree(newid)
}

func (r *table64) FindFree() (uint64, error) {
	id, err := r.table.FindFree()
	if err != nil {
		return 0, err
	}
	return calculateIDFromIndex(r.start, id), nil
}

func (r *table64) GetAll() tree.Entries {
	entries := make(tree.Entries, 0, r.table.Size())
	for _, entry := range r.table.GetAll() {
		// need to remap the id for the outside world
		entry := tree.NewEntry(id64.NewID(uint64(calculateIDFromIndex(r.start, entry.ID())), id64.IDBitSize), entry.Data().Labels())

		entries = append(entries, entry)
	}
	return entries
}

func (r *table64) GetByLabel(selector labels.Selector) tree.Entries {
	entries := make(tree.Entries, 0, r.table.Size())

	iter := r.table.Iterate()

	for iter.Next() {
		entry := iter.Value().Data()
		if selector.Matches(entry.Labels()) {
			// need to remap the id for the outside world
			entry := tree.NewEntry(id64.NewID(uint64(calculateIDFromIndex(r.start, uint64(entry.ID().ID()))), id64.IDBitSize), entry.Labels())
			entries = append(entries, entry)
		}
	}
	return entries
}

func (r *table64) validateID(id uint64) error {
	if id < r.start {
		return fmt.Errorf("id %d, does not fit in the range from %d to %d", id, r.start, r.end)
	}
	if id > r.end {
		return fmt.Errorf("id %d, does not fit in the range from %d to %d", id, r.start, r.end)
	}
	return nil
}

func calculateIndex(id, start uint64) uint64 {
	// Calculate the index in the bitmap
	return uint64(id - start)
}

func calculateIDFromIndex(start uint64, id uint64) uint64 {
	return start + id
}
