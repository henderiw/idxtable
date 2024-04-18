package iptable

import (
	"fmt"
	"math/big"
	"net/netip"

	"github.com/hansthienpondt/nipam/pkg/table"
	"github.com/henderiw/idxtable/pkg/idxtable"
	"go4.org/netipx"
)

type IPTable interface {
	Get(addr string) (table.Route, error)
	Claim(addr string, d table.Route) error
	Release(addr string) error
	Update(addr string, d table.Route) error

	Count() int
	Has(addr string) bool

	IsFree(addr string) bool
	FindFree() (netip.Addr, error)

	GetAll() []table.Route
}

func New(from, to netip.Addr) (IPTable, error) {
	t, err := idxtable.NewTable[table.Route](
		int64(numIPs(from, to)),
		map[int64]table.Route{},
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &ipTable{
		table:   t,
		ipRange: netipx.IPRangeFrom(from, to),
	}, nil

}

type ipTable struct {
	table   idxtable.Table[table.Route]
	ipRange netipx.IPRange
}

func (r *ipTable) Get(addr string) (table.Route, error) {
	var route table.Route
	// Validate IP address
	claimIP, err := r.validateIP(addr)
	if err != nil {
		return route, err
	}
	id := calculateIndex(claimIP, r.ipRange.From())
	return r.table.Get(id)
}

func (r *ipTable) Claim(addr string, d table.Route) error {
	// Validate IP address
	claimIP, err := r.validateIP(addr)
	if err != nil {
		return err
	}
	id := calculateIndex(claimIP, r.ipRange.From())
	if !r.table.IsFree(id) {
		return fmt.Errorf("ip address %s is already claimed", addr)
	}
	return r.table.Claim(id, d)
}

func (r *ipTable) Release(addr string) error {
	// Validate IP address
	claimIP, err := r.validateIP(addr)
	if err != nil {
		return err
	}
	id := calculateIndex(claimIP, r.ipRange.From())
	return r.table.Release(id)
}

func (r *ipTable) Update(addr string, d table.Route) error {
	// Validate IP address
	claimIP, err := r.validateIP(addr)
	if err != nil {
		return err
	}
	id := calculateIndex(claimIP, r.ipRange.From())
	if !r.table.IsFree(id) {
		return fmt.Errorf("ip address %s is already claimed", addr)
	}
	return r.table.Update(id, d)
}

func (r *ipTable) Count() int {
	return r.table.Count()
}

func (r *ipTable) Has(addr string) bool {
	// Validate IP address
	claimIP, err := r.validateIP(addr)
	if err != nil {
		return false
	}
	id := calculateIndex(claimIP, r.ipRange.From())
	return r.table.Has(id)
}

func (r *ipTable) IsFree(addr string) bool {
	// Validate IP address
	claimIP, err := r.validateIP(addr)
	if err != nil {
		return false
	}
	id := calculateIndex(claimIP, r.ipRange.From())
	return r.table.IsFree(id)
}

func (r *ipTable) FindFree() (netip.Addr, error) {
	var addr netip.Addr
	
	id, err := r.table.FindFree()
	if err != nil {
		return addr, err
	}
	return calculateIPFromIndex(r.ipRange.From(), id), nil
}

func (r *ipTable) GetAll() []table.Route {
	return r.table.GetAll()
}

func (r *ipTable) validateIP(addr string) (netip.Addr, error) {
	// Parse IP address
	claimIP, err := netip.ParseAddr(addr)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("ip address %s is invalid", addr)
	}
	if !r.ipRange.Contains(claimIP) {
		return netip.Addr{}, fmt.Errorf("ip address %s, does not fit in the range from %s to %s", addr, r.ipRange.From().String(), r.ipRange.To().String())
	}
	return claimIP, nil
}


func calculateIndex(ip, start netip.Addr) int64 {
	// Calculate the index in the bitmap
	return new(big.Int).Sub(ipToInt(ip), ipToInt(start)).Int64()
}

func numIPs(startIP, endIP netip.Addr) int {
	// Convert IP addresses to big integers
	start := ipToInt(startIP)
	end := ipToInt(endIP)

	diff := new(big.Int).Sub(end, start)
	return int(diff.Int64()) + 1 // Add 1 to include the start IP
}

func ipToInt(ip netip.Addr) *big.Int {
	// Convert IP address to big integer
	bytes := ip.As16()
	ipInt := new(big.Int)
	ipInt.SetBytes(bytes[:])
	return ipInt
}

func calculateIPFromIndex(startIP netip.Addr, id int64) netip.Addr {
	// Calculate the IP address corresponding to the index
	ipInt := new(big.Int).Add(ipToInt(startIP), big.NewInt(id))
	// Convert the big.Int representing the IP address to a byte slice with length 16
	ipBytes := ipInt.Bytes()

	if len(ipBytes) < 16 {
		// If the byte slice is shorter than 16 bytes, pad it with leading zeros
		paddedBytes := make([]byte, 16-len(ipBytes))
		ipBytes = append(paddedBytes, ipBytes...)
	}

	// Convert the byte slice to a [16]byte
	var ip16 [16]byte
	copy(ip16[:], ipBytes)

	if startIP.Is4() {
		return netip.AddrFrom4(netip.AddrFrom16(ip16).As4())
	}
	return netip.AddrFrom16(ip16)
}