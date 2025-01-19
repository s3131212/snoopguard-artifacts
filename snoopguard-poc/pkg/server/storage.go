package server

import (
	pb "chatbot-poc-go/pkg/protos/services"
	"go.mau.fi/libsignal/keys/prekey"
	"sync"
)

var mutexLock sync.Mutex

type Storage struct {
	users  map[string]*ServerSideUser
	groups map[string]*ServerSideGroup
}

// NewStorage creates a new storage.
func NewStorage() *Storage {
	mutexLock = sync.Mutex{}
	return &Storage{
		users:  make(map[string]*ServerSideUser),
		groups: make(map[string]*ServerSideGroup),
	}
}

// AddUser adds a user to the storage.
func (s *Storage) AddUser(userID string) {
	s.users[userID] = NewServerSideUser()
}

// GetUser gets the user by the userID.
func (s *Storage) GetUser(userID string) *ServerSideUser {
	return s.users[userID]
}

// ContainUser check if a user is presented.
func (s *Storage) ContainUser(userID string) bool {
	_, ok := s.users[userID]
	return ok
}

// AddChatbot adds a chatbot to the storage.
func (s *Storage) AddChatbot(chatbotID string) {
	s.users[chatbotID] = NewServerSideChatbot()
}

// GetChatbot gets the chatbot by the chatbotID.
func (s *Storage) GetChatbot(chatbotID string) *ServerSideUser {
	chatbot, ok := s.users[chatbotID]
	if !ok || !chatbot.isChatbot {
		return nil
	}
	return chatbot
}

// ContainChatbot check if a chatbot is presented.
func (s *Storage) ContainChatbot(chatbotID string) bool {
	chatbot, ok := s.users[chatbotID]
	return ok && chatbot.isChatbot
}

// AddGroup adds a group to the storage.
func (s *Storage) AddGroup(groupID string, groupType int) {
	s.groups[groupID] = NewServerSideGroup(groupID, groupType)
}

// ContainGroup check if a group is presented.
func (s *Storage) ContainGroup(groupID string) bool {
	_, ok := s.groups[groupID]
	return ok
}

// GetGroup gets the group by the groupID.
func (s *Storage) GetGroup(groupID string) *ServerSideGroup {
	return s.groups[groupID]
}

type ServerSideUser struct {
	preKeyBundles              []*prekey.Bundle
	serializedPreKeys          map[uint32][]byte
	serializedSignedPreKeys    map[uint32][]byte
	serializedSignedPreKeySigs map[uint32][]byte
	serializedMlsKeyPackages   map[uint32][]byte
	identityKey                []byte
	registrationID             uint32
	messageQueue               chan *pb.MessageWrapper
	eventQueue                 chan *pb.ServerEvent
	isChatbot                  bool
}

// NewServerSideUser creates a new ServerSideUser.
func NewServerSideUser() *ServerSideUser {
	return &ServerSideUser{
		preKeyBundles:              make([]*prekey.Bundle, 0),
		serializedPreKeys:          make(map[uint32][]byte),
		serializedSignedPreKeys:    make(map[uint32][]byte),
		serializedSignedPreKeySigs: make(map[uint32][]byte),
		serializedMlsKeyPackages:   make(map[uint32][]byte),
		messageQueue:               make(chan *pb.MessageWrapper, 10000),
		eventQueue:                 make(chan *pb.ServerEvent, 10000),
		isChatbot:                  false,
	}
}

// NewServerSideChatbot creates a new ServerSideUser designed to be a chatbot.
func NewServerSideChatbot() *ServerSideUser {
	return &ServerSideUser{
		preKeyBundles:              make([]*prekey.Bundle, 0),
		serializedPreKeys:          make(map[uint32][]byte),
		serializedSignedPreKeys:    make(map[uint32][]byte),
		serializedSignedPreKeySigs: make(map[uint32][]byte),
		serializedMlsKeyPackages:   make(map[uint32][]byte),
		messageQueue:               make(chan *pb.MessageWrapper, 10000),
		eventQueue:                 make(chan *pb.ServerEvent, 10000),
		isChatbot:                  true,
	}
}

// AddSerializedPreKey adds a preKey to the ServerSideUser.
func (s *ServerSideUser) AddSerializedPreKey(serializedPreKey []byte, preKeyID uint32) {
	mutexLock.Lock()
	s.serializedPreKeys[preKeyID] = serializedPreKey
	mutexLock.Unlock()
}

// GetSerializedPreKey choose a preKey from the ServerSideUser.
func (s *ServerSideUser) GetSerializedPreKey() ([]byte, uint32) {
	mutexLock.Lock()
	var serializedPreKeyIdTemp uint32
	var serializedPreKeyTemp []byte
	for id, serializedPreKey := range s.serializedPreKeys {
		serializedPreKeyIdTemp = id
		serializedPreKeyTemp = serializedPreKey
		break
	}

	delete(s.serializedPreKeys, serializedPreKeyIdTemp) // remove it from serializedPreKeys
	mutexLock.Unlock()
	return serializedPreKeyTemp, serializedPreKeyIdTemp
}

// SetSerializedSignedPreKey add a signedPreKey to the ServerSideUser.
func (s *ServerSideUser) SetSerializedSignedPreKey(signedPreKey []byte, signedPreKeySig []byte, signedPreKeyID uint32) {
	mutexLock.Lock()
	s.serializedSignedPreKeys[signedPreKeyID] = signedPreKey
	s.serializedSignedPreKeySigs[signedPreKeyID] = signedPreKeySig
	mutexLock.Unlock()
}

// GetSerializedSignedPreKey gets the signedPreKey from the ServerSideUser.
func (s *ServerSideUser) GetSerializedSignedPreKey() ([]byte, []byte, uint32) {
	mutexLock.Lock()
	for id, serializedSignedPreKey := range s.serializedSignedPreKeys {
		mutexLock.Unlock()
		return serializedSignedPreKey, s.serializedSignedPreKeySigs[id], id
	}
	mutexLock.Unlock()

	return nil, nil, 0
}

// SetSerializedMlsKeyPackage adds a mlsKeyPackage to the ServerSideUser.
func (s *ServerSideUser) SetSerializedMlsKeyPackage(mlsKeyPackage []byte, mlsKeyPackageID uint32) {
	mutexLock.Lock()
	s.serializedMlsKeyPackages[mlsKeyPackageID] = mlsKeyPackage
	mutexLock.Unlock()
}

// GetSerializedMlsKeyPackage gets the mlsKeyPackage from the ServerSideUser.
func (s *ServerSideUser) GetSerializedMlsKeyPackage() ([]byte, uint32) {
	mutexLock.Lock()
	var serializedMlsKeyPackageIdTemp uint32
	var serializedMlsKeyPackageTemp []byte
	for id, serializedMlsKeyPackage := range s.serializedMlsKeyPackages {
		serializedMlsKeyPackageIdTemp = id
		serializedMlsKeyPackageTemp = serializedMlsKeyPackage
		break
	}

	delete(s.serializedMlsKeyPackages, serializedMlsKeyPackageIdTemp) // remove it from serializedMlsKeyPackages
	mutexLock.Unlock()

	return serializedMlsKeyPackageTemp, serializedMlsKeyPackageIdTemp
}

// PushMessageToQueue pushes a message to the ServerSideUser's message queue.
func (s *ServerSideUser) PushMessageToQueue(message *pb.MessageWrapper) {
	s.messageQueue <- message
}

// PopMessageFromQueue pops a message from the ServerSideUser's message queue.
func (s *ServerSideUser) PopMessageFromQueue() *pb.MessageWrapper {
	return <-s.messageQueue
}

// PushServerEventToQueue pushes a server event to the ServerSideUser's event queue.
func (s *ServerSideUser) PushServerEventToQueue(event *pb.ServerEvent) {
	s.eventQueue <- event
}

// PopServerEventFromQueue pops a message from the ServerSideUser's event queue.
func (s *ServerSideUser) PopServerEventFromQueue() *pb.ServerEvent {
	return <-s.eventQueue
}

type ServerSideGroup struct {
	GroupID         string
	ParticipantIDs  []string
	ChatbotIDs      []string
	ChatbotIsIGA    map[string]bool
	ChatbotIsPseudo map[string]bool
	GroupType       int
}

// NewServerSideGroup creates a new ServerSideGroup.
func NewServerSideGroup(groupID string, groupType int) *ServerSideGroup {
	return &ServerSideGroup{
		GroupID:         groupID,
		GroupType:       groupType,
		ChatbotIsIGA:    make(map[string]bool),
		ChatbotIsPseudo: make(map[string]bool),
	}
}

// AddParticipantByID adds a participantID to the ServerSideGroup.
func (s *ServerSideGroup) AddParticipantByID(participantID string) {
	mutexLock.Lock()
	s.ParticipantIDs = append(s.ParticipantIDs, participantID)
	mutexLock.Unlock()
}

// RemoveParticipantByID removes a participantID from the ServerSideGroup.
func (s *ServerSideGroup) RemoveParticipantByID(participantID string) {
	mutexLock.Lock()
	for i, v := range s.ParticipantIDs {
		if v == participantID {
			s.ParticipantIDs = append(s.ParticipantIDs[:i], s.ParticipantIDs[i+1:]...)
			mutexLock.Unlock()
			return
		}
	}
	mutexLock.Unlock()
}

// GetParticipantIDs gets the participantIDs from the ServerSideGroup.
func (s *ServerSideGroup) GetParticipantIDs() []string {
	return s.ParticipantIDs
}

// AddChatbotByID adds a chatbotID to the ServerSideGroup.
func (s *ServerSideGroup) AddChatbotByID(chatbotID string, isIGA bool, isPseudo bool) {
	mutexLock.Lock()
	s.ChatbotIDs = append(s.ChatbotIDs, chatbotID)
	s.ChatbotIsIGA[chatbotID] = isIGA
	s.ChatbotIsPseudo[chatbotID] = isPseudo
	mutexLock.Unlock()
}

// RemoveChatbotByID removes a chatbotID from the ServerSideGroup.
func (s *ServerSideGroup) RemoveChatbotByID(chatbotID string) {
	mutexLock.Lock()
	for i, v := range s.ChatbotIDs {
		if v == chatbotID {
			s.ChatbotIDs = append(s.ChatbotIDs[:i], s.ChatbotIDs[i+1:]...)
			mutexLock.Unlock()
			return
		}
	}
	mutexLock.Unlock()
}

// GetChatbotIDs gets the chatbotIDs from the ServerSideGroup.
func (s *ServerSideGroup) GetChatbotIDs() []string {
	return s.ChatbotIDs
}

// GetChatbotIsIGA gets the chatbotIsIGA from the ServerSideGroup.
func (s *ServerSideGroup) GetChatbotIsIGA() map[string]bool {
	return s.ChatbotIsIGA
}

// GetChatbotIsPseudo gets the chatbotIsPseudo from the ServerSideGroup.
func (s *ServerSideGroup) GetChatbotIsPseudo() map[string]bool {
	return s.ChatbotIsPseudo
}
