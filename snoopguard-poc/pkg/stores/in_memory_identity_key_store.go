package stores

import (
	"go.mau.fi/libsignal/keys/identity"
	"go.mau.fi/libsignal/protocol"
	"sync"
)

func NewInMemoryIdentityKeyStore(identityKey *identity.KeyPair, localRegistrationID uint32) *InMemoryIdentityKeyStore {
	return &InMemoryIdentityKeyStore{
		trustedKeys:         make(map[*protocol.SignalAddress]*identity.Key),
		identityKeyPair:     identityKey,
		localRegistrationID: localRegistrationID,
	}
}

type InMemoryIdentityKeyStore struct {
	trustedKeys         map[*protocol.SignalAddress]*identity.Key
	identityKeyPair     *identity.KeyPair
	localRegistrationID uint32
	mutexLock           sync.RWMutex
}

func (i *InMemoryIdentityKeyStore) GetIdentityKeyPair() *identity.KeyPair {
	return i.identityKeyPair
}

func (i *InMemoryIdentityKeyStore) GetLocalRegistrationId() uint32 {
	return i.localRegistrationID
}

func (i *InMemoryIdentityKeyStore) SaveIdentity(address *protocol.SignalAddress, identityKey *identity.Key) {
	i.mutexLock.Lock()
	i.trustedKeys[address] = identityKey
	i.mutexLock.Unlock()
}

func (i *InMemoryIdentityKeyStore) IsTrustedIdentity(address *protocol.SignalAddress, identityKey *identity.Key) bool {
	i.mutexLock.RLock()
	trusted := i.trustedKeys[address]
	i.mutexLock.RUnlock()
	return (trusted == nil || trusted.Fingerprint() == identityKey.Fingerprint())
}
