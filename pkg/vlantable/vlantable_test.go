package vlantable

import (
	"testing"

	"github.com/tj/assert"
	"k8s.io/apimachinery/pkg/labels"
)

func TestClaim(t *testing.T) {
	cases := map[string]struct {
		initEntries       map[int64]labels.Set
		newSuccessEntries map[int64]labels.Set
		newFailedEntries  map[int64]labels.Set
		expectedEntries   int
	}{

		"Normal": {
			initEntries: initEntries,
			newSuccessEntries: map[int64]labels.Set{
				10: map[string]string{},
				11: map[string]string{},
			},
			newFailedEntries: map[int64]labels.Set{
				5000: map[string]string{},
			},
			expectedEntries: 5,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r, err := New()
			assert.NoError(t, err)

			for id, d := range tc.newSuccessEntries {
				err := r.Claim(id, d)
				assert.NoError(t, err)

			}
			for id, d := range tc.newFailedEntries {
				err := r.Claim(id, d)
				assert.Error(t, err)
			}
			// check table
			for id := range tc.initEntries {
				if !r.Has(id) {
					t.Errorf("%s expecting initEntry: %d\n", name, id)
				}
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
			if r.Count() != tc.expectedEntries {
				t.Errorf("%s: -want %d, +got: %d\n", name, tc.expectedEntries, len(r.GetAll()))
			}
		})
	}
}
