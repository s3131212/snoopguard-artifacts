package util

import (
	"chatbot-poc-go/pkg/stores"
	"github.com/s3131212/go-mls"
	"go.mau.fi/libsignal/groups"
	"go.mau.fi/libsignal/keys/identity"
	"go.mau.fi/libsignal/keys/prekey"
	"go.mau.fi/libsignal/protocol"
	"go.mau.fi/libsignal/serialize"
	"go.mau.fi/libsignal/session"
	"go.mau.fi/libsignal/state/record"
	"go.mau.fi/libsignal/util/keyhelper"
	"go.mau.fi/libsignal/util/optional"
)

// User is a structure for a signal User.
type User struct {
	UserID   string
	deviceID uint32

	// Signal
	address *protocol.SignalAddress

	identityKeyPair *identity.KeyPair
	registrationID  uint32

	sessionStore      *stores.InMemorySessionStore
	preKeyStore       *stores.InMemoryPreKeyStore
	signedPreKeyStore *stores.InMemorySignedPreKeyStore
	identityStore     *stores.InMemoryIdentityKeyStore
	senderKeyStore    *stores.InMemorySenderKeyStore

	Serializer *serialize.Serializer

	sessionBuilder *session.Builder
	groupBuilder   *groups.SessionBuilder

	// MLS
	mlsIdentityPriv  mls.SignaturePrivateKey
	mlsCredential    mls.Credential
	mlsInitialSecret []byte

	mlsKeyPackageStore *stores.InMemoryKeyPackageStore
}

func (u *User) GeneratePreKey(start int) uint32 {
	preKeys, err := keyhelper.GeneratePreKeys(start, start+1, u.Serializer.PreKeyRecord)

	if err != nil {
		panic(err)
	}

	u.preKeyStore.StorePreKey(
		preKeys[0].ID().Value,
		preKeys[0],
	)

	return preKeys[0].ID().Value
}

func (u *User) GenerateSignedPreKey() uint32 {
	signedPreKey, _ := keyhelper.GenerateSignedPreKey(u.identityKeyPair, 0, u.Serializer.SignedPreKeyRecord)

	u.signedPreKeyStore.StoreSignedPreKey(
		signedPreKey.ID(),
		record.NewSignedPreKey(
			signedPreKey.ID(),
			signedPreKey.Timestamp(),
			signedPreKey.KeyPair(),
			signedPreKey.Signature(),
			u.Serializer.SignedPreKeyRecord,
		),
	)

	return signedPreKey.ID()
}

func (u *User) GetPreKey(preKeyID uint32) *record.PreKey {
	return u.preKeyStore.LoadPreKey(preKeyID)
}

func (u *User) GetSignedPreKey(signedPreKeyID uint32) *record.SignedPreKey {
	return u.signedPreKeyStore.LoadSignedPreKey(signedPreKeyID)
}

func (u *User) GetPreKeyBundle(preKeyID uint32, signedPreKeyID uint32) *prekey.Bundle {
	preKey := u.preKeyStore.LoadPreKey(preKeyID)
	signedPreKeyPair := u.signedPreKeyStore.LoadSignedPreKey(signedPreKeyID)
	return prekey.NewBundle(
		u.registrationID,
		1,
		&optional.Uint32{Value: preKeyID},
		signedPreKeyID,
		preKey.KeyPair().PublicKey(),
		signedPreKeyPair.KeyPair().PublicKey(),
		signedPreKeyPair.Signature(),
		u.identityKeyPair.PublicKey(),
	)
}

// GenerateMLSKeys generates MLS keys and credentials.
func (u *User) GenerateMLSKeys() {
	suite := mls.P256_AES128GCM_SHA256_P256
	scheme := suite.Scheme()
	secret := RandomBytes(32)
	sigPriv, err := scheme.Derive(secret)
	if err != nil {
		panic(err)
	}
	cred := mls.NewBasicCredential([]byte(u.UserID), scheme, sigPriv.PublicKey)

	u.mlsInitialSecret = secret
	u.mlsIdentityPriv = sigPriv
	u.mlsCredential = *cred
}

// GenerateMLSKeyPackage generates a new MLS key package.
func (u *User) GenerateMLSKeyPackage(id uint32) {
	kp, err := mls.NewKeyPackageWithSecret(mls.P256_AES128GCM_SHA256_P256, u.mlsInitialSecret, &u.mlsCredential, u.mlsIdentityPriv)
	if err != nil {
		panic(err)
	}

	u.mlsKeyPackageStore.StoreKeyPackage(id, *kp)
}

// GetMLSInitialSecret returns the MLS initial secret.
func (u *User) GetMLSInitialSecret() []byte {
	return u.mlsInitialSecret
}

// GetMLSKeyPackage returns the MLS key package.
func (u *User) GetMLSKeyPackage(id uint32) mls.KeyPackage {
	return u.mlsKeyPackageStore.LoadKeyPackage(id)
}

// GenerateMLSStateFromEmpty generates a new MLS state from the empty state.
func (u *User) GenerateMLSStateFromEmpty(groupId []byte, keyPackage mls.KeyPackage) (*mls.State, error) {
	return mls.NewEmptyState(groupId, u.GetMLSInitialSecret(), u.GetMLSIdentityKey(), keyPackage)

}

// GenerateMLSStateFromWelcome generates a new MLS state from a welcome message.
func (u *User) GenerateMLSStateFromWelcome(welcome *mls.Welcome, keyPackage mls.KeyPackage) (*mls.State, error) {
	return mls.NewJoinedState(u.GetMLSInitialSecret(), []mls.SignaturePrivateKey{u.GetMLSIdentityKey()}, []mls.KeyPackage{keyPackage}, *welcome)
}

// GetMLSCredential returns the MLS credential.
func (u *User) GetMLSCredential() mls.Credential {
	return u.mlsCredential
}

// GetMLSIdentityKey returns the MLS identity private key.
func (u *User) GetMLSIdentityKey() mls.SignaturePrivateKey {
	return u.mlsIdentityPriv
}

// CreateSessionWrapper will build a session with the given address.
func (u *User) CreateSessionWrapper(address *protocol.SignalAddress, preKeyBundle *prekey.Bundle) *SessionWrapper {
	//if u.sessionStore.ContainsSession(address) {
	//	return u.sessionStore.LoadSession(address)
	//}
	//return u.createSession(address, serializer)
	return NewSessionWrapper(address, u, preKeyBundle, u.Serializer)
}

// BuildGroupSession will build a session with the given address.
func (u *User) BuildGroupSession() {
	if u.groupBuilder != nil {
		return
	}
	u.groupBuilder = groups.NewGroupSessionBuilder(u.senderKeyStore, u.Serializer)
}

// GetIdentityKey returns the identity key for the user.
func (u *User) GetIdentityKey() *identity.KeyPair {
	return u.identityKeyPair
}

// GetRegistrationID returns the registration ID for the user.
func (u *User) GetRegistrationID() uint32 {
	return u.registrationID
}

// NewUser creates a new signal and MLS User for session testing.
func NewUser(userID string, deviceID uint32, serializer *serialize.Serializer) *User {
	user := &User{}

	// Generate an identity keypair
	user.identityKeyPair, _ = keyhelper.GenerateIdentityKeyPair()

	// Generate a registration id
	user.registrationID = keyhelper.GenerateRegistrationID()

	// Create all our record stores using an in-memory implementation.
	user.sessionStore = stores.NewInMemorySessionStore(serializer)
	user.preKeyStore = stores.NewInMemoryPreKeyStore()
	user.signedPreKeyStore = stores.NewInMemorySignedPreKeyStore()
	user.identityStore = stores.NewInMemoryIdentityKeyStore(user.identityKeyPair, user.registrationID)
	user.senderKeyStore = stores.NewInMemorySenderKeyStore()

	// Create a remote address that we'll be building our session with.
	user.UserID = userID
	user.deviceID = deviceID
	user.address = protocol.NewSignalAddress(userID, deviceID)

	user.Serializer = serializer

	// Generate MLS key
	user.mlsKeyPackageStore = stores.NewInMemoryKeyPackageStore()
	user.GenerateMLSKeys()

	return user
}
