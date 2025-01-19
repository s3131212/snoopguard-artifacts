package treekem

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMultiTreeKEM(t *testing.T) {
	for testGroupSize := 3; testGroupSize <= 3; testGroupSize++ {
		leaf, _ := generateRandomBytes(32)

		creator := TreeKEMStateOneMemberGroup(leaf)
		members := []*TreeKEMState{creator}

		for i := 1; i < testGroupSize; i++ {
			leaf, _ = generateRandomBytes(32)
			initKP, _ := NewKeyPairFromSecret(leaf)
			gaGroup, gaJoiner, _ := members[len(members)-1].Add(initKP.Public.Bytes())
			joiner, _ := TreeKEMStateFromGroupAdd(leaf, gaJoiner)
			for _, m := range members {
				m.HandleGroupAdd(gaGroup)
			}

			members = append(members, joiner)
		}

		// Create a multi-treekem for each member
		multiTreeKEMs := make([]*MultiTreeKEM, len(members))
		for i, m := range members {
			multiTreeKEMs[i] = NewMultiTreeKEM(m)
		}

		// Create chatbots
		chatbots := make([]*MultiTreeKEMExternal, 3)
		for i := 0; i < 3; i++ {
			cbct, initLeaf, err := multiTreeKEMs[i].GetExternalNodeJoin(fmt.Sprintf("cb-%d", i))
			assert.Nilf(t, err, "error creating chatbot add: %s", err)
			chatbots[i] = NewMultiTreeKEMExternal(members[i].Nodes()[root(members[i].Size())].Public, members[i].Nodes()[root(members[i].Size())].SignPublic, initLeaf)

			// Have each member handle the chatbot add with id "cb" + i
			for j, mt := range multiTreeKEMs {
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
			for j, mt := range multiTreeKEMs {
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
			for i, mt1 := range multiTreeKEMs {
				userUpdate, ct, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err := mt1.UpdateTreeKEM([]string{fmt.Sprintf("cb-%d", c)})
				assert.Nilf(t, err, "error creating user update: %s", err)

				err = chatbot.HandleTreeKEMUpdate(ct[fmt.Sprintf("cb-%d", c)], newTreeKemRootPubKey, newTreeKemRootSignPubKey)
				assert.Nilf(t, err, "error updating chatbot: %s", err)
				assert.Equal(t, chatbot.GetRootSecret(), mt1.GetRootSecret(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with the updating member")
				assert.Equal(t, chatbot.GetRootPublic(), mt1.GetRootPublic(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with the updating member")
				assert.Equal(t, chatbot.GetRootSignPublic(), mt1.GetRootSignPublic(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with the updating member")

				for j, mt2 := range multiTreeKEMs {
					if members[j].Index() == members[i].Index() {
						continue
					}
					err = mt2.HandleTreeKEMUpdate(userUpdate, []string{fmt.Sprintf("cb-%d", c)})
					assert.Nilf(t, err, "error handling user update: %s", err)

					eq := groupEqual(members[i], members[j])
					assert.Truef(t, eq, "members %d -> %d are not equal", members[i].Index(), members[j].Index())

					// Verify that the chatbot is consistent with the member
					assert.Equal(t, chatbot.GetRootSecret(), mt2.GetRootSecret(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with member %d", j)
					assert.Equal(t, chatbot.GetRootPublic(), mt2.GetRootPublic(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with member %d", j)
					assert.Equal(t, chatbot.GetRootSignPublic(), mt2.GetRootSignPublic(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with member %d", j)
				}

				// Update chatbot key again
				chatbotUpdate, newCbPubKey, newCbSignPubKey, err := chatbot.UpdateExternalNode()
				assert.Nilf(t, err, "error creating chatbot update: %s", err)

				// Have each member handle the chatbot update
				for j, mt := range multiTreeKEMs {
					err = mt.HandleExternalNodeUpdate(fmt.Sprintf("cb-%d", c), chatbotUpdate, newCbPubKey, newCbSignPubKey)
					assert.Nilf(t, err, "error handling chatbot update: %s", err)

					// Verify that the chatbot is consistent with the member
					assert.Equal(t, chatbot.GetRootSecret(), mt.GetRootSecret(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with member %d", j)
					assert.Equal(t, chatbot.GetRootPublic(), mt.GetRootPublic(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with member %d", j)
					assert.Equal(t, chatbot.GetRootSignPublic(), mt.GetRootSignPublic(fmt.Sprintf("cb-%d", c)), "chatbot is not consistent with member %d", j)
				}
			}
		}

		// Add a new member to the group
		leaf, _ = generateRandomBytes(32)
		initKP, _ := NewKeyPairFromSecret(leaf)
		gaGroup, gaJoiner, _ := members[0].Add(initKP.Public.Bytes())
		finalJoiner, _ := TreeKEMStateFromGroupAdd(leaf, gaJoiner)
		finalJoinerMT := NewMultiTreeKEM(finalJoiner)
		for _, m := range members {
			m.HandleGroupAdd(gaGroup)
		}
		members = append(members, finalJoiner)
		multiTreeKEMs = append(multiTreeKEMs, finalJoinerMT)

		// The final joiner build the external nodes for the three chatbots using the information given by the first user
		for i, chatbot := range chatbots {
			// member 0 send the new root to chatbot
			ctToChatbot, err := ECKEMEncrypt(hash(members[0].Nodes()[root(members[0].Size())].Secret), multiTreeKEMs[0].externalNodes[fmt.Sprintf("cb-%d", i)].Public)
			assert.Nil(t, err)

			// chatbot receive the new root
			cbNewRootSecret, err := ECKEMDecrypt(ctToChatbot, chatbot.selfNode.Private)

			assert.Equal(t, cbNewRootSecret, hash(members[0].Nodes()[root(members[0].Size())].Secret), "chatbot fails to decrypt new secret")

			kp, err := NewKeyPairFromSecret(cbNewRootSecret)
			chatbot.root = Node{
				Secret:  cbNewRootSecret,
				Public:  kp.Public.Bytes(),
				Private: kp.Private.Bytes(),
			}
			chatbot.treekemRoot = Node{Public: multiTreeKEMs[0].treekem.Nodes()[root(multiTreeKEMs[0].treekem.Size())].Public}

			// Verify that the chatbot is consistent with the member
			assert.Equal(t, chatbot.treekemRoot.Public, multiTreeKEMs[0].treekem.RootPublic(), "chatbot is not consistent with member 0")
		}
	}
}

func TestMultiTreeKEMAsyncUpdate(t *testing.T) {
	for testGroupSize := 3; testGroupSize <= 32; testGroupSize++ {
		leaf, _ := generateRandomBytes(32)

		creator := TreeKEMStateOneMemberGroup(leaf)
		members := []*TreeKEMState{creator}

		for i := 1; i < testGroupSize; i++ {
			leaf, _ = generateRandomBytes(32)
			initKP, _ := NewKeyPairFromSecret(leaf)
			gaGroup, gaJoiner, _ := members[len(members)-1].Add(initKP.Public.Bytes())
			joiner, _ := TreeKEMStateFromGroupAdd(leaf, gaJoiner)
			for _, m := range members {
				m.HandleGroupAdd(gaGroup)
			}

			members = append(members, joiner)
		}

		// Create a multi-treekem for each member
		multiTreeKEMs := make([]*MultiTreeKEM, len(members))
		for i, m := range members {
			multiTreeKEMs[i] = NewMultiTreeKEM(m)
		}

		// Create chatbots
		chatbots := make([]*MultiTreeKEMExternal, 3)
		for i := 0; i < 3; i++ {
			cbct, initLeaf, err := multiTreeKEMs[i].GetExternalNodeJoin(fmt.Sprintf("cb-%d", i))
			assert.Nilf(t, err, "error creating chatbot add: %s", err)
			chatbots[i] = NewMultiTreeKEMExternal(members[i].Nodes()[root(members[i].Size())].Public, members[i].Nodes()[root(members[i].Size())].SignPublic, initLeaf)

			// Have each member handle the chatbot add with id "cb" + i
			for j, mt := range multiTreeKEMs {
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
			for j, mt := range multiTreeKEMs {
				err = mt.HandleExternalNodeUpdate(fmt.Sprintf("cb-%d", i), chatbotUpdate, newCbPubKey, newCbSignPubKey)
				assert.Nilf(t, err, "error handling chatbot update: %s", err)

				// Verify that the chatbot is consistent with the member
				assert.Equal(t, chatbot.GetRootSecret(), mt.GetRootSecret(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member %d", j)
				assert.Equal(t, chatbot.GetRootPublic(), mt.GetRootPublic(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member %d", j)
				assert.Equal(t, chatbot.GetRootSignPublic(), mt.GetRootSignPublic(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member %d", j)
			}
		}

		// Cache chatbot updates
		ctCache, newCbPubKeyCache, newCbSignPubKeyCache := make(map[string]ECKEMCipherText, len(chatbots)), make([]byte, len(chatbots)), make([]byte, len(chatbots))

		// Each member update their key but do not notify the chatbot
		for i, mt1 := range multiTreeKEMs {
			userUpdate, ct, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err := mt1.UpdateTreeKEM([]string{"cb-0", "cb-1", "cb-2"})
			assert.Nilf(t, err, "error creating user update: %s", err)

			ctCache = ct
			newCbPubKeyCache = newTreeKemRootPubKey
			newCbSignPubKeyCache = newTreeKemRootSignPubKey

			for j, mt2 := range multiTreeKEMs {
				if members[j].Index() == members[i].Index() {
					continue
				}
				err = mt2.HandleTreeKEMUpdate(userUpdate, []string{"cb-0", "cb-1", "cb-2"})
				assert.Nilf(t, err, "error handling user update: %s", err)

				eq := groupEqual(members[i], members[j])
				assert.Truef(t, eq, "members %d -> %d are not equal", members[i].Index(), members[j].Index())
			}
		}

		// Each chatbot applies the update
		for i, chatbot := range chatbots {
			err := chatbot.HandleTreeKEMUpdate(ctCache[fmt.Sprintf("cb-%d", i)], newCbPubKeyCache, newCbSignPubKeyCache)
			assert.Nilf(t, err, "error updating chatbot: %s", err)
			// Verify with each member
			for j, mt := range multiTreeKEMs {
				assert.Equal(t, chatbot.GetRootSecret(), mt.GetRootSecret(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member %d", j)
				assert.Equal(t, chatbot.GetRootPublic(), mt.GetRootPublic(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member %d", j)
				assert.Equal(t, chatbot.GetRootSignPublic(), mt.GetRootSignPublic(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member %d", j)
			}
		}

		// Update each chatbot's key and update users' trees simultaneously
		chatbotUpdateCache, newCbPubKeyCache_, newCbSignPubKeyCache_ := make(map[string]ECKEMCipherText, len(chatbots)), make(map[string][]byte, len(chatbots)), make(map[string][]byte, len(chatbots))
		for i, chatbot := range chatbots {
			// Chatbot issue a key update
			chatbotUpdate, newCbPubKey, newCbSignPubKey, err := chatbot.UpdateExternalNode()
			assert.Nilf(t, err, "error creating chatbot update: %s", err)

			chatbotUpdateCache[fmt.Sprintf("cb-%d", i)] = chatbotUpdate
			newCbPubKeyCache_[fmt.Sprintf("cb-%d", i)] = newCbPubKey
			newCbSignPubKeyCache_[fmt.Sprintf("cb-%d", i)] = newCbSignPubKey
		}

		// Have each member handle the chatbot update
		for j, mt := range multiTreeKEMs {
			for i, chatbot := range chatbots {
				err := mt.HandleExternalNodeUpdate(fmt.Sprintf("cb-%d", i), chatbotUpdateCache[fmt.Sprintf("cb-%d", i)], newCbPubKeyCache_[fmt.Sprintf("cb-%d", i)], newCbSignPubKeyCache_[fmt.Sprintf("cb-%d", i)])
				assert.Nilf(t, err, "error handling chatbot update: %s", err)

				// Verify that the chatbot is consistent with the member
				assert.Equal(t, chatbot.GetRootSecret(), mt.GetRootSecret(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member %d", j)
				assert.Equal(t, chatbot.GetRootPublic(), mt.GetRootPublic(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member %d", j)
				assert.Equal(t, chatbot.GetRootSignPublic(), mt.GetRootSignPublic(fmt.Sprintf("cb-%d", i)), "chatbot is not consistent with member %d", j)
			}
		}

		// The first chatbot update key and update users' trees simultaneously.
		chatbotUpdate, newCbPubKey, newCbSignPubKey, err := chatbots[0].UpdateExternalNode()
		assert.Nilf(t, err, "error creating chatbot update: %s", err)
		for j, mt := range multiTreeKEMs {
			err := mt.HandleExternalNodeUpdate("cb-0", chatbotUpdate, newCbPubKey, newCbSignPubKey)
			assert.Nilf(t, err, "error handling chatbot update: %s", err)
			assert.Equal(t, chatbots[0].GetRootSecret(), mt.GetRootSecret("cb-0"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[0].GetRootPublic(), mt.GetRootPublic("cb-0"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[0].GetRootSignPublic(), mt.GetRootSignPublic("cb-0"), "chatbot is not consistent with member %d", j)
		}

		// Then the second chatbot update key without acknowledging that the first chatbot has updated.
		chatbotUpdate, newCbPubKey, newCbSignPubKey, err = chatbots[1].UpdateExternalNode()
		assert.Nilf(t, err, "error creating chatbot update: %s", err)
		for j, mt := range multiTreeKEMs {
			err := mt.HandleExternalNodeUpdate("cb-1", chatbotUpdate, newCbPubKey, newCbSignPubKey)
			assert.Nilf(t, err, "error handling chatbot update: %s", err)
			assert.Equal(t, chatbots[1].GetRootSecret(), mt.GetRootSecret("cb-1"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[1].GetRootPublic(), mt.GetRootPublic("cb-1"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[1].GetRootSignPublic(), mt.GetRootSignPublic("cb-1"), "chatbot is not consistent with member %d", j)
		}

		// The user issue update for the third chatbot and notify the chatbot
		userUpdate, ct, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err := multiTreeKEMs[0].UpdateTreeKEM([]string{"cb-2"})
		assert.Nilf(t, err, "error creating user update: %s", err)
		for j, mt := range multiTreeKEMs {
			if members[j].Index() == members[0].Index() {
				continue
			}
			err = mt.HandleTreeKEMUpdate(userUpdate, []string{"cb-2"})
			assert.Nilf(t, err, "error handling user update: %s", err)
			eq := groupEqual(members[0], members[j])
			assert.Truef(t, eq, "members %d -> %d are not equal", members[0].Index(), members[j].Index())
		}

		// The third chatbot applies the update
		err = chatbots[2].HandleTreeKEMUpdate(ct["cb-2"], newTreeKemRootPubKey, newTreeKemRootSignPubKey)
		assert.Nilf(t, err, "error updating chatbot: %s", err)

		// Verify with each member
		for j, mt := range multiTreeKEMs {
			assert.Equal(t, chatbots[2].GetRootSecret(), mt.GetRootSecret("cb-2"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[2].GetRootPublic(), mt.GetRootPublic("cb-2"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[2].GetRootSignPublic(), mt.GetRootSignPublic("cb-2"), "chatbot is not consistent with member %d", j)
		}

		// The first chatbot update key without acknowledging that the user has updated.
		chatbotUpdate, newCbPubKey, newCbSignPubKey, err = chatbots[0].UpdateExternalNode()
		assert.Nilf(t, err, "error creating chatbot update: %s", err)
		for j, mt := range multiTreeKEMs {
			err := mt.HandleExternalNodeUpdate("cb-0", chatbotUpdate, newCbPubKey, newCbSignPubKey)
			assert.Nilf(t, err, "error handling chatbot update: %s", err)
			assert.Equal(t, chatbots[0].GetRootSecret(), mt.GetRootSecret("cb-0"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[0].GetRootPublic(), mt.GetRootPublic("cb-0"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[0].GetRootSignPublic(), mt.GetRootSignPublic("cb-0"), "chatbot is not consistent with member %d", j)
		}

		// The second chatbot update key without acknowledging that the user has updated.
		chatbotUpdate, newCbPubKey, newCbSignPubKey, err = chatbots[1].UpdateExternalNode()
		assert.Nilf(t, err, "error creating chatbot update: %s", err)
		for j, mt := range multiTreeKEMs {
			err := mt.HandleExternalNodeUpdate("cb-1", chatbotUpdate, newCbPubKey, newCbSignPubKey)
			assert.Nilf(t, err, "error handling chatbot update: %s", err)
			assert.Equal(t, chatbots[1].GetRootSecret(), mt.GetRootSecret("cb-1"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[1].GetRootPublic(), mt.GetRootPublic("cb-1"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[1].GetRootSignPublic(), mt.GetRootSignPublic("cb-1"), "chatbot is not consistent with member %d", j)
		}

		// The second user issue update for the first chatbot and notify the chatbot
		userUpdate, ct, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err = multiTreeKEMs[1].UpdateTreeKEM([]string{"cb-0"})
		assert.Nilf(t, err, "error creating user update: %s", err)
		for j, mt := range multiTreeKEMs {
			if members[j].Index() == members[1].Index() {
				continue
			}
			err = mt.HandleTreeKEMUpdate(userUpdate, []string{"cb-0"})
			assert.Nilf(t, err, "error handling user update: %s", err)
			eq := groupEqual(members[1], members[j])
			assert.Truef(t, eq, "members %d -> %d are not equal", members[1].Index(), members[j].Index())
		}

		// The first chatbot applies the update
		err = chatbots[0].HandleTreeKEMUpdate(ct["cb-0"], newTreeKemRootPubKey, newTreeKemRootSignPubKey)
		assert.Nilf(t, err, "error updating chatbot: %s", err)

		// Verify with each member
		for j, mt := range multiTreeKEMs {
			assert.Equal(t, chatbots[0].GetRootSecret(), mt.GetRootSecret("cb-0"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[0].GetRootPublic(), mt.GetRootPublic("cb-0"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[0].GetRootSignPublic(), mt.GetRootSignPublic("cb-0"), "chatbot is not consistent with member %d", j)
		}

		// The third chatbot update key without acknowledging that the user has updated.
		chatbotUpdate, newCbPubKey, newCbSignPubKey, err = chatbots[2].UpdateExternalNode()
		assert.Nilf(t, err, "error creating chatbot update: %s", err)
		for j, mt := range multiTreeKEMs {
			err := mt.HandleExternalNodeUpdate("cb-2", chatbotUpdate, newCbPubKey, newCbSignPubKey)
			assert.Nilf(t, err, "error handling chatbot update: %s", err)
			assert.Equal(t, chatbots[2].GetRootSecret(), mt.GetRootSecret("cb-2"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[2].GetRootPublic(), mt.GetRootPublic("cb-2"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[2].GetRootSignPublic(), mt.GetRootSignPublic("cb-2"), "chatbot is not consistent with member %d", j)
		}

		// Add a new user to the treekem
		leaf, _ = generateRandomBytes(32)
		initKP, _ := NewKeyPairFromSecret(leaf)
		gaGroup, gaJoiner, _ := members[0].Add(initKP.Public.Bytes())
		joiner, _ := TreeKEMStateFromGroupAdd(leaf, gaJoiner)
		for _, m := range members {
			m.HandleGroupAdd(gaGroup)
		}
		members = append(members, joiner)
		joinerMT := NewMultiTreeKEM(joiner)
		multiTreeKEMs = append(multiTreeKEMs, joinerMT)

		// Get ciphertexts of all existing chatbots' secret
		chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts, err := multiTreeKEMs[0].GetExternalNodeJoinsWithoutUpdate(joiner.Self().Public)
		assert.Nilf(t, err, "error creating chatbot keys: %s", err)

		// Notify the new user of the chatbots
		err = joinerMT.SetExternalNodeJoinsWithoutUpdate(chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts)
		assert.Nilf(t, err, "error handling chatbot add: %s", err)

		// The new user issue a key update to chatbot 2
		userUpdate, ct, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err = joinerMT.UpdateTreeKEM([]string{"cb-2"})
		assert.Nilf(t, err, "error creating user update: %s", err)
		for j, mt := range multiTreeKEMs {
			if members[j].Index() == members[len(members)-1].Index() {
				continue
			}
			err = mt.HandleTreeKEMUpdate(userUpdate, []string{"cb-2"})
			assert.Nilf(t, err, "error handling user update: %s", err)
			eq := groupEqual(members[len(members)-1], members[j])
			assert.Truef(t, eq, "members %d -> %d are not equal", members[len(members)-1].Index(), members[j].Index())
		}

		// The third chatbot applies the update
		err = chatbots[2].HandleTreeKEMUpdate(ct["cb-2"], newTreeKemRootPubKey, newTreeKemRootSignPubKey)
		assert.Nilf(t, err, "error updating chatbot: %s", err)

		// Verify with each member
		for j, mt := range multiTreeKEMs {
			assert.Equal(t, chatbots[2].GetRootSecret(), mt.GetRootSecret("cb-2"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[2].GetRootPublic(), mt.GetRootPublic("cb-2"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[2].GetRootSignPublic(), mt.GetRootSignPublic("cb-2"), "chatbot is not consistent with member %d", j)
		}

		// Chatbot 2 issue a key update
		chatbotUpdate, newCbPubKey, newCbSignPubKey, err = chatbots[2].UpdateExternalNode()
		assert.Nilf(t, err, "error creating chatbot update: %s", err)

		// Have each member handle the chatbot update
		for j, mt := range multiTreeKEMs {
			err = mt.HandleExternalNodeUpdate("cb-2", chatbotUpdate, newCbPubKey, newCbSignPubKey)
			assert.Nilf(t, err, "error handling chatbot update: %s", err)
			assert.Equal(t, chatbots[2].GetRootSecret(), mt.GetRootSecret("cb-2"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[2].GetRootPublic(), mt.GetRootPublic("cb-2"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[2].GetRootSignPublic(), mt.GetRootSignPublic("cb-2"), "chatbot is not consistent with member %d", j)
		}

		// Add a new user to the treekem
		leaf, _ = generateRandomBytes(32)
		gik := members[0].GroupInitKey()
		ua, err := TreeKEMStateJoin(leaf, gik)
		joiner, err = TreeKEMStateFromUserAdd(leaf, gik)
		assert.Nilf(t, err, "error joining group: %s", err)

		for _, m := range members {
			m.HandleUserAdd(ua)
		}
		members = append(members, joiner)
		joinerMT = NewMultiTreeKEM(joiner)
		multiTreeKEMs = append(multiTreeKEMs, joinerMT)

		// Get ciphertexts of all existing chatbots' secret
		//chatbotPubKeys, lastTreeKemRootCiphertexts, err = multiTreeKEMs[0].GetExternalNodeJoinsWithoutUpdate(joiner.Self().Public)
		chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts, err = multiTreeKEMs[0].GetExternalNodeJoinsWithoutUpdate(ua.Nodes[(ua.Size-1)*2].Public)
		assert.Nilf(t, err, "error creating chatbot keys: %s", err)

		// Notify the new user of the chatbots
		err = joinerMT.SetExternalNodeJoinsWithoutUpdate(chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts)
		assert.Nilf(t, err, "error handling chatbot add: %s", err)

		// The first chatbot issues a key update
		chatbotUpdate, newCbPubKey, newCbSignPubKey, err = chatbots[0].UpdateExternalNode()
		assert.Nilf(t, err, "error creating chatbot update: %s", err)
		for j, mt := range multiTreeKEMs {
			err := mt.HandleExternalNodeUpdate("cb-0", chatbotUpdate, newCbPubKey, newCbSignPubKey)
			assert.Nilf(t, err, "error handling chatbot update: %s", err)
			assert.Equal(t, chatbots[0].GetRootSecret(), mt.GetRootSecret("cb-0"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[0].GetRootPublic(), mt.GetRootPublic("cb-0"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[0].GetRootSignPublic(), mt.GetRootSignPublic("cb-0"), "chatbot is not consistent with member %d", j)

			eq := groupEqual(members[0], members[j])
			assert.Truef(t, eq, "members %d -> %d are not equal", members[1].Index(), members[j].Index())
		}

		// The new user issues a key update
		userUpdate, ct, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err = joinerMT.UpdateTreeKEM([]string{"cb-0"})
		assert.Nilf(t, err, "error creating user update: %s", err)
		for j, mt := range multiTreeKEMs {
			if members[j].Index() == members[len(members)-1].Index() {
				continue
			}
			err = mt.HandleTreeKEMUpdate(userUpdate, []string{"cb-0"})
			assert.Nilf(t, err, "error handling user update: %s", err)
			eq := groupEqual(members[len(members)-1], members[j])
			assert.Truef(t, eq, "members %d -> %d are not equal", members[len(members)-1].Index(), members[j].Index())
		}

		// The first chatbot applies the update
		err = chatbots[0].HandleTreeKEMUpdate(ct["cb-0"], newTreeKemRootPubKey, newTreeKemRootSignPubKey)
		assert.Nilf(t, err, "error updating chatbot: %s", err)

		// Verify with each member
		for j, mt := range multiTreeKEMs {
			assert.Equal(t, chatbots[0].GetRootSecret(), mt.GetRootSecret("cb-0"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[0].GetRootPublic(), mt.GetRootPublic("cb-0"), "chatbot is not consistent with member %d", j)
			assert.Equal(t, chatbots[0].GetRootSignPublic(), mt.GetRootSignPublic("cb-0"), "chatbot is not consistent with member %d", j)
		}

	}
}
