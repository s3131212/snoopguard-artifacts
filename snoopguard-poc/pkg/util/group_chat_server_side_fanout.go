package util

import (
	"go.mau.fi/libsignal/groups"
	"go.mau.fi/libsignal/protocol"
)

type GroupChatServerSideFanout struct {
	senderKeyName       *protocol.SenderKeyName
	groupSessionBuilder *groups.SessionBuilder

	groupId  string
	selfUser *User

	participantIdToSession map[string]*GroupSessionWrapper
	sendingSession         *GroupSessionWrapper
}

func NewGroupChatServerSideFanout(selfUser *User, senderKeyName *protocol.SenderKeyName) *GroupChatServerSideFanout {
	gcssf := &GroupChatServerSideFanout{}
	gcssf.senderKeyName = senderKeyName
	gcssf.groupId = senderKeyName.GroupID()
	gcssf.selfUser = selfUser
	gcssf.participantIdToSession = make(map[string]*GroupSessionWrapper)

	selfUser.BuildGroupSession()

	return gcssf
}

func (gcssf *GroupChatServerSideFanout) CreateSendingGroupSession() *GroupSessionWrapper {
	if gcssf.sendingSession == nil {
		gcssf.sendingSession = NewGroupSessionWrapper(gcssf.selfUser, gcssf.senderKeyName, gcssf.selfUser.Serializer)
	}
	return gcssf.sendingSession
}

func (gcssf *GroupChatServerSideFanout) GetSendingGroupSession() *GroupSessionWrapper {
	return gcssf.sendingSession
}

func (gcssf *GroupChatServerSideFanout) CreateReceivingGroupSession(participantID string, senderKeyDistributionMessage *protocol.SenderKeyDistributionMessage) *GroupSessionWrapper {
	senderKeyName := protocol.NewSenderKeyName(gcssf.groupId, protocol.NewSignalAddress(participantID, 1))
	gcssf.participantIdToSession[participantID] = NewGroupSessionWrapper(gcssf.selfUser, senderKeyName, gcssf.selfUser.Serializer)
	gcssf.participantIdToSession[participantID].ProcessSenderKey(senderKeyName, senderKeyDistributionMessage)
	// protocol.NewSenderKeyName("test group", protocol.NewSignalAddress(participantID, 1))

	return gcssf.participantIdToSession[participantID]
}

func (gcssf *GroupChatServerSideFanout) GetReceivingGroupSession(participantID string) *GroupSessionWrapper {
	return gcssf.participantIdToSession[participantID]
}

func (gcssf *GroupChatServerSideFanout) RemoveReceivingGroupSession(participantID string) {
	delete(gcssf.participantIdToSession, participantID)
}
