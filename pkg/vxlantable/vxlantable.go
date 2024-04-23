package vxlantable

import (
	"fmt"

	"github.com/henderiw/idxtable/pkg/idxtable"
	"k8s.io/apimachinery/pkg/labels"
)

type IPTable interface {
	Get(id int64) (labels.Set, error)
	Claim(id int64, d labels.Set) error
	Release(id int64) error
	Update(id int64, d labels.Set) error

	Count() int
	Has(id int64) bool

	IsFree(id int64) bool
	FindFree() (int64, error)

	GetAll() map[int64]labels.Set
}

func New(offset, max int64) (IPTable, error) {

	t, err := idxtable.NewTable[labels.Set](
		max-offset,
		map[int64]labels.Set{},
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &vxlanTable{
		table:  t,
		offset: offset,
		max:    max,
	}, nil

}

type vxlanTable struct {
	table  idxtable.Table[labels.Set]
	offset int64
	max    int64
}

func (r *vxlanTable) Get(id int64) (labels.Set, error) {
	return r.table.Get(r.calculateIndex(id))
}

func (r *vxlanTable) Claim(id int64, d labels.Set) error {
	id = r.calculateIndex(id)
	if !r.table.IsFree(id) {
		return fmt.Errorf("id %d is already claimed", id)
	}
	return r.table.Claim(id, d)
}

func (r *vxlanTable) Release(id int64) error {
	return r.table.Release(r.calculateIndex(id))
}

func (r *vxlanTable) Update(id int64, d labels.Set) error {
	id = r.calculateIndex(id)
	if !r.table.IsFree(id) {
		return fmt.Errorf("id %d is already claimed", id)
	}
	return r.table.Update(id, d)
}

func (r *vxlanTable) Count() int {
	return r.table.Count()
}

func (r *vxlanTable) Has(id int64) bool {
	return r.table.Has(r.calculateIndex(id))
}

func (r *vxlanTable) IsFree(id int64) bool {
	return r.table.IsFree(r.calculateIndex(id))
}

func (r *vxlanTable) FindFree() (int64, error) {
	id, err := r.table.FindFree()
	if err != nil {
		return -1, err
	}
	return id + r.offset, nil
}

func (r *vxlanTable) GetAll() map[int64]labels.Set {
	return r.table.GetAll()
}

func (r *vxlanTable) calculateIndex(id int64) int64 {
	// Calculate the index in the bitmap
	return id - r.offset
}
