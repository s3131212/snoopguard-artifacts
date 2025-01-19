package stores

import (
	"github.com/s3131212/go-mls"
	"sync"
)

func NewInMemoryKeyPackageStore() *InMemoryKeyPackageStore {
	return &InMemoryKeyPackageStore{
		store: make(map[uint32]mls.KeyPackage),
	}
}

type InMemoryKeyPackageStore struct {
	store     map[uint32]mls.KeyPackage
	mutexLock sync.RWMutex
}

func (i *InMemoryKeyPackageStore) LoadKeyPackage(keyPackageID uint32) mls.KeyPackage {
	i.mutexLock.RLock()
	result := i.store[keyPackageID]
	i.mutexLock.RUnlock()
	return result
}

func (i *InMemoryKeyPackageStore) StoreKeyPackage(keyPackageID uint32, keyPackage mls.KeyPackage) {
	i.mutexLock.Lock()
	i.store[keyPackageID] = keyPackage
	i.mutexLock.Unlock()
}

func (i *InMemoryKeyPackageStore) ContainsKeyPackage(keyPackageID uint32) bool {
	i.mutexLock.RLock()
	_, ok := i.store[keyPackageID]
	i.mutexLock.RUnlock()
	return ok
}

func (i *InMemoryKeyPackageStore) RemoveKeyPackage(keyPackageID uint32) {
	i.mutexLock.Lock()
	delete(i.store, keyPackageID)
	i.mutexLock.Unlock()
}
