package table32

import (
	"fmt"
	"testing"

	"github.com/henderiw/idxtable/pkg/tree/id32"
	"github.com/tj/assert"
	"k8s.io/apimachinery/pkg/labels"
)

func TestClaim(t *testing.T) {
	cases := map[string]struct {
		trange         string
		newSuccessEntries map[uint64]labels.Set
		newFailedEntries  map[uint64]labels.Set
		expectedEntries   int
	}{

		"Low": {
			trange: "100-199",
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
			trange: "4294967000-4294967295",
			newSuccessEntries: map[uint64]labels.Set{
				4294967000: nil,
				4294967295: nil,
			},
			newFailedEntries: map[uint64]labels.Set{
				4294967296: nil,
			},
			expectedEntries: 2,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			trange, err := id32.ParseRange(tc.trange)
			assert.NoError(t, err)

			r := New(uint32(trange.From().ID()), uint32(trange.To().ID()))

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
