package tree

import "fmt"

var _leftMasks16 []uint16
var _leftMasks32 []uint32
var _leftMasks64 []uint64

func initBuildLeftMasks() {
	_leftMasks16 = make([]uint16, 17)
	for i := uint8(1); i < 13; i++ {
		_leftMasks16[i] = uint16(_leftMasks16[i-1] | 1<<(17-i))
	}

	_leftMasks32 = make([]uint32, 33)
	for i := uint8(1); i < 33; i++ {
		_leftMasks32[i] = uint32(_leftMasks32[i-1] | 1<<(32-i))
	}

	_leftMasks64 = make([]uint64, 65)
	for i := uint8(1); i < 65; i++ {
		_leftMasks64[i] = uint64(_leftMasks64[i-1] | 1<<(64-i))
	}
}

// create a new node in the tree, return its index
func (t *Tree[T]) newNode(id ID, length uint8) uint {
	availCount := len(t.availableIndexes)
	if availCount > 0 {
		index := t.availableIndexes[availCount-1]
		t.availableIndexes = t.availableIndexes[:availCount-1]
		t.nodes[index] = treeNode[T]{Id: id.ID(), Length: length}
		return index
	}

	t.nodes = append(t.nodes, treeNode[T]{Id: id.ID(), Length: length})
	return uint(len(t.nodes) - 1)
}

func (iter *TreeIterator[T]) ID() {
	//fmt.Println("leftmask32", _leftMasks32)
	var id uint32
	var length uint8
	for _, i := range iter.nodeHistory {
		//fmt.Println("nodehistory id", iter.t.nodes[i].Id)
		//fmt.Println("nodehistory len", iter.t.nodes[i].Length)
		id, length = MergeID32(id, length, uint32(iter.t.nodes[i].Id), iter.t.nodes[i].Length)
	}
	id, length = MergeID32(id, length, uint32(iter.t.nodes[iter.nodeIndex].Id), iter.t.nodes[iter.nodeIndex].Length)

	fmt.Println(id, length)
}

/*
func (iter *TreeIterator[T]) ID() {
	//fmt.Println("leftmask32", _leftMasks32)
	var id uint16
	var length uint8
	for _, i := range iter.nodeHistory {
		//fmt.Println("nodehistory id", iter.t.nodes[i].Id)
		//fmt.Println("nodehistory len", iter.t.nodes[i].Length)
		id, length = MergeID16(id, length, uint16(iter.t.nodes[i].Id), iter.t.nodes[i].Length)
	}
	id, length = MergeID16(id, length, uint16(iter.t.nodes[iter.nodeIndex].Id), iter.t.nodes[iter.nodeIndex].Length)

	fmt.Println(id, length)
}
*/

func MergeID32(left uint32, leftLength uint8, right uint32, rightLength uint8) (uint32, uint8) {
	return (left & _leftMasks32[leftLength]) | ((right & _leftMasks32[rightLength]) >> leftLength), (leftLength + rightLength)
}

func MergeID16(left uint16, leftLength uint8, right uint16, rightLength uint8) (uint16, uint8) {
	return (left & _leftMasks16[leftLength]) | ((right & _leftMasks16[rightLength]) >> leftLength), (leftLength + rightLength)
}
