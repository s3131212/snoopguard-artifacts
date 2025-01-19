package treekem

import (
	"crypto/rand"
	"github.com/stretchr/testify/assert"
	"testing"
)

// GenerateRandomBytes returns securely generated random bytes.
func generateRandomBytes(n int) ([]byte, error) {
	token := make([]byte, n)
	_, err := rand.Read(token)
	return token, err
}

func testMembers(size int) []*TreeKEM {
	nodeWidth := nodeWidth(size)
	keyPairs := make([]*Keypair, nodeWidth)
	nodes := make(map[int]*Node)

	for i := 0; i < nodeWidth; i++ {
		keyPairs[i], _ = NewKeyPair()
	}

	for i, kp := range keyPairs {
		nodes[i] = &Node{
			Private: kp.Private.Bytes(),
			Public:  kp.Public.Bytes(),
		}
	}

	members := make([]*TreeKEM, size)
	root := root(size)

	for i := 0; i < size; i++ {
		members[i] = NewTreeKEM()
		members[i].Size = size
		members[i].Index = i

		// Public keys along its copath
		for _, n := range trCopath(2*i, size) {
			members[i].Nodes[n] = &Node{
				Public: nodes[n].Public,
			}
		}

		// Private keys along its direct path
		dirpath := trDirpath(2*i, size)
		dirpath = append(dirpath, root)
		for _, n := range dirpath {
			members[i].Nodes[n] = &Node{
				Private: nodes[n].Private,
				Public:  nodes[n].Public,
			}
		}
	}

	return members
}

func TestECKEM(t *testing.T) {
	original := []byte{0, 1, 2, 3}
	kp, err := NewKeyPair()
	assert.Nilf(t, err, "error generating key pair: %s", err)

	encrypted, err := ECKEMEncrypt(original, kp.Public.Bytes())
	assert.Nilf(t, err, "error encrypting: %s", err)

	decrypted, err := ECKEMDecrypt(encrypted, kp.Private.Bytes())
	assert.Nilf(t, err, "error decrypting: %s", err)

	assert.Equal(t, original, decrypted, "decrypted value should equal original")
}

func TestEncryptDecrypt(t *testing.T) {
	for testGroupSize := 1; testGroupSize <= 32; testGroupSize++ {
		seed := []byte("test seed")

		// Create a group with the specified size
		members := testMembers(testGroupSize)

		// Have each member send and be received by all members
		for _, m := range members {
			ct := m.Encrypt(seed, m.Index)
			privateNodes := hashUp(2*m.Index, m.Size, seed)

			m.merge(ct.Nodes, false)
			m.merge(privateNodes, false)

			for _, m2 := range members {
				if m2.Index == m.Index {
					continue
				}

				pt := m2.Decrypt(m.Index, ct.Ciphertexts)

				// Merge public values, then private
				m2.merge(ct.Nodes, false)
				m2.merge(pt.Nodes, false)

				assert.Truef(t, m.equal(m2), "members %d -> %d are not equal", m.Index, m2.Index)
			}
		}
	}
}

func TestUserAdd(t *testing.T) {
	for testGroupSize := 1; testGroupSize <= 32; testGroupSize++ {
		leaf, _ := generateRandomBytes(32)
		creator := TreeKEMStateOneMemberGroup(leaf)
		members := []*TreeKEMState{creator}

		for i := 1; i < testGroupSize; i++ {
			leaf, _ = generateRandomBytes(32)
			gik := members[0].GroupInitKey()
			ua, err := TreeKEMStateJoin(leaf, gik)
			assert.Nilf(t, err, "error joining group: %s", err)

			joiner, err := TreeKEMStateFromUserAdd(leaf, gik)
			assert.Nilf(t, err, "error creating user add: %s", err)

			for _, m := range members {
				m.HandleUserAdd(ua)
				eq := groupEqual(joiner, m)
				assert.Truef(t, eq, "members %d -> %d are not equal", joiner.Index(), m.Index())
			}

			members = append(members, joiner)
		}
	}
}

func TestGroupAdd(t *testing.T) {
	for testGroupSize := 1; testGroupSize <= 32; testGroupSize++ {
		leaf, _ := generateRandomBytes(32)
		creator := TreeKEMStateOneMemberGroup(leaf)
		members := []*TreeKEMState{creator}

		for i := 1; i < testGroupSize; i++ {
			leaf, _ = generateRandomBytes(32)
			initKP, err := NewKeyPairFromSecret(leaf)
			assert.Nilf(t, err, "error creating key pair: %s", err)

			gaGroup, gaJoiner, err := members[len(members)-1].Add(initKP.Public.Bytes())
			assert.Nilf(t, err, "error creating group add: %s", err)

			joiner, err := TreeKEMStateFromGroupAdd(leaf, gaJoiner)
			assert.Nilf(t, err, "error handling group add: %s", err)

			for _, m := range members {
				m.HandleGroupAdd(gaGroup)
				eq := groupEqual(joiner, m)
				assert.Truef(t, eq, "members %d -> %d are not equal", joiner.Index(), m.Index())
			}

			members = append(members, joiner)
		}
	}
}

func TestUpdate(t *testing.T) {
	for testGroupSize := 1; testGroupSize <= 32; testGroupSize++ {
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

		// Have each member update and verify that others are consistent
		for _, m1 := range members {
			leaf, _ = generateRandomBytes(32)
			userUpdate := m1.Update(leaf)

			userUpdate = PbTreeKEMUserUpdateConvert(TreeKEMUserUpdatePbConvert(&userUpdate))

			m1.HandleSelfUpdate(userUpdate, leaf)

			for _, m2 := range members {
				if m2.Index() == m1.Index() {
					continue
				}
				m2.HandleUpdate(userUpdate)
				eq := groupEqual(m1, m2)
				assert.Truef(t, eq, "members %d -> %d are not equal", m1.Index(), m2.Index())
			}
		}
	}
}

func TestRemove(t *testing.T) {
	for testGroupSize := 4; testGroupSize <= 32; testGroupSize++ {
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

		// Have the first member remove two members
		remover := members[0]
		removed := []int{2, 3}
		for _, index := range removed {
			leaf, _ = generateRandomBytes(32)
			remove := remover.Remove(leaf, index, map[int]*Node{})

			var newMembers []*TreeKEMState
			for _, m := range members {
				if m.Index() != index {
					newMembers = append(newMembers, m)
					m.HandleRemove(remove)
				}
			}

			members = newMembers

			for _, m := range members {
				eq := groupEqual(remover, m)
				assert.Truef(t, eq, "members %d -> %d are not equal", remover.Index(), m.Index())
			}
		}

		// Have each remaining member update and verify that others are consistent
		for _, m1 := range members {
			leaf, _ = generateRandomBytes(32)
			update := m1.Update(leaf)
			m1.HandleSelfUpdate(update, leaf)

			for _, m2 := range members {
				if m2.Index() == m1.Index() {
					continue
				}

				m2.HandleUpdate(update)

				eq := groupEqual(m1, m2)
				assert.Truef(t, eq, "members %d -> %d are not equal", m1.Index(), m2.Index())
			}
		}
	}
}
