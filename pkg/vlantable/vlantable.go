package vlantable

import (
	"fmt"

	"github.com/henderiw/idxtable/pkg/idxtable"
	"k8s.io/apimachinery/pkg/labels"
)

var initEntries = map[int64]labels.Set{
	0:    map[string]string{"type": "untagged", "status": "reserved"},
	1:    map[string]string{"type": "untagged", "status": "reserved"},
	4095: map[string]string{"type": "untagged", "status": "reserved"},
}

func New() (idxtable.Table[labels.Set], error) {
	return idxtable.NewTable[labels.Set](
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
}
