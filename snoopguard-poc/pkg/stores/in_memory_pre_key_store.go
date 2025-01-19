package stores

import (
	"go.mau.fi/libsignal/state/record"
	"sync"
)

func NewInMemoryPreKeyStore() *InMemoryPreKeyStore {
	return &InMemoryPreKeyStore{
		store: make(map[uint32]*record.PreKey),
	}
}

type InMemoryPreKeyStore struct {
	store     map[uint32]*record.PreKey
	mutexLock sync.RWMutex
}

func (i *InMemoryPreKeyStore) LoadPreKey(preKeyID uint32) *record.PreKey {
	i.mutexLock.RLock()
	result := i.store[preKeyID]
	i.mutexLock.RUnlock()
	return result
}

func (i *InMemoryPreKeyStore) StorePreKey(preKeyID uint32, preKeyRecord *record.PreKey) {
	i.mutexLock.Lock()
	i.store[preKeyID] = preKeyRecord
	i.mutexLock.Unlock()
}

func (i *InMemoryPreKeyStore) ContainsPreKey(preKeyID uint32) bool {
	i.mutexLock.RLock()
	_, ok := i.store[preKeyID]
	i.mutexLock.RUnlock()
	return ok
}

func (i *InMemoryPreKeyStore) RemovePreKey(preKeyID uint32) {
	i.mutexLock.Lock()
	delete(i.store, preKeyID)
	i.mutexLock.Unlock()
}
