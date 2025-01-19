package treekem

import (
	"crypto/rand"
)

type TreeKEMState struct {
	tkem *TreeKEM
}

type GroupInitKey struct {
	Size     int
	Frontier map[int]*Node
}

type GroupAddForJoiner struct {
	Size          int
	EncryptedLeaf ECKEMCipherText
	Frontier      map[int]*Node
	Path          map[int]*Node
}

type GroupAddForGroup struct {
	Size        int
	Ciphertexts []map[int]ECKEMCipherText
	Nodes       map[int]*Node
}

type UserAdd struct {
	Size        int
	Ciphertexts []map[int]ECKEMCipherText
	Nodes       map[int]*Node
}

type UserUpdate struct {
	From        int
	Ciphertexts []map[int]ECKEMCipherText
	Nodes       map[int]*Node
}

type UserRemove struct {
	Index       int
	Ciphertexts []map[int]ECKEMCipherText
	Copath      map[int]*Node
}

type UserMove struct {
	From        int
	To          int
	Ciphertexts []map[int]ECKEMCipherText
	Nodes       map[int]*Node
	Copath      map[int]*Node
}

func NewTreeKEMState() *TreeKEMState {
	return &TreeKEMState{
		tkem: NewTreeKEM(),
	}
}

func (t *TreeKEMState) Index() int {
	return t.tkem.Index
}

func (t *TreeKEMState) Size() int {
	return t.tkem.Size
}

func (t *TreeKEMState) Trim(size int) {
	t.tkem.trim(size)
}

func (t *TreeKEMState) Nodes() map[int]*Node {
	return t.tkem.Nodes
}

func (t *TreeKEMState) Copath() map[int]*Node {
	return t.tkem.copath(t.tkem.Index)
}

func (t *TreeKEMState) Equal(other *TreeKEMState) bool {
	return t.tkem.equal(other.tkem)
}

func TreeKEMStateOneMemberGroup(leaf []byte) *TreeKEMState {
	state := NewTreeKEMState()
	tkem := oneMemberGroup(leaf)
	state.tkem = tkem
	return state
}

func TreeKEMStateFromGroupAdd(initLeaf []byte, groupAdd GroupAddForJoiner) (*TreeKEMState, error) {
	kp, err := NewKeyPairFromSecret(initLeaf)
	if err != nil {
		return nil, err
	}

	leaf, err := ECKEMDecrypt(groupAdd.EncryptedLeaf, kp.Private.Bytes())
	if err != nil {
		return nil, err
	}

	state := NewTreeKEMState()
	state.tkem = FromFrontier(groupAdd.Size, groupAdd.Frontier, leaf)
	return state, nil
}

func TreeKEMStateFromUserAdd(leaf []byte, groupInitKey GroupInitKey) (*TreeKEMState, error) {
	state := NewTreeKEMState()
	state.tkem = FromFrontier(groupInitKey.Size, groupInitKey.Frontier, leaf)
	return state, nil
}

func TreeKEMStateJoin(leaf []byte, groupInitKey GroupInitKey) (UserAdd, error) {
	tkem := FromFrontier(groupInitKey.Size, groupInitKey.Frontier, leaf)
	ct := tkem.Encrypt(leaf, tkem.Index)
	ua := UserAdd{
		Size:        tkem.Size,
		Ciphertexts: ct.Ciphertexts,
		Nodes:       ct.Nodes,
	}
	return ua, nil
}

func (t *TreeKEMState) Add(userInitPub []byte) (GroupAddForGroup, GroupAddForJoiner, error) {
	leaf := make([]byte, 32)
	_, err := rand.Read(leaf)
	if err != nil {
		return GroupAddForGroup{}, GroupAddForJoiner{}, err
	}

	encryptedLeaf, err := ECKEMEncrypt(leaf, userInitPub)
	if err != nil {
		return GroupAddForGroup{}, GroupAddForJoiner{}, err
	}

	gik := t.GroupInitKey()
	ua, err := TreeKEMStateJoin(leaf, gik)
	if err != nil {
		return GroupAddForGroup{}, GroupAddForJoiner{}, err
	}

	groupAddForGroup := GroupAddForGroup{
		Size:        t.Size(),
		Ciphertexts: ua.Ciphertexts,
		Nodes:       ua.Nodes,
	}

	groupAddForJoiner := GroupAddForJoiner{
		Size:          t.Size(),
		EncryptedLeaf: encryptedLeaf,
		Frontier:      t.tkem.frontier(),
	}

	return groupAddForGroup, groupAddForJoiner, nil
}

func (t *TreeKEMState) Update(leaf []byte) UserUpdate {
	ct := t.tkem.Encrypt(leaf, t.tkem.Index)
	return UserUpdate{
		From:        t.tkem.Index,
		Ciphertexts: ct.Ciphertexts,
		Nodes:       ct.Nodes,
	}
}

func (t *TreeKEMState) Remove(leaf []byte, index int, copath map[int]*Node) UserRemove {
	t.tkem.merge(copath, true)
	ct := t.tkem.Encrypt(leaf, index)
	return UserRemove{
		Index:       index,
		Ciphertexts: ct.Ciphertexts,
		Copath:      t.tkem.copath(index),
	}
}

func (t *TreeKEMState) Move(leaf []byte, index int, copath map[int]*Node) UserMove {
	t.tkem.merge(copath, true)
	ct := t.tkem.Encrypt(leaf, index)
	return UserMove{
		From:        t.tkem.Index,
		To:          index,
		Ciphertexts: ct.Ciphertexts,
		Nodes:       ct.Nodes,
		Copath:      t.tkem.copath(index),
	}
}

func (t *TreeKEMState) GroupInitKey() GroupInitKey {
	return GroupInitKey{
		Size:     t.Size(),
		Frontier: t.tkem.frontier(),
	}
}

// RootPublic returns the public key of the root node.
func (t *TreeKEMState) RootPublic() []byte {
	return t.tkem.Nodes[root(t.tkem.Size)].Public
}

// RootSignPublic returns the public key of the root node.
func (t *TreeKEMState) RootSignPublic() []byte {
	return t.tkem.Nodes[root(t.tkem.Size)].SignPublic
}

// RootPrivate returns the private key of the root node.
func (t *TreeKEMState) RootPrivate() []byte {
	return t.tkem.Nodes[root(t.tkem.Size)].Private
}

// Self return the self node.
func (t *TreeKEMState) Self() *Node {
	return t.tkem.Nodes[t.tkem.Index*2]
}

func (t *TreeKEMState) HandleUserAdd(ua UserAdd) {
	pt := t.tkem.Decrypt(t.tkem.Size, ua.Ciphertexts)
	t.tkem.merge(ua.Nodes, false)
	t.tkem.merge(pt.Nodes, false)
	t.tkem.Size += 1
}

func (t *TreeKEMState) HandleGroupAdd(ga GroupAddForGroup) {
	pt := t.tkem.Decrypt(t.tkem.Size, ga.Ciphertexts)
	t.tkem.merge(ga.Nodes, false)
	t.tkem.merge(pt.Nodes, false)
	t.tkem.Size += 1
}

func (t *TreeKEMState) HandleSelfUpdate(update UserUpdate, leaf []byte) {
	privateNodes := hashUp(2*t.tkem.Index, t.tkem.Size, leaf)
	t.tkem.merge(privateNodes, false)
}

func (t *TreeKEMState) HandleUpdate(update UserUpdate) {
	pt := t.tkem.Decrypt(update.From, update.Ciphertexts)
	t.tkem.merge(update.Nodes, false)
	t.tkem.merge(pt.Nodes, false)
}

func (t *TreeKEMState) HandleRemove(remove UserRemove) {
	pt := t.tkem.Decrypt(remove.Index, remove.Ciphertexts)
	t.tkem.remove(remove.Index)
	t.tkem.merge(pt.Root, false)
	t.tkem.merge(remove.Copath, true)
}

func (t *TreeKEMState) HandleSelfMove(move UserMove, leaf []byte) {
	privateNodes := hashUp(2*move.To, t.tkem.Size, leaf)
	t.tkem.remove(move.From)
	t.tkem.merge(privateNodes, false)
	t.tkem.merge(move.Copath, true)
	t.tkem.Index = move.To
}

func (t *TreeKEMState) HandleMove(move UserMove) {
	pt := t.tkem.Decrypt(move.To, move.Ciphertexts)
	t.tkem.remove(move.From)
	t.tkem.merge(move.Nodes, false)
	t.tkem.merge(move.Copath, true)
	t.tkem.merge(pt.Nodes, false)
}
