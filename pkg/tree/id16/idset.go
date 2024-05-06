package id16

import (
	"errors"
	"fmt"

	"github.com/henderiw/idxtable/pkg/tree"
)

type IDSetBuilder struct {
	in   []tree.Range
	out  []tree.Range
	errs error
}

func (s *IDSetBuilder) AddId(id tree.ID) {
	if r := RangeOfID(id); r.IsValid() {
		s.AddRange(r)
	} else {
		s.errs = errors.Join(s.errs, fmt.Errorf("addId(%v-%v)", id.ID(), id.Length()))
	}
	//fmt.Println("bldr AddId in", s.in, id.ID(), id.Length())
	//fmt.Println("bldr AddId out", s.out, id.ID(), id.Length())
}

// RemoveId removes all Ids in p from s.
func (s *IDSetBuilder) RemoveId(id tree.ID) {
	if r := RangeOfID(id); r.IsValid() {
		s.RemoveRange(r)
	} else {
		s.errs = errors.Join(s.errs, fmt.Errorf("removeId(%v-%v)", id.ID(), id.Length()))
	}
	//fmt.Println("bldr RemoveId in", s.in, id.ID(), id.Length())
	//fmt.Println("bldr RemoveId out", s.out, id.ID(), id.Length())
}

func (s *IDSetBuilder) AddRange(r tree.Range) {
	if !r.IsValid() {
		s.errs = errors.Join(s.errs, fmt.Errorf("addRange(%v-%v)", r.From(), r.To()))
		return
	}
	if len(s.out) > 0 {
		s.normalize()
	}
	s.in = append(s.in, r)
}

// RemoveRange removes all IPs in r from s.
func (s *IDSetBuilder) RemoveRange(r tree.Range) {
	if r.IsValid() {
		s.out = append(s.out, r)
	} else {
		s.errs = errors.Join(s.errs, fmt.Errorf("removeRange(%v-%v)", r.From(), r.To()))
		return
	}
}

// AddSet adds all IPs in b to s.
func (s *IDSetBuilder) AddSet(b *IDSet) {
	if b == nil {
		return
	}
	for _, r := range b.rr {
		s.AddRange(r)
	}
}

// normalize normalizes s: s.in becomes the minimal sorted list of
// ranges required to describe s, and s.out becomes empty.

func (s *IDSetBuilder) normalize() {
	in, ok := mergeRanges(s.in)
	if !ok {
		return
	}
	out, ok := mergeRanges(s.out)
	if !ok {
		return
	}
	// in and out are sorted in ascending range order, and have no
	// overlaps within each other. We can run a merge of the two lists
	// in one pass.

	//fmt.Println("normalize in", in)
	//fmt.Println("normalize out", out)

	min := make([]tree.Range, 0, len(in))
	for len(in) > 0 && len(out) > 0 {
		rin, rout := in[0], out[0]

		switch {
		case !rout.IsValid() || !rin.IsValid():
			//fmt.Println("normalize not valid")
			// mergeIPRanges should have prevented invalid ranges from
			// sneaking in.
			panic("invalid IPRanges during Ranges merge")
		case rout.EntirelyBefore(rin):
			//fmt.Println("out EntirelyBefore in")
			// "out" is entirely before "in".
			//
			//    out         in
			// f-------t   f-------t
			out = out[1:]
		case rin.EntirelyBefore(rout):
			//fmt.Println("in EntirelyBefore out")
			// "in" is entirely before "out".
			//
			//    in         out
			// f------t   f-------t
			min = append(min, rin)
			in = in[1:]
		case rin.CoveredBy(rout):
			//fmt.Println("out coveredBy in")
			// "out" entirely covers "in".
			//
			//       out
			// f-------------t
			//    f------t
			//       in
			in = in[1:]
		case rout.InMiddleOf(rin):
			//fmt.Println("out in middle of in")
			// "in" entirely covers "out".
			//
			//       in
			// f-------------t
			//    f------t
			//       out
			min = append(min, r16{from: rin.From(), to: rout.From().Prev()})
			// Adjust in[0], not ir, because we want to consider the
			// mutated range on the next iteration.
			in[0] = in[0].SetFrom(rout.To().Next())
			out = out[1:]
		case rout.OverlapsStartOf(rin):
			//fmt.Println("out overlaps start of in")
			// "out" overlaps start of "in".
			//
			//   out
			// f------t
			//    f------t
			//       in
			
			//fmt.Println("out overlaps start of in, next", rout.To().Next())
			in[0] = in[0].SetFrom(rout.To().Next())
			// Can't move ir onto min yet, another later out might
			// trim it further. Just discard or and continue.
			out = out[1:]
			//fmt.Println("out overlaps start of in, in", in)
			//fmt.Println("out overlaps start of in, out", out)
		case rout.OverlapsEndOf(rin):
			//fmt.Println("out overlaps end of in")
			// "out" overlaps end of "in".
			//
			//           out
			//        f------t
			//    f------t
			//       in
			min = append(min, r16{from: rin.From(), to: rout.From().Prev()})
			in = in[1:]
		default:
			// The above should account for all combinations of in and
			// out overlapping, but insert a panic to be sure.
			panic("unexpected additional overlap scenario")
		}
	}
	if len(in) > 0 {
		// Ran out of removals before the end of in.
		min = append(min, in...)
	}

	s.in = min
	s.out = nil

}

func (s *IDSetBuilder) IPSet() (*IDSet, error) {
	//fmt.Println("ipset in", s.in)
	//fmt.Println("ipset out", s.out)
	s.normalize()
	idset := &IDSet{
		rr: append([]tree.Range{}, s.in...),
	}
	//fmt.Println("ipset", idset)
	if s.errs == nil {
		return idset, nil
	} else {
		errs := s.errs
		s.errs = nil
		return idset, errs
	}
}

type IDSet struct {
	// rr is the set of IPs that belong to this IPSet. The IPRanges
	// are normalized according to IPSetBuilder.normalize, meaning
	// they are a sorted, minimal representation (no overlapping
	// ranges, no contiguous ranges). The implementation of various
	// methods rely on this property.
	rr []tree.Range
}

// Ranges returns the minimum and sorted set of IP
// ranges that covers s.
func (s *IDSet) Ranges() []tree.Range {
	return append([]tree.Range{}, s.rr...)
}

// Prefixes returns the minimum and sorted set of IP prefixes
// that covers s.
func (s *IDSet) IDs() []tree.ID {
	out := make([]tree.ID, 0, len(s.rr))
	for _, r := range s.rr {
		out = append(out, r.IDs()...)
	}
	return out
}

// RemoveFreePrefix splits s into a Prefix of length bitLen and a new
// IPSet with that prefix removed.
//
// If no contiguous prefix of length bitLen exists in s,
// RemoveFreePrefix returns ok=false.
func (s *IDSet) RemoveFreePrefix(bitLen uint8) (tree.ID, *IDSet, bool) {
	var bestFit tree.ID
	for _, r := range s.rr {
		//fmt.Println("RemoveFreePrefix", r, bitLen)
		for _, id := range r.IDs() {
			//fmt.Println("RemoveFreePrefix", id, bitLen)
			if uint8(id.Length()) > bitLen {
				continue
			}

			if !(bestFit != nil) || id.Length() > bestFit.Length() {
				bestFit = id
				if uint8(bestFit.Length()) == bitLen {
					// exact match, done.
					break
				}
			}
		}
	}

	if bestFit == nil {
		return nil, s, false
	}

	id := NewID(uint16(bestFit.ID()), bitLen)

	//fmt.Println("RemoveFreePrefix bestFit", id)

	var b IDSetBuilder
	b.AddSet(s)
	b.RemoveId(id)
	newSet, _ := b.IPSet()
	return id, newSet, true
}
