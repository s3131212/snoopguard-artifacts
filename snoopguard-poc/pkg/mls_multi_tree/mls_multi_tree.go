package mls_multi_tree

import (
	"fmt"
	"github.com/s3131212/go-mls"
)

/*
Description:
We introduce how CMRT works. Each chatbot (external entity)
has its dedicated chatbot subtree and a root node
shared with the group members placed in the user subtree.
From the user’s point of view, the top of the group member’s
tree is connected to several root nodes, each of which has a
separate chatbot underneath it. From the chatbot’s point of
view, it is in a 3-node tree with only two leaf nodes, itself on
the right and the root node of the group members’ subtree on
the left. However, the chatbot is unaware of group members’
tree except the root. When a key is updated, either by the user
or a chatbot, the user stores the root node of the group mem-
ber’s subtree and the root node dedicated to the chatbot. This
process ensures that group members always have a shared
secret to this chatbot and are capable of updating keys in the
future without storing the complete tree structure. The storage
requirement is reduced from a complete tree structure for each
chatbot to two nodes for each and only a single version of the
group member’s subtree. Such an improvement results in a
“multi-root” tree structure that can be “compressed” to save
storage.
*/

type MlsNode struct {
	Secret      []byte
	Public      []byte
	Private     []byte
	SignPublic  []byte
	SignPrivate []byte
}

// MlsMultiTree is a struct that contains an MLS state and TreeKEMs.
type MlsMultiTree struct {
	mlsState          *mls.State
	TreeKEMPrivateKey map[string]*mls.TreeKEMPrivateKey
	TreeKEMPublicKey  map[string]*mls.TreeKEMPublicKey
}

// NewMlsMultiTree creates a new MlsMultiTree object.
func NewMlsMultiTree(mlsState *mls.State) *MlsMultiTree {
	return &MlsMultiTree{
		mlsState:          mlsState,
		TreeKEMPrivateKey: make(map[string]*mls.TreeKEMPrivateKey),
		TreeKEMPublicKey:  make(map[string]*mls.TreeKEMPublicKey),
	}
}

// AddExternalTreeKEM creates a new TreeKEM object to an MlsMultiTree object.
func (m *MlsMultiTree) AddExternalTreeKEM(externalID string, secret []byte, sigPrivate mls.SignaturePrivateKey, keyPackage *mls.KeyPackage) error {
	if _, exist := m.TreeKEMPrivateKey[externalID]; exist {
		return fmt.Errorf("TreeKEM already exists for external ID %s", externalID)
	}

	suite := m.mlsState.CipherSuite
	m.TreeKEMPublicKey[externalID] = mls.NewTreeKEMPublicKey(suite)
	index := m.TreeKEMPublicKey[externalID].AddLeaf(*keyPackage)
	m.TreeKEMPrivateKey[externalID] = mls.NewTreeKEMPrivateKey(suite, m.TreeKEMPublicKey[externalID].Size(), index, secret)
	//sigPriv := sigPrivate

	return nil
}

// UpdateExternalTreeKEM updates the TreeKEM object in an MlsMultiTree object.
func (m *MlsMultiTree) UpdateExternalTreeKEM(externalID string, secret []byte, sigPrivate mls.SignaturePrivateKey, keyPackage *mls.KeyPackage) error {

}
