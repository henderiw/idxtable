package tree32

import (
	"testing"

	"github.com/henderiw/idxtable/pkg/tree/id32"
	"github.com/tj/assert"
	"k8s.io/apimachinery/pkg/labels"
)

func TestClaim(t *testing.T) {
	cases := map[string]struct {
		newSuccessEntries map[uint16]labels.Set
		newFailedEntries  map[uint16]labels.Set
		expectedEntries   int
	}{

		"Normal": {
			newSuccessEntries: map[uint16]labels.Set{
				10: map[string]string{},
				11: map[string]string{},
			},
			newFailedEntries: map[uint16]labels.Set{
				5000: map[string]string{},
			},
			expectedEntries: 2,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			vt := New()

			for id, d := range tc.newSuccessEntries {
				treeid := id32.NewID(uint32(id), 32)
				err := vt.Claim(treeid, d)
				assert.NoError(t, err)

			}
			for id, d := range tc.newFailedEntries {
				treeid := id32.NewID(uint32(id), 32)
				err := vt.Claim(treeid, d)
				assert.Error(t, err)
			}
			// check table
			for id := range tc.newSuccessEntries {
				treeid := id32.NewID(uint32(id), 32)
				if _, err := vt.Get(treeid); err != nil {
					t.Errorf("%s expecting success claim entry: %d\n", name, id)
				}
			}
			for id := range tc.newFailedEntries {
				treeid := id32.NewID(uint32(id), 32)
				if _, err := vt.Get(treeid); err == nil {
					t.Errorf("%s no expecting failed claim entry: %d\n", name, id)
				}
			}
			if len(vt.GetAll()) != tc.expectedEntries {
				t.Errorf("%s: -want %d, +got: %d\n", name, tc.expectedEntries, len(vt.GetAll()))
			}
		})
	}
}
