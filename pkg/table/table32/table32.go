package table32

import (
	"fmt"

	"github.com/henderiw/idxtable/pkg/idxtable"
	"github.com/henderiw/idxtable/pkg/table"
	"github.com/henderiw/idxtable/pkg/tree"
	"github.com/henderiw/idxtable/pkg/tree/id32"
	"k8s.io/apimachinery/pkg/labels"
)

func New(start, end uint32) table.Table {
	return &table32{
		table: idxtable.NewTable[tree.Entry](
			int64(end - start + 1),
		),
		start: start,
		end:   end,
	}
}

type table32 struct {
	table idxtable.Table[tree.Entry]
	start uint32
	end   uint32
}

func (r *table32) Get(id uint64) (tree.Entry, error) {
	var entry tree.Entry
	// Validate input
	if err := r.validateID(id); err != nil {
		return entry, err
	}
	newid := calculateIndex(uint32(id), r.start)
	e, err := r.table.Get(newid)
	if err != nil {
		return entry, err
	}
	return e.Data(), nil
}

func (r *table32) Claim(id uint64, labels labels.Set) error {
	// Validate input
	if err := r.validateID(id); err != nil {
		return err
	}
	newid := calculateIndex(uint32(id), r.start)
	if !r.table.IsFree(newid) {
		return fmt.Errorf("claim failed id %d already claimed", calculateIDFromIndex(r.start, newid))
	}

	treeId := id32.NewID(uint32(newid), id32.IDBitSize)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)
	return r.table.Claim(newid, treeEntry)
}

func (r *table32) ClaimFree(labels labels.Set) (tree.Entry, error) {
	// Validate input

	id, err := r.FindFree()
	if err != nil {
		return nil, err
	}
	if err := r.Claim(id, labels); err != nil {
		return nil, err
	}
	treeId := id32.NewID(uint32(id), id32.IDBitSize)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)
	return treeEntry, nil
}

func (r *table32) Release(id uint64) error {
	// Validate input
	if err := r.validateID(id); err != nil {
		return err
	}
	newid := calculateIndex(uint32(id), r.start)
	return r.table.Release(newid)
}

func (r *table32) Update(id uint64, labels labels.Set) error {
	// Validate input
	if err := r.validateID(id); err != nil {
		return err
	}
	newid := calculateIndex(uint32(id), r.start)
	treeId := id32.NewID(uint32(newid), id32.IDBitSize)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)
	return r.table.Update(newid, treeEntry)
}

func (r *table32) Size() int {
	return r.table.Size()
}

func (r *table32) Has(id uint64) bool {
	// Validate IP address
	if err := r.validateID(id); err != nil {
		return false
	}
	newid := calculateIndex(uint32(id), r.start)
	return r.table.Has(newid)
}

func (r *table32) IsFree(id uint64) bool {
	// Validate IP address
	if err := r.validateID(id); err != nil {
		return false
	}
	newid := calculateIndex(uint32(id), r.start)
	return r.table.IsFree(newid)
}

func (r *table32) FindFree() (uint64, error) {
	id, err := r.table.FindFree()
	if err != nil {
		return 0, err
	}
	return uint64(calculateIDFromIndex(r.start, id)), nil
}

func (r *table32) GetAll() tree.Entries {
	entries := make(tree.Entries, 0, r.table.Size())
	for _, entry := range r.table.GetAll() {
		fmt.Println("getall", entry)
		fmt.Println("getall", entry.ID(), entry.Data())
		fmt.Println("getall", calculateIDFromIndex(r.start, entry.ID()))
		// need to remap the id for the outside world
		entry := tree.NewEntry(id32.NewID(uint32(calculateIDFromIndex(r.start, entry.ID())), id32.IDBitSize), entry.Data().Labels())

		entries = append(entries, entry)
	}
	return entries
}

func (r *table32) GetByLabel(selector labels.Selector) tree.Entries {
	entries := make(tree.Entries, 0, r.table.Size())

	iter := r.table.Iterate()

	for iter.Next() {
		entry := iter.Value().Data()
		if selector.Matches(entry.Labels()) {
			// need to remap the id for the outside world
			entry := tree.NewEntry(id32.NewID(uint32(calculateIDFromIndex(r.start, int64(entry.ID().ID()))), id32.IDBitSize), entry.Labels())
			entries = append(entries, entry)
		}
	}
	return entries
}

func (r *table32) validateID(id uint64) error {
	if id > 4294967295 {
		return fmt.Errorf("id %d, cannot be bigger than 4294967295", id)
	}
	if uint32(id) < r.start {
		return fmt.Errorf("id %d, does not fit in the range from %d to %d", id, r.start, r.end)
	}
	if uint32(id) > r.end {
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
