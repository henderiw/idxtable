package table

import (
	"github.com/henderiw/idxtable/pkg/tree"
	"k8s.io/apimachinery/pkg/labels"
)

type Table interface {
	Get(id uint64) (tree.Entry, error)
	Claim(id uint64, labels labels.Set) error
	ClaimFree(labels labels.Set) (tree.Entry, error)
	Release(id uint64) error
	Update(id uint64, labels labels.Set) error
	Size() int
	Has(id uint64) bool
	IsFree(id uint64) bool
	FindFree() (uint64, error)
	GetAll() tree.Entries
	GetByLabel(selector labels.Selector) tree.Entries
}
