package treekem

import (
	"errors"
)

// MultiTreeKEM is a struct that contains a treeKEM, the additional node, and waited-to-be-sent update message.
type MultiTreeKEM struct {
	treekem          *TreeKEMState
	externalNodes    map[string]Node
	roots            map[string]Node
	lastTreeKemRoots map[string]Node
}

// NewMultiTreeKEM creates a new MultiTreeKEM object.
func NewMultiTreeKEM(tk *TreeKEMState) *MultiTreeKEM {
	return &MultiTreeKEM{
		treekem:          tk,
		externalNodes:    make(map[string]Node),
		roots:            make(map[string]Node),
		lastTreeKemRoots: make(map[string]Node),
	}
}

// AddExternalNode adds an external node to the multi-treekem.
func (m *MultiTreeKEM) AddExternalNode(id string, ct ECKEMCipherText) error {
	if _, ok := m.externalNodes[id]; ok {
		return errors.New("id already exist")
	}

	initLeaf, err := ECKEMDecrypt(ct, m.treekem.Nodes()[root(m.treekem.Size())].Private)
	if err != nil {
		return err
	}
	initKp, _ := NewKeyPairFromSecret(initLeaf)
	initKpSign, _ := NewSigningKeyPairFromSecret(initLeaf)

	m.externalNodes[id] = Node{
		Public:     initKp.Public.Bytes(),
		SignPublic: initKpSign.Public.Bytes(),
	}

	rootSecret := hash(initLeaf)
	rootKp, err := NewKeyPairFromSecret(rootSecret)
	rootKpSign, err := NewSigningKeyPairFromSecret(rootSecret)
	if err != nil {
		return err
	}
	m.roots[id] = Node{
		Secret:      rootSecret,
		Public:      rootKp.Public.Bytes(),
		Private:     rootKp.Private.Bytes(),
		SignPublic:  rootKpSign.Public.Bytes(),
		SignPrivate: rootKpSign.Private.Bytes(),
	}

	// Copy the current treekem root
	m.lastTreeKemRoots[id] = Node{
		Public:      m.treekem.Nodes()[root(m.treekem.Size())].Public,
		Private:     m.treekem.Nodes()[root(m.treekem.Size())].Private,
		SignPublic:  m.treekem.Nodes()[root(m.treekem.Size())].SignPublic,
		SignPrivate: m.treekem.Nodes()[root(m.treekem.Size())].SignPrivate,
		Secret:      m.treekem.Nodes()[root(m.treekem.Size())].Secret,
	}

	return nil
}

// GetExternalNodeJoin generates the external node join message.
func (m *MultiTreeKEM) GetExternalNodeJoin(id string) (ECKEMCipherText, []byte, error) {
	initLeaf, err := GenerateRandomBytes(32)
	if err != nil {
		return ECKEMCipherText{}, nil, err
	}

	initKp, err := NewKeyPairFromSecret(initLeaf)
	if err != nil {
		return ECKEMCipherText{}, nil, err
	}
	ct, err := ECKEMEncrypt(initLeaf, m.treekem.Nodes()[root(m.treekem.Size())].Public)
	if err != nil {
		return ECKEMCipherText{}, nil, err
	}

	rootSecret := hash(initLeaf)
	rootKp, err := NewKeyPairFromSecret(rootSecret)
	rootSignKp, err := NewSigningKeyPairFromSecret(rootSecret)
	if err != nil {
		return ECKEMCipherText{}, nil, err
	}

	m.externalNodes[id] = Node{Public: initKp.Public.Bytes()}
	m.roots[id] = Node{
		Secret:      rootSecret,
		Public:      rootKp.Public.Bytes(),
		Private:     rootKp.Private.Bytes(),
		SignPublic:  rootSignKp.Public.Bytes(),
		SignPrivate: rootSignKp.Private.Bytes(),
	}
	m.lastTreeKemRoots[id] = Node{
		Public:      m.treekem.Nodes()[root(m.treekem.Size())].Public,
		Private:     m.treekem.Nodes()[root(m.treekem.Size())].Private,
		SignPublic:  m.treekem.Nodes()[root(m.treekem.Size())].SignPublic,
		SignPrivate: m.treekem.Nodes()[root(m.treekem.Size())].SignPrivate,
		Secret:      m.treekem.Nodes()[root(m.treekem.Size())].Secret,
	}

	return ct, initLeaf, nil
}

// GetExternalNodeJoinsWithoutUpdate generates the external node join message for all external nodes, encrypted with pubKey, without updating any existing node.
func (m *MultiTreeKEM) GetExternalNodeJoinsWithoutUpdate(pubKey []byte) (map[string][]byte, map[string][]byte, map[string]ECKEMCipherText, error) {
	chatbotPubKeys := make(map[string][]byte)
	chatbotSignPubKeys := make(map[string][]byte)
	lastTreeKemRootCiphertexts := make(map[string]ECKEMCipherText)
	for id, _ := range m.externalNodes {
		chatbotPubKeys[id] = m.externalNodes[id].Public
		chatbotSignPubKeys[id] = m.externalNodes[id].SignPublic

		ct, err := ECKEMEncrypt(m.lastTreeKemRoots[id].Secret, pubKey)
		if err != nil {
			return nil, nil, nil, err
		}
		lastTreeKemRootCiphertexts[id] = ct
	}

	return chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts, nil
}

// SetExternalNodeJoinsWithoutUpdate sets the external node join message for all external nodes without updating any existing node.
func (m *MultiTreeKEM) SetExternalNodeJoinsWithoutUpdate(chatbotPubKeys map[string][]byte, chatbotSignPubKeys map[string][]byte, lastTreeKemRootCiphertexts map[string]ECKEMCipherText) error {
	for id, chatbotPubKey := range chatbotPubKeys {
		m.externalNodes[id] = Node{
			Public:     chatbotPubKey,
			SignPublic: chatbotSignPubKeys[id],
		}
		m.roots[id] = Node{}

		lastTreeKemRootSecret, err := ECKEMDecrypt(lastTreeKemRootCiphertexts[id], m.treekem.Self().Private)
		if err != nil {
			return err
		}
		lastTreeKemRootKey, err := NewKeyPairFromSecret(lastTreeKemRootSecret)
		lastTreeKemRootSignKey, err := NewSigningKeyPairFromSecret(lastTreeKemRootSecret)
		if err != nil {
			return err
		}
		m.lastTreeKemRoots[id] = Node{
			Public:      lastTreeKemRootKey.Public.Bytes(),
			Private:     lastTreeKemRootKey.Private.Bytes(),
			SignPublic:  lastTreeKemRootSignKey.Public.Bytes(),
			SignPrivate: lastTreeKemRootSignKey.Private.Bytes(),
			Secret:      lastTreeKemRootSecret,
		}
	}

	return nil
}

// UpdateTreeKEM issue a treekem user update and also return update messages for external nodes.
func (m *MultiTreeKEM) UpdateTreeKEM(externalNodeIds []string) (*UserUpdate, map[string]ECKEMCipherText, []byte, []byte, error) {
	newLeaf, err := GenerateRandomBytes(32)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	userUpdate := m.treekem.Update(newLeaf)
	m.treekem.HandleSelfUpdate(userUpdate, newLeaf)

	// Find root for TreeKEM
	rootNode := m.treekem.Nodes()[root(m.treekem.Size())]

	// Hash again for the multi-treekem
	h := hash(rootNode.Secret)
	encs := make(map[string]ECKEMCipherText)
	//for id, node := range m.externalNodes {
	for _, id := range externalNodeIds {
		node := m.externalNodes[id]

		enc, err := ECKEMEncrypt(h, node.Public)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		encs[id] = enc

		kp, err := NewKeyPairFromSecret(h)
		kpSign, err := NewSigningKeyPairFromSecret(h)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		m.roots[id] = Node{
			Secret:      h,
			Public:      kp.Public.Bytes(),
			Private:     kp.Private.Bytes(),
			SignPublic:  kpSign.Public.Bytes(),
			SignPrivate: kpSign.Private.Bytes(),
		}
		m.lastTreeKemRoots[id] = Node{
			Public:      m.treekem.Nodes()[root(m.treekem.Size())].Public,
			Private:     m.treekem.Nodes()[root(m.treekem.Size())].Private,
			SignPublic:  m.treekem.Nodes()[root(m.treekem.Size())].SignPublic,
			SignPrivate: m.treekem.Nodes()[root(m.treekem.Size())].SignPrivate,
			Secret:      m.treekem.Nodes()[root(m.treekem.Size())].Secret,
		}
	}

	return &userUpdate, encs, m.treekem.Nodes()[root(m.treekem.Size())].Public, m.treekem.Nodes()[root(m.treekem.Size())].SignPublic, nil
}

// HandleTreeKEMUpdate handles the UserUpdate request from other members.
func (m *MultiTreeKEM) HandleTreeKEMUpdate(userUpdate *UserUpdate, externalNodeIds []string) error {
	m.treekem.HandleUpdate(*userUpdate)

	// Update the roots
	h := hash(m.treekem.Nodes()[root(m.treekem.Size())].Secret)
	kp, err := NewKeyPairFromSecret(h)
	kpSign, err := NewSigningKeyPairFromSecret(h)
	if err != nil {
		return err
	}
	for _, externalNodeId := range externalNodeIds {
		m.roots[externalNodeId] = Node{
			Secret:      h,
			Public:      kp.Public.Bytes(),
			Private:     kp.Private.Bytes(),
			SignPublic:  kpSign.Public.Bytes(),
			SignPrivate: kpSign.Private.Bytes(),
		}
		m.lastTreeKemRoots[externalNodeId] = Node{
			Public:      m.treekem.Nodes()[root(m.treekem.Size())].Public,
			Private:     m.treekem.Nodes()[root(m.treekem.Size())].Private,
			SignPublic:  m.treekem.Nodes()[root(m.treekem.Size())].SignPublic,
			SignPrivate: m.treekem.Nodes()[root(m.treekem.Size())].SignPrivate,
			Secret:      m.treekem.Nodes()[root(m.treekem.Size())].Secret,
		}
	}

	return nil
}

// HandleExternalNodeUpdate handles the update message from the external node.
func (m *MultiTreeKEM) HandleExternalNodeUpdate(id string, updateMessage ECKEMCipherText, newPubKey []byte, newSignPubKey []byte) error {
	if _, ok := m.externalNodes[id]; !ok {
		return errors.New("id does not exist")
	}

	h, err := ECKEMDecrypt(updateMessage, m.lastTreeKemRoots[id].Private)
	if err != nil {
		return err
	}

	kp, err := NewKeyPairFromSecret(h)
	kpSign, err := NewSigningKeyPairFromSecret(h)
	if err != nil {
		panic(err)
	}
	m.roots[id] = Node{
		Secret:      h,
		Public:      kp.Public.Bytes(),
		Private:     kp.Private.Bytes(),
		SignPublic:  kpSign.Public.Bytes(),
		SignPrivate: kpSign.Private.Bytes(),
	}
	m.externalNodes[id] = Node{
		Public:     newPubKey,
		SignPublic: newSignPubKey,
	}

	return nil
}

// GetRootSecret returns the secret of the root node.
func (m *MultiTreeKEM) GetRootSecret(id string) []byte {
	return m.roots[id].Secret
}

// GetRootPublic returns the public key of the root node.
func (m *MultiTreeKEM) GetRootPublic(id string) []byte {
	return m.roots[id].Public
}

// GetRootSignPublic returns the public signing key of the root node.
func (m *MultiTreeKEM) GetRootSignPublic(id string) []byte {
	return m.roots[id].SignPublic
}

// GetRoots returns the roots.
func (m *MultiTreeKEM) GetRoots() map[string]Node {
	return m.roots
}

// GetTreeKEM returns the treekem.
func (m *MultiTreeKEM) GetTreeKEM() *TreeKEMState {
	return m.treekem
}

// GetExternalNode returns the external node.
func (m *MultiTreeKEM) GetExternalNode(id string) Node {
	return m.externalNodes[id]
}

// MultiTreeKEMExternal is a multi TreeKEM for the external node.
type MultiTreeKEMExternal struct {
	treekemRoot Node
	selfNode    Node
	root        Node
}

// NewMultiTreeKEMExternal creates a new MultiTreeKEMExternal object.
func NewMultiTreeKEMExternal(treekemRootPub, treekemRootSignPub, initLeaf []byte) *MultiTreeKEMExternal {
	treekemRoot := Node{Public: treekemRootPub, SignPublic: treekemRootSignPub}

	kpSelf, err := NewKeyPairFromSecret(initLeaf)
	kpSelfSign, err := NewSigningKeyPairFromSecret(initLeaf)
	if err != nil {
		panic(err)
	}
	selfNode := Node{
		Secret:      initLeaf,
		Public:      kpSelf.Public.Bytes(),
		Private:     kpSelf.Private.Bytes(),
		SignPublic:  kpSelfSign.Public.Bytes(),
		SignPrivate: kpSelfSign.Private.Bytes(),
	}

	rootSecret := hash(initLeaf)
	kpRoot, err := NewKeyPairFromSecret(rootSecret)
	kpRootSign, err := NewSigningKeyPairFromSecret(rootSecret)
	if err != nil {
		panic(err)
	}
	rootNode := Node{
		Secret:      rootSecret,
		Public:      kpRoot.Public.Bytes(),
		Private:     kpRoot.Private.Bytes(),
		SignPublic:  kpRootSign.Public.Bytes(),
		SignPrivate: kpRootSign.Private.Bytes(),
	}

	return &MultiTreeKEMExternal{
		treekemRoot: treekemRoot,
		selfNode:    selfNode,
		root:        rootNode,
	}
}

// UpdateExternalNode issue an external node update.
func (m *MultiTreeKEMExternal) UpdateExternalNode() (ECKEMCipherText, []byte, []byte, error) {
	newLeaf, err := GenerateRandomBytes(32)
	if err != nil {
		return ECKEMCipherText{}, nil, nil, err
	}

	kpSelf, err := NewKeyPairFromSecret(newLeaf)
	kpSelfSign, err := NewSigningKeyPairFromSecret(newLeaf)
	if err != nil {
		return ECKEMCipherText{}, nil, nil, err
	}
	m.selfNode = Node{
		Secret:      newLeaf,
		Public:      kpSelf.Public.Bytes(),
		Private:     kpSelf.Private.Bytes(),
		SignPublic:  kpSelfSign.Public.Bytes(),
		SignPrivate: kpSelfSign.Private.Bytes(),
	}

	kpRoot, err := NewKeyPairFromSecret(hash(newLeaf))
	kpRootSign, err := NewSigningKeyPairFromSecret(hash(newLeaf))
	if err != nil {
		return ECKEMCipherText{}, nil, nil, err
	}
	m.root = Node{
		Secret:      hash(newLeaf),
		Public:      kpRoot.Public.Bytes(),
		Private:     kpRoot.Private.Bytes(),
		SignPublic:  kpRootSign.Public.Bytes(),
		SignPrivate: kpRootSign.Private.Bytes(),
	}

	enc, err := ECKEMEncrypt(hash(newLeaf), m.treekemRoot.Public)
	if err != nil {
		return ECKEMCipherText{}, nil, nil, err
	}

	return enc, m.selfNode.Public, m.selfNode.SignPublic, nil
}

// HandleTreeKEMUpdate handles the treekem user update.
func (m *MultiTreeKEMExternal) HandleTreeKEMUpdate(updateMessage ECKEMCipherText, newPubKey []byte, newSignPubKey []byte) error {
	h, err := ECKEMDecrypt(updateMessage, m.selfNode.Private)
	if err != nil {
		return err
	}

	kp, err := NewKeyPairFromSecret(h)
	kpSign, err := NewSigningKeyPairFromSecret(h)
	if err != nil {
		return err
	}
	m.root = Node{
		Secret:      h,
		Public:      kp.Public.Bytes(),
		Private:     kp.Private.Bytes(),
		SignPublic:  kpSign.Public.Bytes(),
		SignPrivate: kpSign.Private.Bytes(),
	}
	m.treekemRoot = Node{Public: newPubKey, SignPublic: newSignPubKey}

	return nil
}

// GetRootSecret returns the secret of the root node.
func (m *MultiTreeKEMExternal) GetRootSecret() []byte {
	return m.root.Secret
}

// GetRootPublic returns the public key of the root node.
func (m *MultiTreeKEMExternal) GetRootPublic() []byte {
	return m.root.Public
}

// GetRootSignPublic returns the public signing key of the root node.
func (m *MultiTreeKEMExternal) GetRootSignPublic() []byte {
	return m.root.SignPublic
}

// GetSelfNode returns the self node.
func (m *MultiTreeKEMExternal) GetSelfNode() Node {
	return m.selfNode
}
