package treekem

import (
	"chatbot-poc-go/pkg/util"
	"fmt"
	"github.com/s3131212/go-mls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type StateTest struct {
	initSecrets   [][]byte
	identityPrivs []mls.SignaturePrivateKey
	credentials   []mls.Credential
	initPrivs     []mls.HPKEPrivateKey
	keyPackages   []mls.KeyPackage
	states        []mls.State
}

func TestMlsMultiTree(t *testing.T) {
	groupSize := 10
	stateTest := setup(t, groupSize)
	// start with the group creator
	s0, err := mls.NewEmptyState([]byte("test"), stateTest.initSecrets[0], stateTest.identityPrivs[0], stateTest.keyPackages[0])
	require.Nil(t, err)
	stateTest.states = append(stateTest.states, *s0)

	// add proposals for rest of the participants
	for i := 1; i < groupSize; i++ {
		add, err := stateTest.states[0].Add(stateTest.keyPackages[i])
		require.Nil(t, err)
		_, err = stateTest.states[0].Handle(add)
		require.Nil(t, err)
	}

	// commit the adds
	secret := util.RandomBytes(32)
	_, welcome, next, err := stateTest.states[0].Commit(secret)
	require.Nil(t, err)
	stateTest.states[0] = *next
	// initialize the new joiners from the welcome
	for i := 1; i < groupSize; i++ {
		s, err := mls.NewJoinedState(stateTest.initSecrets[i], stateTest.identityPrivs[i:i+1], stateTest.keyPackages[i:i+1], *welcome)
		require.Nil(t, err)
		stateTest.states = append(stateTest.states, *s)
	}

	// Verify that the states are all equivalent
	for _, lhs := range stateTest.states {
		for _, rhs := range stateTest.states {
			require.True(t, lhs.Equals(rhs))
		}
	}

	// Set up a MlsMultiTree
	mlsMultiTrees := make([]*MlsMultiTree, groupSize)

	for i := 0; i < groupSize; i++ {
		mlsMultiTrees[i] = NewMlsMultiTree(&stateTest.states[i])
	}

	// Create chatbots
	chatbots := make([]*MlsMultiTreeExternal, 3)
	for i := 0; i < 3; i++ {
		cbct, initLeaf, err := mlsMultiTrees[i].GetExternalNodeJoin(fmt.Sprintf("cb-%d", i))
		assert.Nilf(t, err, "error creating chatbot add: %s", err)

		// Generate treekem root keys
		treekemRootSecret := stateTest.states[i].Tree.RootHash()
		treekemRootKp, _ := NewKeyPairFromSecret(treekemRootSecret)
		treekemRootKpSign, _ := NewSigningKeyPairFromSecret(treekemRootSecret)

		chatbots[i] = NewMlsMultiTreeExternal(treekemRootKp.Public.Bytes(), treekemRootKpSign.Public.Bytes(), initLeaf)

		// Have each member handle the chatbot add with id "cb" + i
		for j, mt := range mlsMultiTrees {
			if i == j {
				continue
			}
			err := mt.AddExternalNode(fmt.Sprintf("cb-%d", i), cbct)
			assert.Nilf(t, err, "error handling chatbot add: %s", err)

			// Verify that the chatbot is consistent with the member
			assert.Equal(t, chatbots[i].GetRootSecret(), mt.GetRootSecret(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member")
			assert.Equal(t, chatbots[i].GetRootPublic(), mt.GetRootPublic(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member")
			assert.Equal(t, chatbots[i].GetRootSignPublic(), mt.GetRootSignPublic(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member")

		}
	}

	// Update chatbots' keys.
	for i, chatbot := range chatbots {
		// Chatbot issue a key update
		chatbotUpdate, newCbPubKey, newCbSignPubKey, err := chatbot.UpdateExternalNode()
		assert.Nilf(t, err, "error creating chatbot update: %s", err)

		// Have each member handle the chatbot update
		for j, mt := range mlsMultiTrees {
			err = mt.HandleExternalNodeUpdate(fmt.Sprintf("cb-%d", i), chatbotUpdate, newCbPubKey, newCbSignPubKey)
			assert.Nilf(t, err, "error handling chatbot update: %s", err)

			// Verify that the chatbot is consistent with the member
			assert.Equal(t, chatbot.GetRootSecret(), mt.GetRootSecret(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbot.GetRootPublic(), mt.GetRootPublic(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbot.GetRootSignPublic(), mt.GetRootSignPublic(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member %d", j)
		}
	}

	// Have each member update and verify that others are consistent
	for c, chatbot := range chatbots {
		for i, mt1 := range mlsMultiTrees {
			ct, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err := mt1.UpdateTreeKEM([]string{fmt.Sprintf("cb-%d", c)}, stateTest.states[i].Tree.RootHash())
			assert.Nilf(t, err, "error creating user update: %s", err)

			err = chatbot.HandleTreeKEMUpdate(ct[fmt.Sprintf("cb-%d", c)], newTreeKemRootPubKey, newTreeKemRootSignPubKey)
			assert.Nilf(t, err, "error updating chatbot: %s", err)
			assert.Equal(t, chatbot.GetRootSecret(), mt1.GetRootSecret(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with the updating member")
			assert.Equal(t, chatbot.GetRootPublic(), mt1.GetRootPublic(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with the updating member")
			assert.Equal(t, chatbot.GetRootSignPublic(), mt1.GetRootSignPublic(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with the updating member")

			for j, mt2 := range mlsMultiTrees {
				if i == j {
					continue
				}
				err = mt2.HandleTreeKEMUpdate([]string{fmt.Sprintf("cb-%d", c)})
				assert.Nilf(t, err, "error handling user update: %s", err)

				// assert state equal between members
				assert.Equal(t, mt1.GetRootSecret(fmt.Sprintf("cb-%d", c)), mt2.GetRootSecret(fmt.Sprintf("cb-%d", c)), "states are not equal")

				// Verify that the chatbot is consistent with the member
				assert.Equal(t, chatbot.GetRootSecret(), mt2.GetRootSecret(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with member %d", j)
				assert.Equal(t, chatbot.GetRootPublic(), mt2.GetRootPublic(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with member %d", j)
				assert.Equal(t, chatbot.GetRootSignPublic(), mt2.GetRootSignPublic(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with member %d", j)
			}

			// Update chatbot key again
			chatbotUpdate, newCbPubKey, newCbSignPubKey, err := chatbot.UpdateExternalNode()
			assert.Nilf(t, err, "error creating chatbot update: %s", err)

			// Have each member handle the chatbot update
			for j, mt := range mlsMultiTrees {
				err = mt.HandleExternalNodeUpdate(fmt.Sprintf("cb-%d", c), chatbotUpdate, newCbPubKey, newCbSignPubKey)
				assert.Nilf(t, err, "error handling chatbot update: %s", err)

				// Verify that the chatbot is consistent with the member
				assert.Equal(t, chatbot.GetRootSecret(), mt.GetRootSecret(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with member %d", j)
				assert.Equal(t, chatbot.GetRootPublic(), mt.GetRootPublic(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with member %d", j)
				assert.Equal(t, chatbot.GetRootSignPublic(), mt.GetRootSignPublic(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with member %d", j)
			}
		}
	}

	// New StateTest
	groupSize2 := 3
	stateTest2 := setup(t, groupSize2)

	// Add proposal for the three new members
	for i := 0; i < groupSize2; i++ {
		add, err := stateTest.states[1].Add(stateTest2.keyPackages[i])
		require.Nil(t, err)
		_, err = stateTest.states[1].Handle(add)
		require.Nil(t, err)

		// original members handle the add
		for j := 0; j < groupSize; j++ {
			_, err = stateTest.states[j].Handle(add)
			require.Nil(t, err)
		}
	}

	// commit the adds
	secret = util.RandomBytes(32)
	addCommit, welcome, next, err := stateTest.states[1].Commit(secret)
	require.Nil(t, err)
	stateTest.states[1] = *next

	// original members handle the add commit
	for i := 0; i < groupSize; i++ {
		if i == 1 { // states[1] issues the add commit, so skip
			continue
		}
		next, err = stateTest.states[i].Handle(addCommit)
		require.Nil(t, err)
		stateTest.states[i] = *next
	}

	// initialize the new joiners from the welcome
	for i := 0; i < groupSize2; i++ {
		s, err := mls.NewJoinedState(stateTest2.initSecrets[i], stateTest2.identityPrivs[i:i+1], stateTest2.keyPackages[i:i+1], *welcome)
		require.Nil(t, err)
		stateTest2.states = append(stateTest2.states, *s)
	}

	// Verify that the states are all equivalent
	for _, lhs := range stateTest.states {
		for _, rhs := range stateTest.states {
			require.True(t, lhs.Equals(rhs))
		}
		for _, rhs := range stateTest2.states {
			require.True(t, lhs.Equals(rhs))
		}
	}

	// Set up another MlsMultiTree
	mlsMultiTrees2 := make([]*MlsMultiTree, groupSize2)
	for i := 0; i < groupSize2; i++ {
		mlsMultiTrees2[i] = NewMlsMultiTree(stateTest2.states[i])
	}

	// member 1 should also send the info of chatbots to the new joiners
	for _, mt2 := range mlsMultiTrees2 {
		chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts, err := mlsMultiTrees[1].GetExternalNodeJoinsWithoutUpdate(mt2.selfPubKey)
		assert.Nil(t, err)
		err = mt2.SetExternalNodeJoinsWithoutUpdate(chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts)
		assert.Nil(t, err)
	}

	// Verify that all MlsMultiTrees are consistent
	for i, mt1 := range mlsMultiTrees {
		// Verify that the chatbot is consistent with the member
		assert.Equal(t, chatbots[0].GetRootSecret(), mt1.GetRootSecret("cb-0"), "chatbot 0 is not consistent with member %d", i)
		assert.Equal(t, chatbots[0].GetRootPublic(), mt1.GetRootPublic("cb-0"), "chatbot 0 is not consistent with member %d", i)
		assert.Equal(t, chatbots[0].GetRootSignPublic(), mt1.GetRootSignPublic("cb-0"), "chatbot 0 is not consistent with member %d", i)
	}

	// Chatbot 2 issue a key update
	chatbotUpdate, newCbPubKey, newCbSignPubKey, err := chatbots[2].UpdateExternalNode()
	assert.Nilf(t, err, "error creating chatbot update: %s", err)

	// Have each member handle the chatbot update
	for j, mt := range mlsMultiTrees {
		err = mt.HandleExternalNodeUpdate("cb-2", chatbotUpdate, newCbPubKey, newCbSignPubKey)
		assert.Nilf(t, err, "error handling chatbot update: %s", err)

		// Verify that the chatbot is consistent with the member
		assert.Equal(t, chatbots[2].GetRootSecret(), mt.GetRootSecret("cb-2"), "chatbot is not consistent with member %d", j)
		assert.Equal(t, chatbots[2].GetRootPublic(), mt.GetRootPublic("cb-2"), "chatbot is not consistent with member %d", j)
		assert.Equal(t, chatbots[2].GetRootSignPublic(), mt.GetRootSignPublic("cb-2"), "chatbot is not consistent with member %d", j)
	}
	for j, mt2 := range mlsMultiTrees2 {
		err = mt2.HandleExternalNodeUpdate("cb-2", chatbotUpdate, newCbPubKey, newCbSignPubKey)
		assert.Nilf(t, err, "error handling chatbot update: %s", err)

		// Verify that the chatbot is consistent with the member
		assert.Equal(t, chatbots[2].GetRootSecret(), mt2.GetRootSecret("cb-2"), "chatbot is not consistent with member %d", j)
		assert.Equal(t, chatbots[2].GetRootPublic(), mt2.GetRootPublic("cb-2"), "chatbot is not consistent with member %d", j)
		assert.Equal(t, chatbots[2].GetRootSignPublic(), mt2.GetRootSignPublic("cb-2"), "chatbot is not consistent with member %d", j)
	}
}

func setup(t *testing.T, groupSize int) StateTest {
	stateTest := StateTest{}
	stateTest.keyPackages = make([]mls.KeyPackage, groupSize)
	scheme := mls.P256_AES128GCM_SHA256_P256.Scheme()

	for i := 0; i < groupSize; i++ {
		// cred gen
		secret := util.RandomBytes(32)
		sigPriv, err := scheme.Derive(secret)
		require.Nil(t, err)

		cred := mls.NewBasicCredential([]byte("test"), scheme, sigPriv.PublicKey)

		//kp gen
		kp, err := mls.NewKeyPackageWithSecret(mls.P256_AES128GCM_SHA256_P256, secret, cred, sigPriv)
		require.Nil(t, err)

		// save all the materials
		stateTest.initSecrets = append(stateTest.initSecrets, secret)
		stateTest.identityPrivs = append(stateTest.identityPrivs, sigPriv)
		stateTest.credentials = append(stateTest.credentials, *cred)
		stateTest.keyPackages[i] = *kp
	}
	return stateTest
}
