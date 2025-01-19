package stores

import (
	groupRecord "go.mau.fi/libsignal/groups/state/record"
	"go.mau.fi/libsignal/protocol"
	"sync"
)

func NewInMemorySenderKeyStore() *InMemorySenderKeyStore {
	return &InMemorySenderKeyStore{
		store: make(map[*protocol.SenderKeyName]*groupRecord.SenderKey),
	}
}

type InMemorySenderKeyStore struct {
	store     map[*protocol.SenderKeyName]*groupRecord.SenderKey
	mutexLock sync.RWMutex
}

func (i *InMemorySenderKeyStore) StoreSenderKey(senderKeyName *protocol.SenderKeyName, keyRecord *groupRecord.SenderKey) {
	i.mutexLock.Lock()
	i.store[senderKeyName] = keyRecord
	i.mutexLock.Unlock()
}

func (i *InMemorySenderKeyStore) LoadSenderKey(senderKeyName *protocol.SenderKeyName) *groupRecord.SenderKey {
	i.mutexLock.RLock()
	result := i.store[senderKeyName]
	i.mutexLock.RUnlock()
	return result
}
