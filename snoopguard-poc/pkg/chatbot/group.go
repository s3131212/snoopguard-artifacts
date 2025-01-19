package chatbot

import (
	pb "chatbot-poc-go/pkg/protos/services"
	syntax "github.com/cisco/go-tls-syntax"
	"github.com/s3131212/go-mls"
	"go.mau.fi/libsignal/logger"
)

/*
JoinGroup joins a group, either server side or client side, and return the group id.
*/
func (csc *ClientSideChatbot) JoinGroup(groupID string, groupType pb.GroupType, participantIDs []string, isIGA bool, isPseudo bool, treekemRootPub []byte, treekemRootSignPub []byte, initLeaf []byte, welcomeMessageSerialized []byte, keyPackageId uint32) {
	logger.Info("Joining group: ", groupID, " with type: ", groupType, " and participant IDs: ", participantIDs)
	switch groupType {
	case pb.GroupType_SERVER_SIDE:
		// Check if already in the group
		_, err := csc.Client.GetServerSideGroupSessionDriver(groupID)
		if err == nil {
			logger.Info("Already in the group: ", groupID)
			return
		}

		csc.Client.JoinGroup(groupID, groupType, participantIDs, nil)

		_, err = csc.PostCreateServerSideGroup(groupID, isIGA, isPseudo, treekemRootPub, treekemRootSignPub, initLeaf)
		if err != nil {
			logger.Error("Failed to listen to group: ", err)
			return
		}
	case pb.GroupType_CLIENT_SIDE:
		// Check if already in the group
		_, err := csc.Client.GetClientSideGroupSessionDriver(groupID)
		if err == nil {
			logger.Info("Already in the group: ", groupID)
			return
		}

		csc.Client.JoinGroup(groupID, groupType, participantIDs, nil)

		_, err = csc.PostCreateClientSideGroup(groupID, treekemRootPub, treekemRootSignPub, initLeaf)
		if err != nil {
			logger.Error("Failed to listen to group: ", err)
			return
		}
	case pb.GroupType_MLS:
		// Check if already in the group
		_, err := csc.Client.GetMlsGroupSessionDriver(groupID)
		if err == nil {
			logger.Info("Already in the group: ", groupID)
			return
		}

		// Deserialize Welcome
		welcome := mls.Welcome{}
		if welcomeMessageSerialized != nil {
			_, err = syntax.Unmarshal(welcomeMessageSerialized, &welcome)
		}

		csc.Client.JoinGroup(groupID, groupType, participantIDs, nil)

		_, err = csc.PostCreateMlsGroup(groupID, isIGA, isPseudo, treekemRootPub, treekemRootSignPub, initLeaf, welcome, keyPackageId)
		if err != nil {
			logger.Error("Failed to listen to group: ", err)
			return
		}
	}

}

/*
AddUserToGroup adds a user to the group.
*/
func (csc *ClientSideChatbot) AddUserToGroup(groupID string, groupType pb.GroupType, addedID string, participantIDs []string, mlsUserAddSerialized []byte, mlsCommitSerialized []byte) {
	logger.Info("Adding user: ", addedID, " to group: ", groupID)
	switch groupType {
	case pb.GroupType_SERVER_SIDE:
		// Check if already in the group
		sessionDriver, err := csc.Client.GetServerSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs - addedID

		sessionDriver.UpdateGroupParticipantIDs(participantIDs)
		//err = csc.DistributeSelfSenderKeyToUserID(addedID, groupID)
		if err != nil {
			logger.Error(err)
		}
	case pb.GroupType_CLIENT_SIDE:
		// Check if already in the group
		sessionDriver, err := csc.Client.GetClientSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs - addedID

		sessionDriver.UpdateGroupParticipantIDs(participantIDs)
	case pb.GroupType_MLS:
		// Check if already in the group
		sessionDriver, err := csc.Client.GetMlsGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs - addedID

		// Deserialize MLSUserAdd and MLSCommit
		mlsUserAdd := mls.MLSPlaintext{}
		mlsCommit := mls.MLSPlaintext{}
		if mlsUserAddSerialized != nil {
			_, err = syntax.Unmarshal(mlsUserAddSerialized, &mlsUserAdd)
		}
		if mlsCommitSerialized != nil {
			_, err = syntax.Unmarshal(mlsCommitSerialized, &mlsCommit)
		}

		err = sessionDriver.AddUser(&mlsUserAdd, &mlsCommit)
		if err != nil {
			logger.Error("Failed to add user to MLS group: ", err)
			return
		}
		sessionDriver.UpdateGroupParticipantIDs(participantIDs)
	}
}

/*
AddChatbotToGroup adds the chatbot to the group by calling AddUserToGroup
*/
func (csc *ClientSideChatbot) AddChatbotToGroup(groupID string, groupType pb.GroupType, chatbotID string, mlsUserAddSerialized []byte, mlsCommitSerialized []byte) {
	if groupType != pb.GroupType_MLS {
		logger.Error("Chatbot can only be added to MLS group")
		return
	}

	// New participant IDs = participantIDs + chatbotID
	sessionDriver, err := csc.Client.GetMlsGroupSessionDriver(groupID)
	if err != nil {
		logger.Error("Not in the group: ", groupID)
		return
	}

	participantIDs := append(sessionDriver.GetGroupParticipants(), chatbotID)
	csc.AddUserToGroup(groupID, groupType, chatbotID, participantIDs, mlsUserAddSerialized, mlsCommitSerialized)
}

/*
RemoveUserFromGroup removes user from the group's participant list.
*/
func (csc *ClientSideChatbot) RemoveUserFromGroup(groupID string, groupType pb.GroupType, removedID string, participantIDs []string, mlsRemoveSerialized []byte, mlsRemoveCommitSerialized []byte) {
	logger.Info("Removing user: ", removedID, " from group: ", groupID)
	switch groupType {
	case pb.GroupType_SERVER_SIDE:
		// Check if already in the group
		sessionDriver, err := csc.Client.GetServerSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs + removedID

		sessionDriver.RemoveUserSession(removedID)
		sessionDriver.UpdateGroupParticipantIDs(participantIDs)
	case pb.GroupType_CLIENT_SIDE:
		// Check if already in the group
		sessionDriver, err := csc.Client.GetClientSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs + removedID

		sessionDriver.RemoveUserSession(removedID)
		sessionDriver.UpdateGroupParticipantIDs(participantIDs)
	case pb.GroupType_MLS:
		// Check if already in the group
		sessionDriver, err := csc.Client.GetMlsGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Deserialize MLSRemove and MLSRemoveCommit
		mlsRemove := mls.MLSPlaintext{}
		mlsRemoveCommit := mls.MLSPlaintext{}
		if mlsRemoveSerialized != nil {
			_, err = syntax.Unmarshal(mlsRemoveSerialized, &mlsRemove)
		}
		if mlsRemoveCommitSerialized != nil {
			_, err = syntax.Unmarshal(mlsRemoveCommitSerialized, &mlsRemoveCommit)
		}

		err = sessionDriver.RemoveUser(&mlsRemove, &mlsRemoveCommit)
		if err != nil {
			logger.Error("Failed to remove user from MLS group: ", err)
			return
		}
		sessionDriver.UpdateGroupParticipantIDs(participantIDs)
	}
}

/*
LeaveGroup leaves a group.
*/
func (csc *ClientSideChatbot) LeaveGroup(groupID string, groupType pb.GroupType) {
	logger.Info("Leaving group: ", groupID)
	csc.Client.LeaveGroup(groupID, groupType)
}
