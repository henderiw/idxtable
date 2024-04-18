package iptable

import (
	"fmt"
	"testing"

	"github.com/hansthienpondt/nipam/pkg/table"
	"github.com/tj/assert"
	"go4.org/netipx"
)

func TestClaim(t *testing.T) {
	cases := map[string]struct {
		ipRange           string
		newSuccessEntries map[string]table.Route
		newFailedEntries  map[string]table.Route
		expectedEntries   int
	}{

		"Normal": {
			ipRange: "10.0.0.10-10.0.0.20",
			newSuccessEntries: map[string]table.Route{
				"10.0.0.10": {},
				"10.0.0.11": {},
			},
			newFailedEntries: map[string]table.Route{
				"10.0.0.21": {},
			},
			expectedEntries: 2,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			ipRange, err := netipx.ParseIPRange(tc.ipRange)
			assert.NoError(t, err)

			r, err := New(ipRange.From(), ipRange.To())
			assert.NoError(t, err)

			for addr, d := range tc.newSuccessEntries {
				err := r.Claim(addr, d)
				assert.NoError(t, err)

			}
			for addr, d := range tc.newFailedEntries {
				err := r.Claim(addr, d)
				assert.Error(t, err)
			}
			for addr := range tc.newSuccessEntries {
				if !r.Has(addr) {
					t.Errorf("%s expecting success claim entry: %s\n", name, addr)
				}
			}
			for addr := range tc.newFailedEntries {
				if r.Has(addr) {
					t.Errorf("%s no expecting failed claim entry: %s\n", name, addr)
				}
			}
			if r.Count() != tc.expectedEntries {
				t.Errorf("%s: -want %d, +got: %d\n", name, tc.expectedEntries, len(r.GetAll()))
			}

			a, err := r.FindFree()
			assert.NoError(t, err)
			fmt.Println(a.String())
		})
	}
}
