package idxtable

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

var initEntries = map[int64]string{
	0:   "a",
	1:   "b",
	999: "c",
}

func TestNewTable(t *testing.T) {
	cases := map[string]struct {
		size            int64
		initEntries     map[int64]string
		validation      ValidationFn
		expectedEntries int
		expectedErr     bool
	}{

		"NewWithoutInitEntries": {
			size:            1000,
			initEntries:     nil,
			expectedEntries: 0,
		},
		"NewWithInitEntries": {
			size:            1000,
			initEntries:     initEntries,
			validation:      func(id int64) error { return nil },
			expectedEntries: 3,
		},
		"NewErrorMaxEntries": {
			size:        100,
			initEntries: initEntries,
			expectedErr: true,
		},
		"NewErrorValidation": {
			size:        999,
			initEntries: initEntries,
			validation: func(id int64) error {
				if id == 5000 {
					return errors.New("vaidation")
				}
				return nil
			},
			expectedErr: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r, err := NewTable[string](tc.size, tc.initEntries, tc.validation)
			if tc.expectedErr {
				assert.Error(t, err)
				return
			} else {
				assert.NoError(t, err)
			}
			if r.Count() != tc.expectedEntries {
				t.Errorf("%s: -want %d, +got: %d\n", name, tc.expectedEntries, len(r.GetAll()))
			}
		})
	}
}

func TestClaim(t *testing.T) {
	cases := map[string]struct {
		size              int64
		initEntries       map[int64]string
		newSuccessEntries map[int64]string
		newFailedEntries  map[int64]string
		expectedEntries   int
	}{

		"Normal": {
			size:        1000,
			initEntries: initEntries,
			newSuccessEntries: map[int64]string{
				10: "a",
				11: "b",
			},
			newFailedEntries: map[int64]string{
				1000: "x",
			},
			expectedEntries: 5,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r, err := NewTable[string](tc.size, tc.initEntries, nil)
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

func TestRelease(t *testing.T) {
	cases := map[string]struct {
		size                 int64
		initEntries          map[int64]string
		newSuccessEntries    map[int64]string
		expectedEntries      int
		deleteSuccessEntries []int64
		deleteFailedEntries  []int64
	}{

		"Normal": {
			size:        1000,
			initEntries: initEntries,
			newSuccessEntries: map[int64]string{
				10: "a",
				11: "b",
			},
			deleteSuccessEntries: []int64{0, 10, 11},
			deleteFailedEntries:  []int64{20, 21},

			expectedEntries: 2,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r, err := NewTable[string](tc.size, tc.initEntries, nil)
			assert.NoError(t, err)

			for id, d := range tc.newSuccessEntries {
				err := r.Claim(id, d)
				assert.NoError(t, err)

			}
			// delete entries
			for _, id := range tc.deleteSuccessEntries {
				err := r.Release(id)
				assert.NoError(t, err)
			}
			for _, id := range tc.deleteFailedEntries {
				err := r.Release(id)
				assert.NoError(t, err)
			}
			for id := range tc.initEntries {
				found := false
				for _, did := range tc.deleteSuccessEntries {
					if did == id {
						found = true
						break
					}
				}
				if found {
					_, err := r.Get(id)
					assert.Error(t, err)
					if r.Has(id) {
						t.Errorf("%s not expecting deleted claim entry: %d\n", name, id)
					}
				} else {
					_, err := r.Get(id)
					assert.NoError(t, err)
					if !r.Has(id) {
						t.Errorf("%s expecting non deleted claim entry: %d\n", name, id)
					}
				}
			}
			for id := range tc.newSuccessEntries {
				found := false
				for _, did := range tc.deleteSuccessEntries {
					if did == id {
						found = true
						break
					}
				}
				if found {
					_, err := r.Get(id)
					assert.Error(t, err)
					if r.Has(id) {
						t.Errorf("%s not expecting deleted claim entry: %d\n", name, id)
					}
				} else {
					_, err := r.Get(id)
					assert.NoError(t, err)
					if !r.Has(id) {
						t.Errorf("%s expecting non deleted claim entry: %d\n", name, id)
					}
				}
			}

			if r.Count() != tc.expectedEntries {
				t.Errorf("%s: -want %d, +got: %d\n", name, tc.expectedEntries, len(r.GetAll()))
			}
		})
	}
}

func TestIterate(t *testing.T) {
	cases := map[string]struct {
		size        int64
		initEntries map[int64]string
		keys        []int64
	}{

		"Normal": {
			size:        1000,
			initEntries: initEntries,
			keys:        []int64{0, 1, 999},
		},
		"None": {
			size:        1000,
			initEntries: nil,
			keys:        []int64{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r, err := NewTable[string](tc.size, tc.initEntries, nil)
			assert.NoError(t, err)

			i := r.Iterate()
			if diff := cmp.Diff(tc.keys, i.keys); diff != "" {
				t.Errorf("%s: -want, +got:\n%s", name, diff)
			}
		})
	}
}

func TestClaimRange(t *testing.T) {
	cases := map[string]struct {
		size            int64
		initEntries     map[int64]string
		start           int64
		total           int64
		expectedEntries int
		expectedErr     bool
	}{

		"Normal": {
			size:            10,
			initEntries:     nil,
			start:           5,
			total:           5,
			expectedEntries: 5,
		},
		"ErrorMax": {
			size:            10,
			initEntries:     nil,
			start:           5,
			total:           6,
			expectedEntries: 0,
			expectedErr:     true,
		},
		"ErrorOverlap": {
			size:            1000,
			initEntries:     initEntries,
			start:           0,
			total:           5,
			expectedEntries: 3,
			expectedErr:     true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r, err := NewTable[string](tc.size, tc.initEntries, nil)
			assert.NoError(t, err)

			err = r.ClaimRange(tc.start, tc.total, "a")
			if tc.expectedErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			for id := tc.start; id < tc.total; id++ {
				if !r.Has(id) {
					t.Errorf("%s expecting entry: %d\n", name, id)
				}
			}

			if r.Count() != tc.expectedEntries {
				t.Errorf("%s: -want %d, +got: %d\n", name, tc.expectedEntries, len(r.GetAll()))
			}
		})
	}
}

func TestClaimSize(t *testing.T) {
	cases := map[string]struct {
		size            int64
		initEntries     map[int64]string
		total           int64
		expectedEntries int
		expectedErr     bool
	}{

		"Normal": {
			size: 1000,
			//initEntries:     initEntries,
			total:           1000,
			expectedEntries: 1000,
		},
		"ErrorMax": {
			size:            10,
			total:           11,
			expectedEntries: 0,
			expectedErr:     true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r, err := NewTable[string](tc.size, tc.initEntries, nil)
			assert.NoError(t, err)

			err = r.ClaimSize(tc.total, "a")
			if tc.expectedErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			if r.Count() != tc.expectedEntries {
				t.Errorf("%s: -want %d, +got: %d\n", name, tc.expectedEntries, len(r.GetAll()))
			}
		})
	}
}
