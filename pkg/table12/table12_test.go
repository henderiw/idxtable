package table12

import (
	"fmt"
	"testing"

	"github.com/henderiw/idxtable/pkg/tree/id32"
	"github.com/tj/assert"
	"k8s.io/apimachinery/pkg/labels"
)

func TestClaim(t *testing.T) {
	cases := map[string]struct {
		range12         string
		newSuccessEntries map[uint16]labels.Set
		newFailedEntries  map[uint16]labels.Set
		expectedEntries   int
	}{

		"Normal": {
			range12: "100-199",
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

			range12, err := id32.ParseRange(tc.range12)
			assert.NoError(t, err)

			r := New(uint16(range12.From().ID()), uint16(range12.To().ID()))

			for id, labels := range tc.newSuccessEntries {
				err := r.Claim(id, labels)
				assert.NoError(t, err)

			}
			for id, labels := range tc.newFailedEntries {
				err := r.Claim(id, labels)
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
