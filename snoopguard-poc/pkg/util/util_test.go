package util

import (
	"crypto/ecdh"
	"crypto/sha256"
	"github.com/s3131212/go-mls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mau.fi/libsignal/protocol"
	"go.mau.fi/libsignal/serialize"
	"testing"
)

func TestNewUser(t *testing.T) {
	serializer := serialize.NewProtoBufSerializer()
	user := NewUser("Alice", 1, serializer)
	assert.Equal(t, user.UserID, "Alice", "the user ID should be Alice")
}

func TestGetKeys(t *testing.T) {
	serializer := serialize.NewProtoBufSerializer()
	user := NewUser("Alice", 1, serializer)

	/* Test get identity key */
	assert.NotNil(t, user.GetIdentityKey(), "should return the identity key")

	/* Test generating and getting prekeys */
	preKeyID := user.GeneratePreKey(0)
	assert.NotNil(t, user.GetPreKey(preKeyID), "should return the prekey")

	/* Test generating and getting signed prekeys */
	signedPreKeyID := user.GenerateSignedPreKey()
	assert.NotNil(t, user.GetSignedPreKey(signedPreKeyID))

	/* Test get MLS identity key */
	assert.NotNil(t, user.GetMLSIdentityKey(), "should return the MLS identity key")

	/* Test get MLS credential */
	assert.NotNil(t, user.GetMLSCredential(), "should return the MLS credential")

	/* Test MLS key package */
	user.GenerateMLSKeyPackage(1)
	assert.NotNil(t, user.GetMLSKeyPackage(1), "should return the MLS key package")
}

func TestPeerMessages(t *testing.T) {
	serializer := serialize.NewProtoBufSerializer()
	alice := NewUser("Alice", 1, serializer)
	bob := NewUser("Bob", 1, serializer)

	alicePreKeyID := alice.GeneratePreKey(0)
	aliceSignedPreKeyID := alice.GenerateSignedPreKey()

	bobPreKeyID := bob.GeneratePreKey(0)
	bobSignedPreKeyID := bob.GenerateSignedPreKey()

	//aliceIdentityKeyPublic := alice.GetIdentityKey().PublicKey()
	//bobIdentityKeyPublic := bob.GetIdentityKey().PublicKey()

	alicePreKeyBundle := alice.GetPreKeyBundle(alicePreKeyID, aliceSignedPreKeyID)
	bobPreKeyBundle := bob.GetPreKeyBundle(bobPreKeyID, bobSignedPreKeyID)

	aliceSession := alice.CreateSessionWrapper(protocol.NewSignalAddress("Bob", 1), bobPreKeyBundle)
	bobSession := bob.CreateSessionWrapper(protocol.NewSignalAddress("Alice", 1), alicePreKeyBundle)

	aliceMsg1 := aliceSession.EncryptMsg([]byte("Alice Message 1"))
	bobReceiveAliceMsg1, err := bobSession.DecryptMsg(aliceMsg1)
	assert.Nil(t, err, "should not return error")
	assert.Equal(t, "Alice Message 1", string(bobReceiveAliceMsg1), "Alice's first message should be the same")

	bobMsg1 := bobSession.EncryptMsg([]byte("Bob Message 1"))
	aliceReceiveBobMsg1, err := aliceSession.DecryptMsg(bobMsg1)
	assert.Nil(t, err, "should not return error")
	assert.Equal(t, "Bob Message 1", string(aliceReceiveBobMsg1), "Bob's first message should be the same")

	aliceMsg2 := aliceSession.EncryptMsg([]byte("Alice Message 2"))
	aliceMsg3 := aliceSession.EncryptMsg([]byte("Alice Message 3"))

	bobReceiveAliceMsg3, err := bobSession.DecryptMsg(aliceMsg3)
	assert.Nil(t, err, "should not return error")
	assert.Equal(t, "Alice Message 3", string(bobReceiveAliceMsg3), "Out of order messages should not failed")

	bobReceiveAliceMsg2, err := bobSession.DecryptMsg(aliceMsg2)
	assert.Nil(t, err, "should not return error")
	assert.Equal(t, "Alice Message 2", string(bobReceiveAliceMsg2), "Out of order messages should not failed")

}

func TestServerGroupMessages(t *testing.T) {
	serializer := serialize.NewProtoBufSerializer()
	alice := NewUser("Alice", 1, serializer)
	bob := NewUser("Bob", 1, serializer)
	carol := NewUser("Carol", 1, serializer)

	aliceGroupSession := NewGroupChatServerSideFanout(alice, protocol.NewSenderKeyName("test group", protocol.NewSignalAddress("Alice", 1)))
	bobGroupSession := NewGroupChatServerSideFanout(bob, protocol.NewSenderKeyName("test group", protocol.NewSignalAddress("Bob", 1)))
	carolGroupSession := NewGroupChatServerSideFanout(carol, protocol.NewSenderKeyName("test group", protocol.NewSignalAddress("Carol", 1)))

	aliceGroupSession.CreateSendingGroupSession()
	aliceSenderKey := aliceGroupSession.GetSendingGroupSession().DistributeSenderKey().Serialize()
	bobAliceSenderKey, _ := protocol.NewSenderKeyDistributionMessageFromBytes(aliceSenderKey, alice.Serializer.SenderKeyDistributionMessage)
	bobGroupSession.CreateReceivingGroupSession("Alice", bobAliceSenderKey)

	ciphertextFromAlice1 := aliceGroupSession.GetSendingGroupSession().EncryptGroupMessage([]byte("Alice Message 1")).SignedSerialize()
	bobCiphertextFromAlice1, _ := protocol.NewSenderKeyMessageFromBytes(ciphertextFromAlice1, bob.Serializer.SenderKeyMessage)
	assert.Equal(t, "Alice Message 1", string(bobGroupSession.GetReceivingGroupSession("Alice").DecryptGroupMessage(bobCiphertextFromAlice1)))
	ciphertextFromAlice2 := aliceGroupSession.GetSendingGroupSession().EncryptGroupMessage([]byte("Alice Message 2")).SignedSerialize()
	bobCiphertextFromAlice2, _ := protocol.NewSenderKeyMessageFromBytes(ciphertextFromAlice2, bob.Serializer.SenderKeyMessage)
	assert.Equal(t, "Alice Message 2", string(bobGroupSession.GetReceivingGroupSession("Alice").DecryptGroupMessage(bobCiphertextFromAlice2)))

	bobGroupSession.CreateSendingGroupSession()
	bobSenderKey := bobGroupSession.GetSendingGroupSession().DistributeSenderKey().Serialize()

	aliceBobSenderKey, _ := protocol.NewSenderKeyDistributionMessageFromBytes(bobSenderKey, alice.Serializer.SenderKeyDistributionMessage)
	aliceGroupSession.CreateReceivingGroupSession("Bob", aliceBobSenderKey)

	carolBobSenderKey, _ := protocol.NewSenderKeyDistributionMessageFromBytes(bobSenderKey, carol.Serializer.SenderKeyDistributionMessage)
	carolGroupSession.CreateReceivingGroupSession("Bob", carolBobSenderKey)

	ciphertextFromBob1 := bobGroupSession.GetSendingGroupSession().EncryptGroupMessage([]byte("Bob Message 1")).SignedSerialize()
	ciphertextFromBob2 := bobGroupSession.GetSendingGroupSession().EncryptGroupMessage([]byte("Bob Message 2")).SignedSerialize()

	aliceCiphertextFromBob1, _ := protocol.NewSenderKeyMessageFromBytes(ciphertextFromBob1, alice.Serializer.SenderKeyMessage)
	aliceCiphertextFromBob2, _ := protocol.NewSenderKeyMessageFromBytes(ciphertextFromBob2, alice.Serializer.SenderKeyMessage)
	assert.Equal(t, "Bob Message 1", string(aliceGroupSession.GetReceivingGroupSession("Bob").DecryptGroupMessage(aliceCiphertextFromBob1)))
	assert.Equal(t, "Bob Message 2", string(aliceGroupSession.GetReceivingGroupSession("Bob").DecryptGroupMessage(aliceCiphertextFromBob2)))

	carolCiphertextFromBob1, _ := protocol.NewSenderKeyMessageFromBytes(ciphertextFromBob1, carol.Serializer.SenderKeyMessage)
	carolCiphertextFromBob2, _ := protocol.NewSenderKeyMessageFromBytes(ciphertextFromBob2, carol.Serializer.SenderKeyMessage)
	assert.Equal(t, "Bob Message 2", string(carolGroupSession.GetReceivingGroupSession("Bob").DecryptGroupMessage(carolCiphertextFromBob2)))
	assert.Equal(t, "Bob Message 1", string(carolGroupSession.GetReceivingGroupSession("Bob").DecryptGroupMessage(carolCiphertextFromBob1)))
}

func TestEncrypt(t *testing.T) {
	plaintext := []byte(RandomString(100))
	key := []byte("1234567890123456")

	ciphertext, err := Encrypt(plaintext, key, nil)
	assert.Nil(t, err, "should not return error")

	ciphertextSerialized := ciphertext.Serialize()
	ciphertextDeserialized := DeserializeCipherText(ciphertextSerialized)

	decrypted, err := Decrypt(ciphertextDeserialized, key, nil)
	assert.Nil(t, err, "should not return error")

	assert.Equal(t, plaintext, decrypted, "should be the same")
}

func TestMLSPeerMessage(t *testing.T) {
	serializer := serialize.NewProtoBufSerializer()

	alice := NewUser("Alice", 1, serializer)
	alice.GenerateMLSKeyPackage(1)

	bob := NewUser("Bob", 1, serializer)
	bob.GenerateMLSKeyPackage(1)

	groupId := RandomBytes(32)

	aliceState, err := mls.NewEmptyState(groupId, alice.GetMLSInitialSecret(), alice.GetMLSIdentityKey(), alice.GetMLSKeyPackage(1))
	assert.Nil(t, err, "should not return error")

	add, err := aliceState.Add(bob.GetMLSKeyPackage(1))
	assert.Nil(t, err, "should not return error")
	_, err = aliceState.Handle(add)
	assert.Nil(t, err, "should not return error")

	secret := RandomBytes(32)
	_, welcome, aliceState, err := aliceState.Commit(secret)
	assert.Nil(t, err, "should not return error")
	//require.Equal(t, aliceState.NewCredentials, map[LeafIndex]bool{1: true})

	bobState, err := mls.NewJoinedState(bob.GetMLSInitialSecret(), []mls.SignaturePrivateKey{bob.GetMLSIdentityKey()}, []mls.KeyPackage{bob.GetMLSKeyPackage(1)}, *welcome)
	require.Nil(t, err)
	//require.Equal(t, bobState.NewCredentials, map[LeafIndex]bool{0: true, 1: true})

	// Verify that the two states are equivalent
	require.True(t, aliceState.Equals(*bobState))

	// Verify that they can exchange protected messages
	testMessage := []byte("Hello, world!")
	ct, err := aliceState.Protect(testMessage)
	require.Nil(t, err)

	serializedCt, err := SerializeMLSCiphertext(*ct)
	require.Nil(t, err)
	deserializedCt, err := DeserializeMLSCiphertext(serializedCt)
	require.Nil(t, err)

	pt, err := bobState.Unprotect(&deserializedCt)
	require.Nil(t, err)
	require.Equal(t, pt, testMessage)
}

func TestMLSGroupMessage(t *testing.T) {
	serializer := serialize.NewProtoBufSerializer()

	alice := NewUser("Alice", 1, serializer)
	alice.GenerateMLSKeyPackage(1)

	bob := NewUser("Bob", 1, serializer)
	bob.GenerateMLSKeyPackage(1)

	groupId := RandomBytes(32)

	aliceState, err := mls.NewEmptyState(groupId, alice.GetMLSInitialSecret(), alice.GetMLSIdentityKey(), alice.GetMLSKeyPackage(1))
	assert.Nil(t, err, "should not return error")

	// add proposals for rest of the participants
	add, err := aliceState.Add(bob.GetMLSKeyPackage(1))
	assert.Nil(t, err, "should not return error")
	_, err = aliceState.Handle(add)
	assert.Nil(t, err, "should not return error")

	// commit the adds
	secret := RandomBytes(32)
	_, welcome, aliceState, err := aliceState.Commit(secret)
	require.Nil(t, err)
	// initialize the new joiners from the welcome
	bobState, err := mls.NewJoinedState(bob.GetMLSInitialSecret(), []mls.SignaturePrivateKey{bob.GetMLSIdentityKey()}, []mls.KeyPackage{bob.GetMLSKeyPackage(1)}, *welcome)
	require.Nil(t, err)

	// Verify that the states are all equivalent
	require.True(t, aliceState.Equals(*bobState))

	// verify that everyone can send and be received
	testMessage := []byte("Hello, world!")
	ct, _ := aliceState.Protect(testMessage)
	secret = RandomBytes(32)
	commit, _, aliceState, err := aliceState.Commit(secret)

	pt, _ := bobState.Unprotect(ct)
	require.Equal(t, pt, testMessage)
	bobState, err = bobState.Handle(commit)
	require.True(t, aliceState.Equals(*bobState))

	ct, _ = bobState.Protect(testMessage)
	secret = RandomBytes(32)
	commit, _, bobState, err = bobState.Commit(secret)

	pt, _ = aliceState.Unprotect(ct)
	require.Equal(t, pt, testMessage)
	aliceState, err = aliceState.Handle(commit)
	require.True(t, aliceState.Equals(*bobState))

	// Alice adds another member
	carol := NewUser("Carol", 1, serializer)
	carol.GenerateMLSKeyPackage(1)

	add, err = aliceState.Add(carol.GetMLSKeyPackage(1))
	assert.Nil(t, err, "should not return error")
	_, err = aliceState.Handle(add)
	assert.Nil(t, err, "should not return error")

	// commit the adds
	secret = RandomBytes(32)
	addCommit, welcome, aliceState, err := aliceState.Commit(secret)
	require.Nil(t, err)

	carolState, err := mls.NewJoinedState(carol.GetMLSInitialSecret(), []mls.SignaturePrivateKey{carol.GetMLSIdentityKey()}, []mls.KeyPackage{carol.GetMLSKeyPackage(1)}, *welcome)
	require.Nil(t, err)

	_, err = bobState.Handle(add)
	bobState, err = bobState.Handle(addCommit)
	require.Nil(t, err)

	// Verify that the states are all equivalent
	require.True(t, aliceState.Equals(*bobState))
	require.True(t, aliceState.Equals(*carolState))
	require.True(t, bobState.Equals(*carolState))

	// Bob adds another member
	dave := NewUser("Dave", 1, serializer)
	dave.GenerateMLSKeyPackage(1)

	add, err = bobState.Add(dave.GetMLSKeyPackage(1))
	assert.Nil(t, err, "should not return error")
	_, err = bobState.Handle(add)

	// Commit the add
	secret = RandomBytes(32)
	addCommit, welcome, bobState, err = bobState.Commit(secret)
	require.Nil(t, err)

	// initialize the new joiners from the welcome
	daveState, err := mls.NewJoinedState(dave.GetMLSInitialSecret(), []mls.SignaturePrivateKey{dave.GetMLSIdentityKey()}, []mls.KeyPackage{dave.GetMLSKeyPackage(1)}, *welcome)
	require.Nil(t, err)

	_, err = aliceState.Handle(add)
	aliceState, err = aliceState.Handle(addCommit)
	_, err = carolState.Handle(add)
	carolState, err = carolState.Handle(addCommit)

	// Verify that the states are all equivalent
	require.True(t, aliceState.Equals(*bobState))
	require.True(t, aliceState.Equals(*carolState))
	require.True(t, aliceState.Equals(*daveState))
	require.True(t, bobState.Equals(*carolState))
	require.True(t, bobState.Equals(*daveState))
	require.True(t, carolState.Equals(*daveState))

	// Test update

}

func TestEncryptAndSign(t *testing.T) {
	plaintext := []byte(RandomString(100))
	key := []byte("1234567890123456")
	kp, err := newKeyPairFromSecret(key)
	assert.Nil(t, err, "should not return error")

	ciphertext, err := Encrypt(plaintext, key, kp.Private.Bytes())
	assert.Nil(t, err, "should not return error")

	ciphertextSerialized := ciphertext.Serialize()
	ciphertextDeserialized := DeserializeCipherText(ciphertextSerialized)

	assert.Equal(t, ciphertext.IV, ciphertextDeserialized.IV, "IV should be the same")
	assert.Equal(t, ciphertext.Signature, ciphertextDeserialized.Signature, "signature should be the same")
	assert.Equal(t, ciphertext.CipherText, ciphertextDeserialized.CipherText, "ciphertext should be the same")

	decrypted, err := Decrypt(ciphertextDeserialized, key, kp.Public.Bytes())
	assert.Nil(t, err, "should not return error")

	assert.Equal(t, plaintext, decrypted, "should be the same")
}

func TestMLSSerialization(t *testing.T) {
	serializer := serialize.NewProtoBufSerializer()
	alice := NewUser("Alice", 1, serializer)
	for i := 0; i < 100; i++ {
		alice.GenerateMLSKeyPackage(uint32(i))

		kp := alice.GetMLSKeyPackage(uint32(i))
		serialized, err := SerializeMLSKeyPackage(kp)
		require.Nil(t, err, "should not return error")

		deserialized, err := DeserializeMLSKeyPackage(serialized)
		require.Nil(t, err, "should not return error")

		assert.True(t, kp.Equals(deserialized), "should be the same")
	}
}

type Keypair struct {
	Private *ecdh.PrivateKey
	Public  *ecdh.PublicKey
}

func newKeyPairFromSecret(secret []byte) (*Keypair, error) {
	// Calculate SHA-256 of the secret
	digest := sha256.Sum256(secret)

	privateKey, err := ecdh.P256().NewPrivateKey(digest[:])
	if err != nil {
		return nil, err
	}

	publicKey := privateKey.PublicKey()
	return &Keypair{
		Private: privateKey,
		Public:  publicKey,
	}, nil
}
