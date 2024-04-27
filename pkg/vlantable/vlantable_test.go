package vlantable

import (
	"fmt"
	"testing"

	"github.com/henderiw/idxtable/pkg/tree/id32"
	"github.com/henderiw/idxtable/pkg/vlantree"
	"github.com/tj/assert"
)

func TestClaim(t *testing.T) {
	cases := map[string]struct {
		vlanRange         string
		newSuccessEntries map[uint16]vlantree.Entry
		newFailedEntries  map[uint16]vlantree.Entry
		expectedEntries   int
	}{

		"Normal": {
			vlanRange: "100-199",
			newSuccessEntries: map[uint16]vlantree.Entry{
				100: vlantree.NewEntry(100, nil),
				199: vlantree.NewEntry(199, nil),
			},
			newFailedEntries: map[uint16]vlantree.Entry{
				500: vlantree.NewEntry(500, nil),
			},
			expectedEntries: 2,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			vlanRange, err := id32.ParseRange(tc.vlanRange)
			assert.NoError(t, err)

			r := New(uint16(vlanRange.From().ID()), uint16(vlanRange.To().ID()))

			for id, e := range tc.newSuccessEntries {
				err := r.Claim(id, e)
				assert.NoError(t, err)

			}
			for id, e := range tc.newFailedEntries {
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
