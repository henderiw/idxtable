package vlantable

import (
	"fmt"

	"github.com/henderiw/idxtable/pkg/idxtable"
	"k8s.io/apimachinery/pkg/labels"
)

type VLANTable interface {
	Get(id int64) (labels.Set, error)
	Claim(id int64, d labels.Set) error
	ClaimDynamic(d labels.Set) (int64, error)
	Release(id int64) error
	Update(id int64, d labels.Set) error

	Count() int
	Has(id int64) bool

	IsFree(id int64) bool
	FindFree() (int64, error)

	GetAll() map[int64]labels.Set
	GetByLabel(selector labels.Selector) map[int64]labels.Set
}

var initEntries = map[int64]labels.Set{
	0:    map[string]string{"type": "untagged", "status": "reserved"},
	1:    map[string]string{"type": "untagged", "status": "reserved"},
	4095: map[string]string{"type": "untagged", "status": "reserved"},
}

func New() (VLANTable, error) {

	t, err := idxtable.NewTable[labels.Set](
		4096,
		initEntries,
		func(id int64) error {
			switch id {
			case 0:
				return fmt.Errorf("VLAN %d is the untagged VLAN, cannot be added to the database", id)
			case 1:
				return fmt.Errorf("VLAN %d is the default VLAN, cannot be added to the database", id)
			case 4095:
				return fmt.Errorf("VLAN %d is reserved, cannot be added to the database", id)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return &vlanTable{
		table:  t,
		offset: 0,
		max:    4095,
	}, nil
}

type vlanTable struct {
	table  idxtable.Table[labels.Set]
	offset int64
	max    int64
}

func (r *vlanTable) Get(id int64) (labels.Set, error) {
	return r.table.Get(r.calculateIndex(id))
}

func (r *vlanTable) Claim(id int64, d labels.Set) error {
	id = r.calculateIndex(id)
	if !r.table.IsFree(id) {
		return fmt.Errorf("id %d is already claimed", id)
	}
	return r.table.Claim(id, d)
}

func (r *vlanTable) ClaimDynamic(d labels.Set) (int64, error) {
	return r.table.ClaimDynamic(d)
}

func (r *vlanTable) ClaimRange(start, size int64, d labels.Set) error {
	return r.table.ClaimRange(start, size, d)
}

func (r *vlanTable) ClaimSize(size int64, d labels.Set) error {
	return r.table.ClaimSize(size, d)
}

func (r *vlanTable) Release(id int64) error {
	return r.table.Release(r.calculateIndex(id))
}

func (r *vlanTable) Update(id int64, d labels.Set) error {
	id = r.calculateIndex(id)
	if !r.table.IsFree(id) {
		return fmt.Errorf("id %d is already claimed", id)
	}
	return r.table.Update(id, d)
}

func (r *vlanTable) Count() int {
	return r.table.Count()
}

func (r *vlanTable) Has(id int64) bool {
	return r.table.Has(r.calculateIndex(id))
}

func (r *vlanTable) IsFree(id int64) bool {
	return r.table.IsFree(r.calculateIndex(id))
}

func (r *vlanTable) FindFree() (int64, error) {
	id, err := r.table.FindFree()
	if err != nil {
		return -1, err
	}
	return id + r.offset, nil
}

func (r *vlanTable) GetAll() map[int64]labels.Set {
	return r.table.GetAll()
}

func (r *vlanTable) GetByLabel(selector labels.Selector) map[int64]labels.Set {
	entries := map[int64]labels.Set{}

	iter := r.table.Iterate()

	for iter.Next() {
		if selector.Matches(iter.Value()) {
			entries[iter.ID()] = iter.Value()
		}
	}
	return entries
}

func (r *vlanTable) calculateIndex(id int64) int64 {
	// Calculate the index in the bitmap
	return id - r.offset
}
