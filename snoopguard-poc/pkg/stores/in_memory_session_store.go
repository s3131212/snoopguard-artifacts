package stores

import (
	"go.mau.fi/libsignal/protocol"
	"go.mau.fi/libsignal/serialize"
	"go.mau.fi/libsignal/state/record"
	"sync"
)

func NewInMemorySessionStore(serializer *serialize.Serializer) *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions:   make(map[*protocol.SignalAddress]*record.Session),
		serializer: serializer,
	}
}

type InMemorySessionStore struct {
	sessions   map[*protocol.SignalAddress]*record.Session
	serializer *serialize.Serializer
	mutexLock  sync.RWMutex
}

func (i *InMemorySessionStore) LoadSession(address *protocol.SignalAddress) *record.Session {
	if i.ContainsSession(address) {
		i.mutexLock.RLock()
		result := i.sessions[address]
		i.mutexLock.RUnlock()
		return result
	}
	i.mutexLock.Lock()
	sessionRecord := record.NewSession(i.serializer.Session, i.serializer.State)
	i.sessions[address] = sessionRecord
	i.mutexLock.Unlock()

	return sessionRecord
}

func (i *InMemorySessionStore) GetSubDeviceSessions(name string) []uint32 {
	i.mutexLock.RLock()
	var deviceIDs []uint32

	for key := range i.sessions {
		if key.Name() == name && key.DeviceID() != 1 {
			deviceIDs = append(deviceIDs, key.DeviceID())
		}
	}

	i.mutexLock.RUnlock()
	return deviceIDs
}

func (i *InMemorySessionStore) StoreSession(remoteAddress *protocol.SignalAddress, record *record.Session) {
	i.mutexLock.Lock()
	i.sessions[remoteAddress] = record
	i.mutexLock.Unlock()
}

func (i *InMemorySessionStore) ContainsSession(remoteAddress *protocol.SignalAddress) bool {
	i.mutexLock.RLock()
	_, ok := i.sessions[remoteAddress]
	i.mutexLock.RUnlock()
	return ok
}

func (i *InMemorySessionStore) DeleteSession(remoteAddress *protocol.SignalAddress) {
	i.mutexLock.Lock()
	delete(i.sessions, remoteAddress)
	i.mutexLock.Unlock()
}

func (i *InMemorySessionStore) DeleteAllSessions() {
	i.mutexLock.Lock()
	i.sessions = make(map[*protocol.SignalAddress]*record.Session)
	i.mutexLock.Unlock()
}
