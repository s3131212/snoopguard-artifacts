package treekem

import (
	"errors"
	"github.com/s3131212/go-mls"
	"sync"
)

// MultiMlsTree is a struct that contains a MLS tree, the additional node, and waited-to-be-sent update message.
type MlsMultiTree struct {
	MlsState      **mls.State
	externalNodes map[string]Node
	roots         map[string]Node
	lastTreeRoots map[string]Node
	selfPubKey    []byte
	selfPrivKey   []byte
	mutexLock     sync.RWMutex
}

// NewMlsMultiTree creates a new MlsMultiTree object.
func NewMlsMultiTree(mlsState **mls.State) *MlsMultiTree {
	selfSecret := (*mlsState).GetSecrets().Keys.InitSecret
	initKp, _ := NewKeyPairFromSecret(selfSecret)

	return &MlsMultiTree{
		MlsState:      mlsState,
		externalNodes: make(map[string]Node),
		roots:         make(map[string]Node),
		lastTreeRoots: make(map[string]Node),
		selfPubKey:    initKp.Public.Bytes(),
		selfPrivKey:   initKp.Private.Bytes(),
	}

}

// AddExternalNode adds an external node to the multi-treekem.
func (m *MlsMultiTree) AddExternalNode(id string, ct ECKEMCipherText) error {
	if _, ok := m.externalNodes[id]; ok {
		return errors.New("id already exist")
	}

	currentRoot, err := m.GetTreeKEMRoot()
	if err != nil {
		return err
	}

	initLeaf, err := ECKEMDecrypt(ct, currentRoot.Private)
	if err != nil {
		return err
	}
	initKp, _ := NewKeyPairFromSecret(initLeaf)
	initKpSign, _ := NewSigningKeyPairFromSecret(initLeaf)

	m.mutexLock.Lock()
	m.externalNodes[id] = Node{
		Public:     initKp.Public.Bytes(),
		SignPublic: initKpSign.Public.Bytes(),
	}

	rootSecret := hash(initLeaf)
	rootKp, err := NewKeyPairFromSecret(rootSecret)
	rootKpSign, err := NewSigningKeyPairFromSecret(rootSecret)
	if err != nil {
		m.mutexLock.Unlock()
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
	m.lastTreeRoots[id] = currentRoot
	m.mutexLock.Unlock()

	return nil
}

// GetExternalNodeJoin generates the external node join message.
func (m *MlsMultiTree) GetExternalNodeJoin(id string) (ECKEMCipherText, []byte, error) {
	currentRoot, err := m.GetTreeKEMRoot()
	if err != nil {
		return ECKEMCipherText{}, nil, err
	}

	initLeaf, err := GenerateRandomBytes(32)
	if err != nil {
		return ECKEMCipherText{}, nil, err
	}

	initKp, err := NewKeyPairFromSecret(initLeaf)
	if err != nil {
		return ECKEMCipherText{}, nil, err
	}
	ct, err := ECKEMEncrypt(initLeaf, currentRoot.Public)
	if err != nil {
		return ECKEMCipherText{}, nil, err
	}

	rootSecret := hash(initLeaf)
	rootKp, err := NewKeyPairFromSecret(rootSecret)
	rootSignKp, err := NewSigningKeyPairFromSecret(rootSecret)
	if err != nil {
		return ECKEMCipherText{}, nil, err
	}

	m.mutexLock.Lock()
	m.externalNodes[id] = Node{Public: initKp.Public.Bytes()}
	m.roots[id] = Node{
		Secret:      rootSecret,
		Public:      rootKp.Public.Bytes(),
		Private:     rootKp.Private.Bytes(),
		SignPublic:  rootSignKp.Public.Bytes(),
		SignPrivate: rootSignKp.Private.Bytes(),
	}
	m.lastTreeRoots[id] = currentRoot
	m.mutexLock.Unlock()

	return ct, initLeaf, nil
}

// GetExternalNodeJoinsWithoutUpdate generates the external node join message for all external nodes, encrypted with pubKey, without updating any existing node.
func (m *MlsMultiTree) GetExternalNodeJoinsWithoutUpdate(pubKey []byte) (map[string][]byte, map[string][]byte, map[string]ECKEMCipherText, error) {
	chatbotPubKeys := make(map[string][]byte)
	chatbotSignPubKeys := make(map[string][]byte)
	lastTreeKemRootCiphertexts := make(map[string]ECKEMCipherText)
	m.mutexLock.RLock()
	for id, _ := range m.externalNodes {
		chatbotPubKeys[id] = m.externalNodes[id].Public
		chatbotSignPubKeys[id] = m.externalNodes[id].SignPublic

		ct, err := ECKEMEncrypt(m.lastTreeRoots[id].Secret, pubKey)
		if err != nil {
			return nil, nil, nil, err
		}
		lastTreeKemRootCiphertexts[id] = ct
	}
	m.mutexLock.RUnlock()

	return chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts, nil
}

// SetExternalNodeJoinsWithoutUpdate sets the external node join message for all external nodes without updating any existing node.
func (m *MlsMultiTree) SetExternalNodeJoinsWithoutUpdate(chatbotPubKeys map[string][]byte, chatbotSignPubKeys map[string][]byte, lastTreeKemRootCiphertexts map[string]ECKEMCipherText) error {
	m.mutexLock.Lock()
	for id, chatbotPubKey := range chatbotPubKeys {
		m.externalNodes[id] = Node{
			Public:     chatbotPubKey,
			SignPublic: chatbotSignPubKeys[id],
		}
		m.roots[id] = Node{}

		lastTreeKemRootSecret, err := ECKEMDecrypt(lastTreeKemRootCiphertexts[id], m.selfPrivKey)
		if err != nil {
			m.mutexLock.Unlock()
			return err
		}
		lastTreeKemRootKey, err := NewKeyPairFromSecret(lastTreeKemRootSecret)
		lastTreeKemRootSignKey, err := NewSigningKeyPairFromSecret(lastTreeKemRootSecret)
		if err != nil {
			m.mutexLock.Unlock()
			return err
		}
		m.lastTreeRoots[id] = Node{
			Public:      lastTreeKemRootKey.Public.Bytes(),
			Private:     lastTreeKemRootKey.Private.Bytes(),
			SignPublic:  lastTreeKemRootSignKey.Public.Bytes(),
			SignPrivate: lastTreeKemRootSignKey.Private.Bytes(),
			Secret:      lastTreeKemRootSecret,
		}
	}
	m.mutexLock.Unlock()

	return nil
}

// GetTreeKEMRoot calculates the treekem root from the hash.
func (m *MlsMultiTree) GetTreeKEMRoot() (Node, error) {
	rootHash := (*m.MlsState).Tree.RootHash()
	kp, err := NewKeyPairFromSecret(rootHash)
	kpSign, err := NewSigningKeyPairFromSecret(rootHash)
	if err != nil {
		return Node{}, err
	}
	return Node{
		Secret:      rootHash,
		Public:      kp.Public.Bytes(),
		Private:     kp.Private.Bytes(),
		SignPublic:  kpSign.Public.Bytes(),
		SignPrivate: kpSign.Private.Bytes(),
	}, nil
}

// UpdateTreeKEM issue a treekem user update and also return update messages for external nodes.
func (m *MlsMultiTree) UpdateTreeKEM(externalNodeIds []string) (map[string]ECKEMCipherText, []byte, []byte, error) {
	// Get current root
	currentRoot, err := m.GetTreeKEMRoot()
	if err != nil {
		return nil, nil, nil, err
	}

	// Hash again for the multi-treekem
	h := hash(currentRoot.Secret)
	encs := make(map[string]ECKEMCipherText)
	m.mutexLock.Lock()
	//for id, node := range m.externalNodes {
	for _, id := range externalNodeIds {
		node, exist := m.externalNodes[id]
		if !exist {
			continue
		}

		enc, err := ECKEMEncrypt(h, node.Public)
		if err != nil {
			m.mutexLock.Unlock()
			return nil, nil, nil, err
		}
		encs[id] = enc

		kp, err := NewKeyPairFromSecret(h)
		kpSign, err := NewSigningKeyPairFromSecret(h)
		if err != nil {
			m.mutexLock.Unlock()
			return nil, nil, nil, err
		}
		m.roots[id] = Node{
			Secret:      h,
			Public:      kp.Public.Bytes(),
			Private:     kp.Private.Bytes(),
			SignPublic:  kpSign.Public.Bytes(),
			SignPrivate: kpSign.Private.Bytes(),
		}
		m.lastTreeRoots[id] = currentRoot
	}
	m.mutexLock.Unlock()

	return encs, currentRoot.Public, currentRoot.SignPublic, nil
}

// HandleTreeKEMUpdate handles the update request from other members. This should be called after the MLS tree is updated
func (m *MlsMultiTree) HandleTreeKEMUpdate(externalNodeIds []string) error {
	currentRoot, err := m.GetTreeKEMRoot()
	if err != nil {
		return err
	}

	// Update the roots
	h := hash(currentRoot.Secret)
	kp, err := NewKeyPairFromSecret(h)
	kpSign, err := NewSigningKeyPairFromSecret(h)
	if err != nil {
		return err
	}
	m.mutexLock.Lock()
	for _, externalNodeId := range externalNodeIds {
		m.roots[externalNodeId] = Node{
			Secret:      h,
			Public:      kp.Public.Bytes(),
			Private:     kp.Private.Bytes(),
			SignPublic:  kpSign.Public.Bytes(),
			SignPrivate: kpSign.Private.Bytes(),
		}
		m.lastTreeRoots[externalNodeId] = currentRoot
	}
	m.mutexLock.Unlock()

	return nil
}

// HandleExternalNodeUpdate handles the update message from the external node.
func (m *MlsMultiTree) HandleExternalNodeUpdate(id string, updateMessage ECKEMCipherText, newPubKey []byte, newSignPubKey []byte) error {
	if _, ok := m.externalNodes[id]; !ok {
		return errors.New("id does not exist")
	}

	h, err := ECKEMDecrypt(updateMessage, m.lastTreeRoots[id].Private)
	if err != nil {
		return err
	}

	kp, err := NewKeyPairFromSecret(h)
	kpSign, err := NewSigningKeyPairFromSecret(h)
	if err != nil {
		panic(err)
	}
	m.mutexLock.Lock()
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
	m.mutexLock.Unlock()

	return nil
}

// GetRootSecret returns the secret of the root node.
func (m *MlsMultiTree) GetRootSecret(id string) []byte {
	m.mutexLock.RLock()
	defer m.mutexLock.RUnlock()
	return m.roots[id].Secret
}

// GetRootPublic returns the public key of the root node.
func (m *MlsMultiTree) GetRootPublic(id string) []byte {
	m.mutexLock.RLock()
	defer m.mutexLock.RUnlock()
	return m.roots[id].Public
}

// GetRootSignPublic returns the public signing key of the root node.
func (m *MlsMultiTree) GetRootSignPublic(id string) []byte {
	m.mutexLock.RLock()
	defer m.mutexLock.RUnlock()
	return m.roots[id].SignPublic
}

// GetRoots returns the roots.
func (m *MlsMultiTree) GetRoots() map[string]Node {
	return m.roots
}

// GetExternalNode returns the external node.
func (m *MlsMultiTree) GetExternalNode(id string) Node {
	m.mutexLock.RLock()
	defer m.mutexLock.RUnlock()
	return m.externalNodes[id]
}

// MlsMultiTreeExternal is a multi TreeKEM for the external node.
type MlsMultiTreeExternal struct {
	treekemRoot Node
	selfNode    Node
	root        Node
}

// NewMlsMultiTreeExternal creates a new MlsMultiTreeExternal object.
func NewMlsMultiTreeExternal(treekemRootPub, treekemRootSignPub, initLeaf []byte) *MlsMultiTreeExternal {
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

	return &MlsMultiTreeExternal{
		treekemRoot: treekemRoot,
		selfNode:    selfNode,
		root:        rootNode,
	}
}

// UpdateExternalNode issue an external node update.
func (m *MlsMultiTreeExternal) UpdateExternalNode() (ECKEMCipherText, []byte, []byte, error) {
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
func (m *MlsMultiTreeExternal) HandleTreeKEMUpdate(updateMessage ECKEMCipherText, newPubKey []byte, newSignPubKey []byte) error {
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

// UpdateRootFromHash calculates the treekem root from the hash.
func (m *MlsMultiTreeExternal) UpdateRootFromHash(hash []byte) error {
	kp, err := NewKeyPairFromSecret(hash)
	kpSign, err := NewSigningKeyPairFromSecret(hash)
	if err != nil {
		return err
	}
	m.treekemRoot = Node{
		Secret:      hash,
		Public:      kp.Public.Bytes(),
		Private:     kp.Private.Bytes(),
		SignPublic:  kpSign.Public.Bytes(),
		SignPrivate: kpSign.Private.Bytes(),
	}
	return nil
}

// GetRootSecret returns the secret of the root node.
func (m *MlsMultiTreeExternal) GetRootSecret() []byte {
	return m.root.Secret
}

// GetRootPublic returns the public key of the root node.
func (m *MlsMultiTreeExternal) GetRootPublic() []byte {
	return m.root.Public
}

// GetRootSignPublic returns the public signing key of the root node.
func (m *MlsMultiTreeExternal) GetRootSignPublic() []byte {
	return m.root.SignPublic
}

// GetSelfNode returns the self node.
func (m *MlsMultiTreeExternal) GetSelfNode() Node {
	return m.selfNode
}
