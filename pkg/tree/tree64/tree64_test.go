package tree64

import (
	"testing"

	"github.com/henderiw/idxtable/pkg/tree/id64"
	"github.com/tj/assert"
	"k8s.io/apimachinery/pkg/labels"
)

func TestClaim(t *testing.T) {
	cases := map[string]struct {
		newSuccessEntries map[uint64]labels.Set
		newFailedEntries  map[uint64]labels.Set
		expectedEntries   int
	}{

		"Normal": {
			newSuccessEntries: map[uint64]labels.Set{
				10: map[string]string{"a": "b"},
				11: map[string]string{"c": "d"},
			},
			newFailedEntries: map[uint64]labels.Set{
				20000000000: map[string]string{},
			},
			expectedEntries: 2,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vt, err := New("dummy", id64.IDBitSize - 32)
			assert.NoError(t, err)

			for id, d := range tc.newSuccessEntries {
				id := id
				d := d
				treeid := id64.NewID(id, id64.IDBitSize)
				err := vt.ClaimID(treeid.Copy(), d)
				assert.NoError(t, err)

			}
			for id, d := range tc.newFailedEntries {
				id := id
				d := d
				treeid := id64.NewID(id, id64.IDBitSize)
				err := vt.ClaimID(treeid, d)
				assert.Error(t, err)
			}
			// check table
			for id := range tc.newSuccessEntries {
				treeid := id64.NewID(id, id64.IDBitSize)
				if _, err := vt.Get(treeid); err != nil {
					t.Errorf("%s expecting success claim entry: %d\n", name, id)
				}
			}
			for id := range tc.newFailedEntries {
				treeid := id64.NewID(id, id64.IDBitSize)
				if _, err := vt.Get(treeid); err == nil {
					t.Errorf("%s expecting failed claim entry: %d\n", name, id)
				}
			}

			//vt.PrintNodes()
			//vt.PrintValues()

			if len(vt.GetAll()) != tc.expectedEntries {
				t.Errorf("%s: -want %d, +got: %d\n", name, tc.expectedEntries, len(vt.GetAll()))
			}
		})
	}
}
