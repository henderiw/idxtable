package vlantable

import (
	"fmt"

	"github.com/henderiw/idxtable/pkg/idxtable"
	"github.com/henderiw/idxtable/pkg/tree"
	"github.com/henderiw/idxtable/pkg/tree/id32"
	"k8s.io/apimachinery/pkg/labels"
)

type VLANTable interface {
	Get(id uint16) (tree.Entry, error)
	Claim(u uint16, labels labels.Set) error
	ClaimFree(labels labels.Set) (tree.Entry, error)
	Release(id uint16) error
	Update(id uint16, labels labels.Set) error

	Size() int
	Has(id uint16) bool

	IsFree(id uint16) bool
	FindFree() (uint16, error)

	GetAll() tree.Entries
	GetByLabel(selector labels.Selector) tree.Entries
}

func New(start, end uint16) VLANTable {
	return &vlanTable{
		table: idxtable.NewTable[tree.Entry](
			int64(end - start + 1),
		),
		start: start,
		end:   end,
	}
}

type vlanTable struct {
	table idxtable.Table[tree.Entry]
	start uint16
	end   uint16
}

func (r *vlanTable) Get(id uint16) (tree.Entry, error) {
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

func (r *vlanTable) Claim(id uint16, labels labels.Set) error {
	// Validate input
	if err := r.validateID(id); err != nil {
		return err
	}
	newid := calculateIndex(id, r.start)
	if !r.table.IsFree(newid) {
		return fmt.Errorf("claim failed id %d already claimed", newid)
	}

	treeId := id32.NewID(uint32(newid), 32)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)
	return r.table.Claim(newid, treeEntry)
}

func (r *vlanTable) ClaimFree(labels labels.Set) (tree.Entry, error) {
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

func (r *vlanTable) Release(id uint16) error {
	// Validate input
	if err := r.validateID(id); err != nil {
		return err
	}
	newid := calculateIndex(id, r.start)
	return r.table.Release(newid)
}

func (r *vlanTable) Update(id uint16, labels labels.Set) error {
	// Validate input
	if err := r.validateID(id); err != nil {
		return err
	}
	newid := calculateIndex(id, r.start)
	treeId := id32.NewID(uint32(newid), 32)
	treeEntry := tree.NewEntry(treeId.Copy(), labels)
	return r.table.Update(newid, treeEntry)
}

func (r *vlanTable) Size() int {
	return r.table.Size()
}

func (r *vlanTable) Has(id uint16) bool {
	// Validate IP address
	if err := r.validateID(id); err != nil {
		return false
	}
	newid := calculateIndex(id, r.start)
	return r.table.Has(newid)
}

func (r *vlanTable) IsFree(id uint16) bool {
	// Validate IP address
	if err := r.validateID(id); err != nil {
		return false
	}
	newid := calculateIndex(id, r.start)
	return r.table.IsFree(newid)
}

func (r *vlanTable) FindFree() (uint16, error) {
	id, err := r.table.FindFree()
	if err != nil {
		return 0, err
	}
	return calculateIDFromIndex(r.start, id), nil
}

func (r *vlanTable) GetAll() tree.Entries {
	var entries tree.Entries
	for _, entry := range r.table.GetAll() {
		entries = append(entries, entry.Data())
	}
	return entries
}

func (r *vlanTable) GetByLabel(selector labels.Selector) tree.Entries {
	var entries tree.Entries

	iter := r.table.Iterate()

	for iter.Next() {
		entry := iter.Value().Data()
		if selector.Matches(entry.Labels()) {
			entries = append(entries, entry)
		}
	}

	return entries
}

func (r *vlanTable) validateID(id uint16) error {

	if id < r.start {
		return fmt.Errorf("id %d, does not fit in the range from %d to %d", id, r.start, r.end)
	}
	if id > r.end {
		return fmt.Errorf("id %d, does not fit in the range from %d to %d", id, r.start, r.end)
	}
	return nil
}

func calculateIndex(id, start uint16) int64 {
	// Calculate the index in the bitmap
	return int64(id - start)
}

func calculateIDFromIndex(start uint16, id int64) uint16 {
	return start + uint16(id)
}
