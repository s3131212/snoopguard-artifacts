package util

import "go.mau.fi/libsignal/logger"

type GroupChatClientSideFanout struct {
	groupID  string
	selfUser *User

	participantIdToSession map[string]*SessionWrapper
}

func NewGroupChatClientSideFanout(selfUser *User, groupID string) *GroupChatClientSideFanout {
	gccsf := &GroupChatClientSideFanout{}
	gccsf.groupID = groupID
	gccsf.selfUser = selfUser

	return gccsf
}

func (gccsf *GroupChatClientSideFanout) setSession(participantId string, session *SessionWrapper) {
	gccsf.participantIdToSession[participantId] = session
}

func (gccsf *GroupChatClientSideFanout) getSession(participantId string) *SessionWrapper {
	session, exist := gccsf.participantIdToSession[participantId]

	if !exist {
		logger.Error("Session not created: ", participantId)
		panic("")
	}

	return session
}
