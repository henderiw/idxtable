package tree

type treeNode[T any] struct {
	Left     uint // left node index: 0 for not set
	Right    uint // right node index: 0 for not set
	Id       uint64
	Length   uint8
	ValCount int
}

// See how many bits match the input address
func (n *treeNode[T]) MatchCount(id ID) uint8 {
	var length uint8
	if id.Length() > n.Length {
		length = n.Length
	} else {
		length = id.Length()
	}

	matches := id.Matches(n.Id)
	if matches > length {
		return length
	}
	return matches
}

// ShiftLength shifts the id by the input shiftCount
func (n *treeNode[T]) ShiftLength(shiftCount uint8) {
	n.Id <<= shiftCount
	n.Length -= shiftCount
}

// MergeFromNodes updates the prefix and prefix length from the two input nodes
func (n *treeNode[T]) MergeFromNodes(left *treeNode[T], right *treeNode[T]) {
	id, l := MergeID32(uint32(left.Id), left.Length, uint32(right.Id), right.Length)
	n.Id, n.Length = uint64(id), l
}
