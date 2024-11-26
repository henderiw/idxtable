package tree

import (
	"fmt"
)

func init() {
	initBuildLeftMasks()
}

// MatchesFunc[T] is called to check if tag data matches the input value
type MatchesFunc[T any] func(payload T, val T) bool

// FilterFunc[T] is called on each result to see if it belongs in the resulting set
type FilterFunc[T any] func(payload T) bool

// UpdatesFunc[T] is called to update the tag value
type UpdatesFunc[T any] func(payload T) T

// treeIteratorNext is an indicator to know what Next() should return
// for the current node.
type treeIteratorNext int

const (
	nextSelf treeIteratorNext = iota
	nextLeft
	nextRight
	nextUp
)

// deleteNodeResult is the return type for deleteNode() function
type deleteNodeResult int

const (
	notDeleted deleteNodeResult = iota
	deletedNodeReplacedByChild
	deletedNodeParentReplacedBySibling
	deletedNodeJustRemoved
)

type Tree[T any] struct {
	name             string         // name of the tree
	isLeftBitSetFn   IsLeftBitSetFn // input
	length           uint8          // input
	nodes            []treeNode[T]  // root is always at [1] - [0] is unused
	availableIndexes []uint         // a place to store node indexes that we deleted, and are available
	vals             map[uint64]T
}

type IsLeftBitSetFn func(id uint64) bool

func NewTree[T any](name string, isLeftBitSetFn IsLeftBitSetFn, length uint8) *Tree[T] {

	return &Tree[T]{
		name:             name,
		isLeftBitSetFn:   isLeftBitSetFn,
		length:           length,
		nodes:            make([]treeNode[T], 2), // index 0 is skipped, 1 is root
		availableIndexes: make([]uint, 0),
		vals:             make(map[uint64]T),
	}
}

// Clone creates an identical copy of the tree
// - Note: the items in the tree are not deep copied
func (r *Tree[T]) Clone() *Tree[T] {
	ret := &Tree[T]{
		nodes:            make([]treeNode[T], len(r.nodes), cap(r.nodes)),
		availableIndexes: make([]uint, len(r.availableIndexes), cap(r.availableIndexes)),
		vals:             make(map[uint64]T, len(r.vals)),
	}

	copy(ret.nodes, r.nodes)
	copy(ret.availableIndexes, r.availableIndexes)
	for k, v := range r.vals {
		ret.vals[k] = v
	}
	return ret
}

// add a tag to the node at the input index
// - if matchFunc is non-nil, it is used to determine equality (if nil, no existing tag match)
// - if udpateFunc is non-nil, it is used to update the tag if it already exists (if nil, the provided tag is used)
// - returns whether the tag count was increased
func (r *Tree[T]) addVal(val T, nodeIndex uint, matchFunc MatchesFunc[T], updateFunc UpdatesFunc[T]) bool {
	key := (uint64(nodeIndex) << uint64(r.length))
	valCount := r.nodes[nodeIndex].ValCount
	if matchFunc != nil {
		// need to check if this value already exists
		for i := 0; i < valCount; i++ {
			if matchFunc(r.vals[key+uint64(i)], val) {
				if updateFunc != nil {
					r.vals[key+(uint64(i))] = updateFunc(r.vals[key+(uint64(i))])
				}
				return false
			}
		}
	}
	r.vals[key+(uint64(valCount))] = val
	r.nodes[nodeIndex].ValCount++
	return true
}

func (r *Tree[T]) moveTags(fromIndex uint, toIndex uint) {
	tagCount := r.nodes[fromIndex].ValCount
	fromKey := uint64(fromIndex) << uint64(r.length)
	toKey := uint64(toIndex) << uint64(r.length)
	for i := 0; i < tagCount; i++ {
		r.vals[toKey+uint64(i)] = r.vals[fromKey+uint64(i)]
		delete(r.vals, fromKey+uint64(i))
	}
	r.nodes[toIndex].ValCount += r.nodes[fromIndex].ValCount
	r.nodes[fromIndex].ValCount = 0
}

// return the tags at the input node index - appending to the input slice if they pass the optional filter func
// - ret is only appended to
func (r *Tree[T]) valsForNode(ret []T, nodeIndex uint, filterFunc FilterFunc[T]) []T {
	if nodeIndex == 0 {
		// useful for base cases where we haven't found anything
		return ret
	}

	// TODO: clean up the typing in here, between uint, uint64
	valCount := r.nodes[nodeIndex].ValCount
	key := uint64(nodeIndex) << uint64(r.length)
	for i := 0; i < valCount; i++ {
		tag := r.vals[key+uint64(i)]
		if filterFunc == nil || filterFunc(tag) {
			ret = append(ret, tag)
		}
	}
	return ret
}

// delete tags at the input node, returning how many were deleted, and how many are left
// - uses input slice to reduce allocations
func (r *Tree[T]) deleteVal(buf []T, nodeIndex uint, matchTag T, matchFunc MatchesFunc[T]) (int, int) {
	// get tags
	buf = buf[:0]
	buf = r.valsForNode(buf, nodeIndex, nil)
	if len(buf) == 0 {
		return 0, 0
	}

	// delete tags
	// TODO: this could be done smarter - delete in place?
	for i := 0; i < r.nodes[nodeIndex].ValCount; i++ {
		delete(r.vals, (uint64(nodeIndex)<<uint64(r.length))+uint64(i))
	}
	r.nodes[nodeIndex].ValCount = 0

	// put them back
	deleteCount := 0
	keepCount := 0
	for _, tag := range buf {
		if matchFunc(tag, matchTag) {
			deleteCount++
		} else {
			// doesn't match - get to keep it
			r.addVal(tag, nodeIndex, matchFunc, nil)
			keepCount++
		}
	}
	return deleteCount, keepCount
}

// Set the single value for a node - overwrites what's there
// Returns whether the val count at this id was increased, and how many vals at this id
func (r *Tree[T]) Set(id ID, val T) (bool, int) {
	b, idx := r.add(id, val,
		func(T, T) bool { return true },
		func(T) T { return val })

	//fmt.Println("set", r.name, id, b, idx)
	return b, idx
}

// Add adds a tag to the tree
// - if matchFunc is non-nil, it will be used to ensure uniqueness at this node
// - returns whether the val count at this address was increased, and how many vals at this address
func (r *Tree[T]) Add(id ID, val T, matchFunc MatchesFunc[T]) (bool, int) {
	return r.add(id, val, matchFunc, nil)
}

func (r *Tree[T]) add(id ID, val T, matchFunc MatchesFunc[T], updateFunc UpdatesFunc[T]) (bool, int) {

	// make sure we have more than enough capacity before we start adding to the tree, which invalidates pointers into the array
	if (len(r.availableIndexes) + cap(r.nodes)) < (len(r.nodes) + 10) {
		temp := make([]treeNode[T], len(r.nodes), (cap(r.nodes)+1)*2)
		copy(temp, r.nodes)
		r.nodes = temp
	}

	root := &r.nodes[1]

	// handle root tags
	if id.Length() == 0 {
		countIncreased := r.addVal(val, 1, matchFunc, updateFunc)
		return countIncreased, r.nodes[1].ValCount
	}

	// root node doesn't have any id, so find the starting point
	nodeIndex := uint(0)
	parent := root
	if !id.IsLeftBitSet() {
		if root.Left == 0 {
			newNodeIndex := r.newNode(id, id.Length())
			countIncreased := r.addVal(val, newNodeIndex, matchFunc, updateFunc)
			root.Left = newNodeIndex
			return countIncreased, r.nodes[newNodeIndex].ValCount
		}
		nodeIndex = root.Left
	} else {
		if root.Right == 0 {
			newNodeIndex := r.newNode(id, id.Length())
			countIncreased := r.addVal(val, newNodeIndex, matchFunc, updateFunc)
			root.Right = newNodeIndex
			return countIncreased, r.nodes[newNodeIndex].ValCount
		}
		nodeIndex = root.Right
	}

	for {
		if nodeIndex == 0 {
			panic("Trying to traverse nodeIndex=0")
		}
		node := &r.nodes[nodeIndex]
		if node.Length == 0 {
			panic("Reached a node with no id")
		}
		matchCount := node.MatchCount(id)
		if matchCount == 0 {
			panic(fmt.Sprintf("tree %s Should not have traversed to a node with no prefix match - node length: %d; id length: %d id %v", r.name, node.Length, id.Length(), id))
		}

		if matchCount == id.Length() {
			// all the bits in the address matched

			if matchCount == node.Length {
				// the whole prefix matched - we're done!
				countIncreased := r.addVal(val, nodeIndex, matchFunc, updateFunc)
				return countIncreased, r.nodes[nodeIndex].ValCount
			}

			// the input id is shorter than the match found - need to create a new, intermediate parent
			newNodeIndex := r.newNode(id, id.Length())
			newNode := &r.nodes[newNodeIndex]
			countIncreased := r.addVal(val, newNodeIndex, matchFunc, updateFunc)

			// the existing node loses those matching bits, and becomes a child of the new node

			// shift
			node.ShiftLength(matchCount)

			if !r.isLeftBitSetFn(node.Id) {
				newNode.Left = nodeIndex
			} else {
				newNode.Right = nodeIndex
			}

			// now give this new node a home
			if parent.Left == nodeIndex {
				parent.Left = newNodeIndex
			} else {
				if parent.Right != nodeIndex {
					panic("node isn't left or right parent - should be impossible! (1)")
				}
				parent.Right = newNodeIndex
			}
			return countIncreased, r.nodes[newNodeIndex].ValCount

		}

		if matchCount == node.Length {
			// partial match - we have to keep traversing

			// chop off what's matched so far
			id = id.ShiftLeft(matchCount)

			if !id.IsLeftBitSet() {
				if node.Left == 0 {
					// nowhere else to go - create a new node here
					newNodeIndex := r.newNode(id, id.Length())
					countIncreased := r.addVal(val, newNodeIndex, matchFunc, updateFunc)
					node.Left = newNodeIndex
					return countIncreased, r.nodes[newNodeIndex].ValCount
				}

				// there's a node to the left - traverse it
				parent = node
				nodeIndex = node.Left
				continue
			}

			// node didn't belong on the left, so it belongs on the right
			if node.Right == 0 {
				// nowhere else to go - create a new node here
				newNodeIndex := r.newNode(id, id.Length())
				countIncreased := r.addVal(val, newNodeIndex, matchFunc, updateFunc)
				node.Right = newNodeIndex
				return countIncreased, r.nodes[newNodeIndex].ValCount
			}

			// there's a node to the right - traverse it
			parent = node
			nodeIndex = node.Right
			continue
		}

		// partial match with this node - need to split this node
		newCommonParentNodeIndex := r.newNode(id, matchCount)
		newCommonParentNode := &r.nodes[newCommonParentNodeIndex]

		// shift
		id = id.ShiftLeft(matchCount)

		newNodeIndex := r.newNode(id, id.Length())
		countIncreased := r.addVal(val, newNodeIndex, matchFunc, updateFunc)

		// see where the existing node fits - left or right
		node.ShiftLength(matchCount)
		if !r.isLeftBitSetFn(node.Id) {
			newCommonParentNode.Left = nodeIndex
			newCommonParentNode.Right = newNodeIndex
		} else {
			newCommonParentNode.Right = nodeIndex
			newCommonParentNode.Left = newNodeIndex
		}

		// now determine where the new node belongs
		if parent.Left == nodeIndex {
			parent.Left = newCommonParentNodeIndex
		} else {
			if parent.Right != nodeIndex {
				panic("node isn't left or right parent - should be impossible! (2)")
			}
			parent.Right = newCommonParentNodeIndex
		}
		return countIncreased, r.nodes[newNodeIndex].ValCount

	}
}

// Delete a tag from the tree if it matches matchVal, as determined by matchFunc. Returns how many tags are removed
// - use DeleteWithBuffer if you can reuse slices, to cut down on allocations
func (r *Tree[T]) Delete(id ID, matchFunc MatchesFunc[T], matchVal T) int {
	return r.DeleteWithBuffer(nil, id, matchFunc, matchVal)
}

// DeleteWithBuffer a tag from the tree if it matches matchVal, as determined by matchFunc. Returns how many tags are removed
// - uses input slice to reduce allocations
func (r *Tree[T]) DeleteWithBuffer(buf []T, id ID, matchFunc MatchesFunc[T], matchVal T) int {
	// traverse the tree, finding the node and its parent
	root := &r.nodes[1]
	var parentIndex uint
	var parent *treeNode[T]
	var targetNode *treeNode[T]
	var targetNodeIndex uint

	if id.Length() == 0 {
		// caller just looking for root tags
		targetNode = root
		targetNodeIndex = 1
	} else {
		nodeIndex := uint(0)

		parentIndex = 1
		parent = root
		if !id.IsLeftBitSet() {
			nodeIndex = root.Left
		} else {
			nodeIndex = root.Right
		}

		// traverse the tree
		for {
			if nodeIndex == 0 {
				return 0
			}

			node := &r.nodes[nodeIndex]
			matchCount := node.MatchCount(id)
			if matchCount < node.Length {
				// didn't match the entire node - we're done
				return 0
			}

			if matchCount == id.Length() {
				// exact match - we're done
				targetNode = node
				targetNodeIndex = nodeIndex
				break
			}

			// there's still more address - keep traversing
			parentIndex = nodeIndex
			parent = node
			id = id.ShiftLeft(matchCount)
			if !id.IsLeftBitSet() {
				nodeIndex = node.Left
			} else {
				nodeIndex = node.Right
			}
		}
	}

	if targetNode == nil || targetNode.ValCount == 0 {
		// no tags found
		return 0
	}

	// delete matching tags
	deleteCount, remainingTagCount := r.deleteVal(buf, targetNodeIndex, matchVal, matchFunc)
	if remainingTagCount > 0 {
		// target node still has tags - we're not deleting it
		return deleteCount
	}
	r.deleteNode(targetNodeIndex, targetNode, parentIndex, parent)
	return deleteCount
}

// deleteNode removes the provided node and compact the tree.
func (r *Tree[T]) deleteNode(targetNodeIndex uint, targetNode *treeNode[T], parentIndex uint, parent *treeNode[T]) (result deleteNodeResult) {
	result = notDeleted
	if targetNodeIndex == 1 {
		// can't delete the root node
		return result
	}

	// compact the tree, if possible
	if targetNode.Left != 0 && targetNode.Right != 0 {
		// target has two children - nothing we can do - not deleting the node
		return result
	} else if targetNode.Left != 0 {
		// target node only has only left child
		result = deletedNodeReplacedByChild
		if parent.Left == targetNodeIndex {
			parent.Left = targetNode.Left
		} else {
			parent.Right = targetNode.Left
		}

		// need to update the child node prefix to include target node's
		tmpNode := &r.nodes[targetNode.Left]
		tmpNode.MergeFromNodes(targetNode, tmpNode, r.length)
	} else if targetNode.Right != 0 {
		// target node has only right child
		result = deletedNodeReplacedByChild
		if parent.Left == targetNodeIndex {
			parent.Left = targetNode.Right
		} else {
			parent.Right = targetNode.Right
		}

		// need to update the child node prefix to include target node's
		tmpNode := &r.nodes[targetNode.Right]
		tmpNode.MergeFromNodes(targetNode, tmpNode, r.length)
	} else {
		// target node has no children - straight-up remove this node
		result = deletedNodeJustRemoved
		if parent.Left == targetNodeIndex {
			parent.Left = 0
			if parentIndex > 1 && parent.ValCount == 0 && parent.Right != 0 {
				// parent isn't root, has no tags, and there's a sibling - merge sibling into parent
				result = deletedNodeParentReplacedBySibling
				siblingIndexToDelete := parent.Right
				tmpNode := &r.nodes[siblingIndexToDelete]
				parent.MergeFromNodes(parent, tmpNode, r.length)

				// move tags
				r.moveTags(siblingIndexToDelete, parentIndex)

				// parent now gets target's sibling's children
				parent.Left = r.nodes[siblingIndexToDelete].Left
				parent.Right = r.nodes[siblingIndexToDelete].Right

				r.availableIndexes = append(r.availableIndexes, siblingIndexToDelete)
			}
		} else {
			parent.Right = 0
			if parentIndex > 1 && parent.ValCount == 0 && parent.Left != 0 {
				// parent isn't root, has no tags, and there's a sibling - merge sibling into parent
				result = deletedNodeParentReplacedBySibling
				siblingIndexToDelete := parent.Left
				tmpNode := &r.nodes[siblingIndexToDelete]
				parent.MergeFromNodes(parent, tmpNode, r.length)

				// move tags
				r.moveTags(siblingIndexToDelete, parentIndex)

				// parent now gets target's sibling's children
				parent.Right = r.nodes[parent.Left].Right
				parent.Left = r.nodes[parent.Left].Left

				r.availableIndexes = append(r.availableIndexes, siblingIndexToDelete)
			}
		}
	}

	targetNode.Left = 0
	targetNode.Right = 0
	r.availableIndexes = append(r.availableIndexes, targetNodeIndex)
	return result
}

// TreeIteratorV4[T] is a stateful iterator over a tree.
type TreeIterator[T any] struct {
	length      uint8 // input to determine which mask to apply
	t           *Tree[T]
	nodeIndex   uint
	nodeHistory []uint
	next        treeIteratorNext
}

// Iterate returns an iterator to find all nodes from a tree. It is
// important for the tree to not be modified while using the iterator.
func (r *Tree[T]) Iterate() *TreeIterator[T] {
	return &TreeIterator[T]{
		length:      r.length, // input to determine which mask to apply
		t:           r,
		nodeIndex:   1,
		nodeHistory: []uint{},
		next:        nextSelf,
	}
}

// Next jumps to the next element of a tree. It returns false if there
// is none.
func (iter *TreeIterator[T]) Next() bool {
	for {
		node := &iter.t.nodes[iter.nodeIndex]
		if iter.next == nextSelf {
			iter.next = nextLeft
			if node.ValCount != 0 {
				return true
			}
		}
		if iter.next == nextLeft {
			if node.Left != 0 {
				iter.nodeHistory = append(iter.nodeHistory, iter.nodeIndex)
				iter.nodeIndex = node.Left
				iter.next = nextSelf
			} else {
				iter.next = nextRight
			}
		}
		if iter.next == nextRight {
			if node.Right != 0 {
				iter.nodeHistory = append(iter.nodeHistory, iter.nodeIndex)
				iter.nodeIndex = node.Right
				iter.next = nextSelf
			} else {
				// We need to backtrack
				iter.next = nextUp
			}
		}
		if iter.next == nextUp {
			nodeHistoryLen := len(iter.nodeHistory)
			if nodeHistoryLen == 0 {
				return false
			}
			previousIndex := iter.nodeHistory[nodeHistoryLen-1]
			previousNode := iter.t.nodes[previousIndex]
			iter.nodeHistory = iter.nodeHistory[:nodeHistoryLen-1]
			if previousNode.Left == iter.nodeIndex {
				iter.nodeIndex = previousIndex
				iter.next = nextRight
			} else if previousNode.Right == iter.nodeIndex {
				iter.nodeIndex = previousIndex
				iter.next = nextUp
			} else {
				panic("unexpected state")
			}
		}
	}
}

// Tags returns the current tags for the iterator. This is not a copy
// and the result should not be used outside the iterator.
func (iter *TreeIterator[T]) Vals() []T {
	return iter.ValsWithBuffer(nil)
}

// TagsWithBuffer returns the current tags for the iterator. To avoid
// allocation, it uses the provided buffer.
func (iter *TreeIterator[T]) ValsWithBuffer(ret []T) []T {
	return iter.t.valsForNode(ret, uint(iter.nodeIndex), nil)
}

// note: this is only used for unit testing
// nolint
func (r *Tree[T]) PrintNodes(nodeIndex uint) {
	for id, n := range r.nodes {
		fmt.Println("node", id, n)
	}
}

// note: this is only used for unit testing
// nolint
func (r *Tree[T]) PrintValues() {
	for id, v := range r.vals {
		fmt.Println("val", id, v)
	}
}
