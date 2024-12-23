package main

import (
	"fmt"
	"strconv"

	"github.com/henderiw/idxtable/pkg/tree/gtree"
	"github.com/henderiw/idxtable/pkg/tree/id16"
	"github.com/henderiw/idxtable/pkg/tree/tree16"
	"github.com/henderiw/idxtable/pkg/tree/tree32"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

var values = []struct {
	id     int
	mask   int
	labels map[string]string
}{
	{id: 100, mask: 32, labels: map[string]string{"a": "b"}},
	{id: 101, mask: 32, labels: map[string]string{"a": "b"}},
	{id: 200, mask: 32},
	{id: 300, mask: 32},
	{id: 4000, mask: 32},
	{id: 3000, mask: 32},
	{id: 2000, mask: 32},
}

func main() {
	/*
		t := tree.NewTree[tree.Entry](id32.IsLeftBitSet)
		for _, v := range values {
			id := id32.NewID(uint32(v.id), uint8(v.mask))
			fmt.Println("id", id.String())
			e := tree.NewEntry(id.Copy(), map[string]string{"a": "b"})
			t.Add(id, e, nil)
		}

		r := id32.RangeFrom(1024, 2048)
		ids := r.AppendIDs([]tree.ID{})
		fmt.Println("range", ids)

		for _, id := range ids {
			fmt.Println("id", id.String())
			e := tree.NewEntry(id.Copy(), map[string]string{"a": "b"})
			t.Add(id, e, nil)
		}

		t.PrintNodes(0)
		t.PrintValues()

		iter := t.Iterate()
		for iter.Next() {
			iter.ID()
			//fmt.Println("iter", iter.Vals())
		}
	*/

	/*
		vlantbl := vlantable2.New()
		e := tree.NewEntry(
			id32.NewID(uint32(100), uint8(26)),
			nil,
		)
			if err := vlantbl.Add(e); err != nil {
				panic(err)
			}
	*/

	/*
		for _, v := range values {
			e := tree.NewEntry(
				id32.NewID(uint32(v.id), uint8(v.mask)),
				v.labels,
			)
			if err := vlantbl.Add(e); err != nil {
				panic(err)
			}
		}
	*/

	/*
		rng := id32.RangeFrom(1024, 2048)
		rids := rng.AppendIDs([]tree.ID{})
		fmt.Println("range", ids)

		for _, rid := range rids {
			fmt.Println("range", rid)
			e := tree.NewEntry(
				rid,
				map[string]string{"range": "range1"},
			)
			if err := vlantbl.Add(e); err != nil {
				panic(err)
			}
		}

		it := vlantbl.Iterate()
		for it.Next() {
			fmt.Println("iter id", it.Entry().ID(), "labels", it.Entry().Labels())
		}

		ls, err := GetLabelSelector(map[string]string{"a": "b"})
		if err != nil {
			panic(err)
		}
		entries := vlantbl.GetByLabel(ls)
		for _, e := range entries {
			fmt.Println("entries by label", e.String())
		}

		//vlantbl.PrintNodes()
		//vlantbl.PrintValues()
		id := id32.NewID(2100, 32)
		e := tree.NewEntry(
			id,
			map[string]string{"range entry": "rentry1"},
		)
		if err := vlantbl.Add(e); err != nil {
			panic(err)
		}

		e, err = vlantbl.Get(id)
		if err != nil {
			panic(err)
		}
		fmt.Println("get", e)
		fmt.Println("children")
		entries = vlantbl.Children(id)
		for _, e := range entries {
			fmt.Println("entries children", e.String())
		}

		fmt.Println("parents")
		entries = vlantbl.Parents(id)
		for _, e := range entries {
			fmt.Println("entries parents", e.String())
		}

		ida := id32.NewID(1024, 22)
		idb := id32.NewID(3000, 32)
		idc := id32.NewID(100, 32)
		fmt.Println("overlap", ida.Overlaps(idb))
		fmt.Println("overlap", ida.Overlaps(idc))

		fmt.Println("lastID", id32.LastID(id32.NewID(0, 16)))
		fmt.Println("rangeOFID", id32.RangeOfID(id32.NewID(0, 16)))

	*/

	fmt.Println("lastID", id16.LastID(id16.NewID(0, 16-12)))

	vt, err := tree16.New("dummy", 12)
	if err != nil {
		panic(err)
	}

	for id := 0; id <= 4095; id++ {
		if err := vt.ClaimID(id16.NewID(uint16(id), id16.IDBitSize), map[string]string{"id": strconv.Itoa(id)}); err != nil {
			panic(err)
		}
		fmt.Println("claimed entry", id)
	}

	vt, err = tree16.New("dummy", 12)
	if err != nil {
		panic(err)
	}
	for id := 0; id <= 4095; id++ {
		e, err := vt.ClaimFree(map[string]string{"id": strconv.Itoa(id)})
		if err != nil {
			panic(err)
		}
		fmt.Println("claimed entry", e.ID(), e.Labels())
	}

	vt, err = tree32.New("dummy", 12)
	if err != nil {
		panic(err)
	}
	//vt.ClaimRange("1000-2000", map[string]string{"range": "test"})

	for id := 0; id <= 4095; id++ {
		e, err := vt.ClaimFree(map[string]string{"id": strconv.Itoa(id)})
		if err != nil {
			fmt.Println("claimed entry error", id, err.Error())
			continue
		}
		fmt.Println("claimed entry", e.ID(), e.Labels())
	}

	fullselector := labels.NewSelector()
	l := map[string]string{
		"range": "test",
	}
	for k, v := range l {
		req, err := labels.NewRequirement(k, selection.Equals, []string{v})
		if err != nil {
			panic(err)
		}
		fullselector = fullselector.Add(*req)
	}

	if err := vt.ReleaseByLabel(fullselector); err != nil {
		panic(err)
	}

	vt, err = tree16.New("dummy", 12)
	if err != nil {
		panic(err)
	}
	vt.ClaimRange("1000-2000", map[string]string{"range": "test"})

	handleId(vt, 1000)
	handleId(vt, 100)

}

func handleId(vt gtree.GTree, id uint16) {
	treeid := id16.NewID(id, id16.IDBitSize)
	e, err := vt.Get(treeid)
	if err != nil {
		fmt.Println(err)
		if err := vt.ClaimID(treeid, nil); err != nil {
			fmt.Println(err)
		}
		_, err := vt.Get(treeid)
		if err != nil {
			panic(err)
		}
		entries := vt.Parents(treeid)
		fmt.Println("parents", entries)
		return
	}
	panic(fmt.Errorf("entry should not exist: entry: %v", e))
}

func GetLabelSelector(l map[string]string) (labels.Selector, error) {
	fullselector := labels.NewSelector()
	for k, v := range l {
		req, err := labels.NewRequirement(k, selection.Equals, []string{v})
		if err != nil {
			return nil, err
		}
		fullselector = fullselector.Add(*req)
	}
	return fullselector, nil
}
