package table64

import (
	"fmt"
	"testing"

	"github.com/henderiw/idxtable/pkg/tree/id64"
	"github.com/tj/assert"
	"k8s.io/apimachinery/pkg/labels"
)

func TestClaim(t *testing.T) {
	cases := map[string]struct {
		range64         string
		newSuccessEntries map[uint64]labels.Set
		newFailedEntries  map[uint64]labels.Set
		expectedEntries   int
	}{

		"Low": {
			range64: "100-199",
			newSuccessEntries: map[uint64]labels.Set{
				100: nil,
				199: nil,
			},
			newFailedEntries: map[uint64]labels.Set{
				500: nil,
			},
			expectedEntries: 2,
		},
		"High": {
			range64: "18446744073709551000-18446744073709551614",
			newSuccessEntries: map[uint64]labels.Set{
				18446744073709551001: nil,
				18446744073709551010: nil,
			},
			newFailedEntries: map[uint64]labels.Set{
				18446744073709551615: nil,
			},
			expectedEntries: 2,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			trange, err := id64.ParseRange(tc.range64)
			assert.NoError(t, err)

			r := New(uint64(trange.From().ID()), uint64(trange.To().ID()))

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
