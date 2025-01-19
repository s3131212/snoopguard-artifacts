package stores

import (
	"go.mau.fi/libsignal/state/record"
	"sync"
)

func NewInMemorySignedPreKeyStore() *InMemorySignedPreKeyStore {
	return &InMemorySignedPreKeyStore{
		store: make(map[uint32]*record.SignedPreKey),
	}
}

type InMemorySignedPreKeyStore struct {
	store     map[uint32]*record.SignedPreKey
	mutexLock sync.RWMutex
}

func (i *InMemorySignedPreKeyStore) LoadSignedPreKey(signedPreKeyID uint32) *record.SignedPreKey {
	i.mutexLock.RLock()
	defer i.mutexLock.RUnlock()

	return i.store[signedPreKeyID]
}

func (i *InMemorySignedPreKeyStore) LoadSignedPreKeys() []*record.SignedPreKey {
	i.mutexLock.RLock()
	defer i.mutexLock.RUnlock()

	var preKeys []*record.SignedPreKey
	for _, record := range i.store {
		preKeys = append(preKeys, record)
	}

	return preKeys
}

func (i *InMemorySignedPreKeyStore) StoreSignedPreKey(signedPreKeyID uint32, record *record.SignedPreKey) {
	i.mutexLock.Lock()
	i.store[signedPreKeyID] = record
	i.mutexLock.Unlock()
}

func (i *InMemorySignedPreKeyStore) ContainsSignedPreKey(signedPreKeyID uint32) bool {
	i.mutexLock.RLock()
	_, ok := i.store[signedPreKeyID]
	i.mutexLock.RUnlock()
	return ok
}

func (i *InMemorySignedPreKeyStore) RemoveSignedPreKey(signedPreKeyID uint32) {
	i.mutexLock.Lock()
	delete(i.store, signedPreKeyID)
	i.mutexLock.Unlock()
}
