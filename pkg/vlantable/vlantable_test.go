package vlantable

import (
	"fmt"
	"testing"

	"github.com/henderiw/idxtable/pkg/tree"
	"github.com/henderiw/idxtable/pkg/tree/id32"
	"github.com/tj/assert"
	"k8s.io/apimachinery/pkg/labels"
)

func TestClaim(t *testing.T) {
	cases := map[string]struct {
		vlanRange         string
		newSuccessEntries map[uint16]labels.Set
		newFailedEntries  map[uint16]labels.Set
		expectedEntries   int
	}{

		"Normal": {
			vlanRange: "100-199",
			newSuccessEntries: map[uint16]labels.Set{
				100: nil,
				199: nil,
			},
			newFailedEntries: map[uint16]labels.Set{
				500: nil,
			},
			expectedEntries: 2,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			vlanRange, err := id32.ParseRange(tc.vlanRange)
			assert.NoError(t, err)

			r := New(uint16(vlanRange.From().ID()), uint16(vlanRange.To().ID()))

			for id, labels := range tc.newSuccessEntries {
				e := tree.NewEntry(id32.NewID(uint32(id), 32), labels)
				err := r.Claim(id, e)
				assert.NoError(t, err)

			}
			for id, labels := range tc.newFailedEntries {
				e := tree.NewEntry(id32.NewID(uint32(id), 32), labels)
				err := r.Claim(id, e)
				assert.Error(t, err)
			}
			for id := range tc.newSuccessEntries {
				if !r.Has(id) {
					t.Errorf("%s expecting success claim entry: %d\n", name, id)
				}
			}
			for id := range tc.newFailedEntries {
				if r.Has(id) {
					t.Errorf("%s no expecting failed claim entry: %d\n", name, id)
				}
			}
			if r.Size() != tc.expectedEntries {
				t.Errorf("%s: -want %d, +got: %d\n", name, tc.expectedEntries, len(r.GetAll()))
			}

			id, err := r.FindFree()
			assert.NoError(t, err)
			fmt.Println(id)
		})
	}
}
