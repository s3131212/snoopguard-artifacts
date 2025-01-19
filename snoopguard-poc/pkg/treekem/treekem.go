package treekem

import (
	"crypto/sha256"
)

// TreeKEM is a struct representing a TreeKEM object.
type TreeKEM struct {
	Size  int
	Index int
	Nodes map[int]*Node
}

// Node is a struct representing a TreeKEM node.
type Node struct {
	Secret      []byte
	Public      []byte
	Private     []byte
	SignPublic  []byte
	SignPrivate []byte
}

func NewTreeKEM() *TreeKEM {
	return &TreeKEM{
		Size:  0,
		Index: 0,
		Nodes: make(map[int]*Node),
	}
}

// hash function
func hash(x64 []byte) []byte {
	h := sha256.New()
	h.Write(x64)
	return h.Sum(nil)
}

// hashUp function
func hashUp(index, size int, h []byte) map[int]*Node {
	nodes := make(map[int]*Node)
	n := index
	root := root(size)
	path := []int{n}

	for {
		kp, err := NewKeyPairFromSecret(h)
		kpSign, err := NewSigningKeyPairFromSecret(h)
		if err != nil {
			panic(err)
		}
		nodes[n] = &Node{
			Secret:      h,
			Public:      kp.Public.Bytes(),
			Private:     kp.Private.Bytes(),
			SignPublic:  kpSign.Public.Bytes(),
			SignPrivate: kpSign.Private.Bytes(),
		}

		if n == root {
			break
		}

		n = parent(n, size)
		path = append(path, n)
		h = hash(h)
	}

	return nodes
}

/*
 * oneMemberGroup constructs a TreeKEM representing a group with a single member, with the given leaf secret.
 */
func oneMemberGroup(leaf []byte) *TreeKEM {
	tkem := &TreeKEM{
		Size:  1,
		Index: 0,
	}
	tkem.Nodes = make(map[int]*Node)
	tkem.merge(hashUp(0, 1, leaf), false)
	return tkem
}

// FromFrontier constructs a tree that extends a tree with the given size and frontier by adding a member with the given leaf secret.
func FromFrontier(size int, frontier map[int]*Node, leaf []byte) *TreeKEM {
	tkem := &TreeKEM{
		Size:  size + 1,
		Index: size,
	}
	tkem.Nodes = make(map[int]*Node)
	tkem.merge(frontier, false)

	nodes := hashUp(2*tkem.Index, tkem.Size, leaf)
	tkem.merge(nodes, false)
	return tkem
}

// EncryptToSubtree encrypts a value so that it can be decrypted by all nodes in the subtree with the indicated head.
func (tkem *TreeKEM) EncryptToSubtree(head int, value []byte) map[int]ECKEMCipherText {
	encryptions := make(map[int]ECKEMCipherText)

	if node, ok := tkem.Nodes[head]; ok {
		enc, err := ECKEMEncrypt(value, node.Public)
		if err != nil {
			panic(err)
		}
		encryptions[head] = enc
		return encryptions
	}

	if left := left(head); left != head {
		leftEncryptions := tkem.EncryptToSubtree(left, value)
		for k, v := range leftEncryptions {
			encryptions[k] = v
		}
	}

	if right := right(head, tkem.Size); right != head {
		rightEncryptions := tkem.EncryptToSubtree(right, value)
		for k, v := range rightEncryptions {
			encryptions[k] = v
		}
	}

	return encryptions
}

// GatherSubtree gather the heads of the populated subtrees below the specified subtree head.
func (tkem *TreeKEM) GatherSubtree(head int) map[int]*Node {
	subtreeNodes := make(map[int]*Node)

	if node, ok := tkem.Nodes[head]; ok {
		subtreeNodes[head] = node
		return subtreeNodes
	}

	left := left(head)
	if left != head {
		leftNodes := tkem.GatherSubtree(left)
		for k, v := range leftNodes {
			subtreeNodes[k] = v
		}
	}

	right := right(head, tkem.Size)
	if right != head {
		rightNodes := tkem.GatherSubtree(right)
		for k, v := range rightNodes {
			subtreeNodes[k] = v
		}
	}

	return subtreeNodes
}

/*
Encrypt function encrypts a fresh root value in a way that all participants in the group can decrypt, except for an excluded node.

	Returns: {
	     index: Int // Index of the sender in the tree
	     nodes: { Int: Node } // Public nodes along the direct path
	     privateNodes: { Int: Node } // Private nodes along the direct path
	     ciphertexts: [ ECKEMCiphertext ] // Ciphertexts along the copath
	}
*/
func (tkem *TreeKEM) Encrypt(leaf []byte, except int) *TreeKEMCiphertext {
	//trDirpath := trDirpath(2*except, tkem.Size)
	copath := trCopath(2*except, tkem.Size)

	// Generate hashes up the tree
	privateNodes := hashUp(2*except, tkem.Size, leaf)
	nodes := make(map[int]*Node)
	for n, node := range privateNodes {
		nodes[n] = &Node{
			Public:     node.Public,
			SignPublic: node.SignPublic,
		}
	}

	// KEM each hash to the corresponding copath node
	ciphertexts := make([]map[int]ECKEMCipherText, 0)
	for _, c := range copath {
		p := parent(c, tkem.Size)
		s := privateNodes[p].Secret
		encryptions := tkem.EncryptToSubtree(c, s)
		ciphertexts = append(ciphertexts, encryptions)
	}

	return &TreeKEMCiphertext{
		Index:        tkem.Index,
		Nodes:        nodes,
		PrivateNodes: privateNodes,
		Ciphertexts:  ciphertexts,
	}
}

/*
Decrypt function decrypts and returns fresh root value.

	Returns: {
	    root: ArrayBuffer // The root hash for the tree
	    nodes: { Int: Node } // Public nodes resulting from hashes on the direct path
	}
*/
func (tkem *TreeKEM) Decrypt(index int, ciphertexts []map[int]ECKEMCipherText) *DecryptionResult {
	senderSize := tkem.Size
	if index == tkem.Size {
		senderSize++
	}

	copath := trCopath(2*index, senderSize)
	dirpath := append(trDirpath(2*tkem.Index, tkem.Size), root(tkem.Size))

	// Decrypt at the point where the dirpath and copath overlap
	overlap := -1
	var coIndex, dirIndex int
	for i, d := range dirpath {
		for j, c := range copath {
			if d == c {
				overlap = d
				coIndex = j
				dirIndex = i
				break
			}
		}
		if overlap != -1 {
			break
		}
	}

	// Extract an encrypted value that we can decrypt, and decrypt it
	encryptions := ciphertexts[coIndex]
	decNode := -1
	for k := range encryptions {
		if containsInt(dirpath, k) {
			decNode = k
			break
		}
	}

	if decNode == -1 {
		panic("Decrypt fail")
	}

	h, err := ECKEMDecrypt(encryptions[decNode], tkem.Nodes[decNode].Private)
	if err != nil {
		panic(err)
	}

	rootNode := root(senderSize)
	newDirpath := append(trDirpath(2*tkem.Index, senderSize), rootNode)
	nodes := hashUp(newDirpath[dirIndex+1], senderSize, h)

	root := make(map[int]*Node)
	root[rootNode] = nodes[rootNode]

	return &DecryptionResult{
		Root:  root,
		Nodes: nodes,
	}
}

/*
DecryptRaw function decrypt the ciphertext and return the plaintext directly.
*/
func (tkem *TreeKEM) DecryptRaw(index int, ciphertexts []map[int]ECKEMCipherText) []byte {
	senderSize := tkem.Size
	if index == tkem.Size {
		senderSize++
	}

	copath := trCopath(2*index, senderSize)
	dirpath := append(trDirpath(2*tkem.Index, tkem.Size), root(tkem.Size))

	// Decrypt at the point where the dirpath and copath overlap
	overlap := -1
	var coIndex int
	for _, d := range dirpath {
		for j, c := range copath {
			if d == c {
				overlap = d
				coIndex = j
				//dirIndex = i
				break
			}
		}
		if overlap != -1 {
			break
		}
	}

	// Extract an encrypted value that we can decrypt, and decrypt it
	encryptions := ciphertexts[coIndex]
	decNode := -1
	for k := range encryptions {
		if containsInt(dirpath, k) {
			decNode = k
			break
		}
	}

	if decNode == -1 {
		panic("Decrypt fail")
	}

	h, err := ECKEMDecrypt(encryptions[decNode], tkem.Nodes[decNode].Private)
	if err != nil {
		panic(err)
	}

	return h
}

// trim removes unnecessary nodes from the tree when the size of the group shrinks.
func (tkem *TreeKEM) trim(size int) {
	if size > tkem.Size {
		panic("Cannot trim upwards")
	}

	width := nodeWidth(size)
	newNodes := make(map[int]*Node, width)
	for i := 0; i < width; i++ {
		if node, ok := tkem.Nodes[i]; ok {
			newNodes[i] = node
		}
	}

	tkem.Nodes = newNodes
	tkem.Size = size
}

// remove function removes a node from the tree, including its direct path
func (tkem *TreeKEM) remove(index int) {
	dirpath := trDirpath(2*index, tkem.Size)
	for _, n := range dirpath {
		delete(tkem.Nodes, n)
	}
}

// merge updates nodes in the tree.
// nodes - Dictionary of nodes to update: { Int: Node }
// reserve - Whether existing nodes should be left alone
func (tkem *TreeKEM) merge(nodes map[int]*Node, preserve bool) {
	for n, node := range nodes {
		if _, ok := tkem.Nodes[n]; ok && preserve {
			continue
		}
		tkem.Nodes[n] = node
	}
}

// frontier returns the nodes on the frontier of the tree { Int: Node }, including subtree heads if the tree is incomplete.
func (tkem *TreeKEM) frontier() map[int]*Node {
	frontierNodes := make(map[int]*Node)
	frontiers := frontier(tkem.Size)
	for _, n := range frontiers {
		subtreeNodes := tkem.GatherSubtree(n)
		for k, v := range subtreeNodes {
			frontierNodes[k] = v
		}
	}
	return frontierNodes
}

// copath returns the nodes on the copath for this node { Int: Node }, including subtree heads if the tree is incomplete.
func (tkem *TreeKEM) copath(index int) map[int]*Node {
	copathNodes := make(map[int]*Node)
	copath := trCopath(2*index, tkem.Size)
	for _, n := range copath {
		subtreeNodes := tkem.GatherSubtree(n)
		for k, v := range subtreeNodes {
			copathNodes[k] = v
		}
	}
	return copathNodes
}

// Equal function in Go
func (tkem *TreeKEM) equal(other *TreeKEM) bool {
	if tkem.Size != other.Size {
		return false
	}

	// Iterate through all tkem.Nodes
	for k, node := range tkem.Nodes {
		// If a node in tkem.Nodes is not in other.Nodes, continue
		otherNode, ok := other.Nodes[k]
		if !ok {
			continue
		}

		fp1 := fingerprint(node.Public)
		fp2 := fingerprint(otherNode.Public)
		if fp1 != fp2 {
			return false
		}
	}

	return true
}

// Helper functions

func fingerprint(publicKey []byte) string {
	// Implement the fingerprint function (not provided in the code)
	// This function should calculate a fingerprint of the public key.
	// Return the fingerprint as a string.

	// This is a dummy implementation that returns the public key as a string.
	return string(publicKey)
}

func containsInt(slice []int, val int) bool {
	// Implement the containsInt function (not provided in the code)
	// This function should check if the given slice contains the given value.
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

// TreeKEMCiphertext is a struct representing a TreeKEM ciphertext.
type TreeKEMCiphertext struct {
	Index        int
	Nodes        map[int]*Node
	PrivateNodes map[int]*Node
	Ciphertexts  []map[int]ECKEMCipherText
}

// DecryptionResult is a struct representing the result of decryption.
type DecryptionResult struct {
	Root  map[int]*Node
	Nodes map[int]*Node
}
