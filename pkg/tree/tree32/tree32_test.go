package tree32

import (
	"fmt"
	"testing"

	"github.com/henderiw/idxtable/pkg/tree/id32"
	"github.com/tj/assert"
	"k8s.io/apimachinery/pkg/labels"
)

func TestClaim(t *testing.T) {
	cases := map[string]struct {
		newSuccessEntries map[uint32]labels.Set
		newFailedEntries  map[uint32]labels.Set
		expectedEntries   int
	}{

		"Normal": {
			newSuccessEntries: map[uint32]labels.Set{
				10: map[string]string{},
				11: map[string]string{},
			},
			newFailedEntries: map[uint32]labels.Set{
				2000000000: map[string]string{},
			},
			expectedEntries: 2,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vt, err := New("dummy", id32.IDBitSize - 2)
			assert.NoError(t, err)

			for id, d := range tc.newSuccessEntries {
				treeid := id32.NewID(id, id32.IDBitSize)
				err := vt.ClaimID(treeid, d)
				assert.NoError(t, err)

			}
			for id, d := range tc.newFailedEntries {
				treeid := id32.NewID(id, id32.IDBitSize)
				err := vt.ClaimID(treeid, d)
				assert.Error(t, err)
			}
			// check table
			for id := range tc.newSuccessEntries {
				treeid := id32.NewID(id, id32.IDBitSize)
				if _, err := vt.Get(treeid); err != nil {
					t.Errorf("%s expecting success claim entry: %d\n", name, id)
				}
			}
			for id := range tc.newFailedEntries {
				treeid := id32.NewID(id, id32.IDBitSize)
				if _, err := vt.Get(treeid); err == nil {
					t.Errorf("%s expecting failed claim entry: %d\n", name, id)
				}
			}
			if len(vt.GetAll()) != tc.expectedEntries {
				t.Errorf("%s: -want %d, +got: %d\n", name, tc.expectedEntries, len(vt.GetAll()))
			}
		})
	}
}

func TestASClaim(t *testing.T) {
	cases := map[string]struct {
		trange  string
		entries []uint32
	}{
		"test": {
			trange:  "65000-65100",
			entries: []uint32{65535},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			vt, err := New("dummy", id32.IDBitSize)
			assert.NoError(t, err)

			trange, err := id32.ParseRange(tc.trange)
			assert.NoError(t, err)
			for _, id := range trange.IDs() {
				err := vt.ClaimID(id, labels.Set{})
				assert.NoError(t, err)
			}

			for _, id := range tc.entries {

				tid := id32.NewID(id, 32)

				err := vt.ClaimID(tid, labels.Set{})
				assert.NoError(t, err)
			}

			vt.PrintNodes()
			vt.PrintValues()

			for _, e := range vt.GetAll() {
				fmt.Println("entry", e)
			}
		})
	}
}
