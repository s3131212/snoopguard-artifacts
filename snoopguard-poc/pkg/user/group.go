package user

import (
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/treekem"
	"context"
	"fmt"
	syntax "github.com/cisco/go-tls-syntax"
	"github.com/s3131212/go-mls"
	"go.mau.fi/libsignal/logger"
	"math/rand"
)

/*
CreateGroup initializes a new group, either server side or client side, registers the group to the server, and return the group id.
*/
func (csu *ClientSideUser) CreateGroup(groupType pb.GroupType) (string, error) {
	// Send group creation request to server
	res, err := csu.chatServiceClient.CreateGroup(context.Background(), &pb.CreateGroupRequest{
		InitiatorID: csu.userID,
		GroupType:   groupType,
	})
	if err != nil {
		logger.Error("Failed to create group: ", err)
		return "", err
	}

	if !res.GetSuccess() || res.GetErrorMessage() != "" {
		logger.Error("Failed to create group: ", res.GetErrorMessage())
		return "", fmt.Errorf(res.GetErrorMessage())

	}

	logger.Info("Created group: ", res.GetGroupID(), " with type: ", groupType)

	// Join the group
	initLeaf, err := treekem.GenerateRandomBytes(32)
	if err != nil {
		logger.Error("Failed to generate random bytes: ", err)
		return "", err
	}

	// Generate MLS key package
	// generate a random int for key package id
	keyPackageId := rand.Uint32()
	csu.Client.GenerateMLSKeyPackage(keyPackageId + 10000)

	csu.JoinGroup(res.GetGroupID(), groupType, []string{csu.userID}, []string{}, nil, nil, treekem.GroupInitKey{}, initLeaf, nil, nil, nil, nil, keyPackageId+10000)

	return res.GetGroupID(), nil
}

/*
JoinGroup joins a group, either server side or client side, and return the group id.
*/
func (csu *ClientSideUser) JoinGroup(groupID string, groupType pb.GroupType, participantIDs []string, chatbotIDs []string, chatbotIsIGA map[string]bool, chatbotIsPseudo map[string]bool, treekemGroupInitKey treekem.GroupInitKey, treekemInitLeaf []byte, chatbotPubKeys map[string][]byte, chatbotSignPubKeys map[string][]byte, lastTreeKemRootCiphertexts map[string]treekem.ECKEMCipherText, welcomeMessageSerialized []byte, keyPackageId uint32) {
	switch groupType {
	case pb.GroupType_SERVER_SIDE:
		// Check if already in the group
		_, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
		if err == nil {
			logger.Info("Already in the group: ", groupID)
			return
		}

		csu.Client.JoinGroup(groupID, groupType, participantIDs, chatbotIDs)

		_, err = csu.PostCreateServerSideGroup(groupID, chatbotIsIGA, chatbotIsPseudo, treekemGroupInitKey, treekemInitLeaf, chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts)
		if err != nil {
			logger.Error("Failed to listen to group: ", err)
			return
		}
	case pb.GroupType_CLIENT_SIDE:
		// Check if already in the group
		_, err := csu.Client.GetClientSideGroupSessionDriver(groupID)
		if err == nil {
			logger.Info("Already in the group: ", groupID)
			return
		}

		csu.Client.JoinGroup(groupID, groupType, participantIDs, chatbotIDs)

		_, err = csu.PostCreateClientSideGroup(groupID, chatbotIsIGA, treekemGroupInitKey, treekemInitLeaf, chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts)
		if err != nil {
			logger.Error("Failed to listen to group: ", err)
			return
		}
	case pb.GroupType_MLS:
		// Check if already in the group
		_, err := csu.Client.GetMlsGroupSessionDriver(groupID)
		if err == nil {
			logger.Info("Already in the group: ", groupID)
			return
		}

		// Deserialize Welcome
		welcome := mls.Welcome{}
		if welcomeMessageSerialized != nil {
			_, err = syntax.Unmarshal(welcomeMessageSerialized, &welcome)
		}

		csu.Client.JoinGroup(groupID, groupType, participantIDs, chatbotIDs)

		_, err = csu.PostCreateMlsGroup(groupID, chatbotIsIGA, chatbotIsPseudo, welcome, keyPackageId)
		if err != nil {
			logger.Error("Failed to listen to group: ", err)
			return
		}
	}

}

/*
AddUserToGroup adds a user to the group.
*/
func (csu *ClientSideUser) AddUserToGroup(groupID string, groupType pb.GroupType, senderID string, addedID string, participantIDs []string, treekemUserAdd treekem.UserAdd, mlsUserAddSerialized []byte, mlsCommitSerialized []byte) {
	logger.Info("Adding user: ", addedID, " to group: ", groupID)
	switch groupType {
	case pb.GroupType_SERVER_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs - addedID

		sessionDriver.UpdateGroupParticipantIDs(participantIDs)
		sessionDriver.AddUserToTreeKEM(&treekemUserAdd)
	case pb.GroupType_CLIENT_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetClientSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs - addedID

		sessionDriver.UpdateGroupParticipantIDs(participantIDs)
		sessionDriver.AddUserToTreeKEM(&treekemUserAdd)

	case pb.GroupType_MLS:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID)
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

		if senderID != csu.userID {
			err = sessionDriver.AddUser(&mlsUserAdd, &mlsCommit)
			if err != nil {
				logger.Error("Failed to add user to MLS group: ", err)
				return
			}
		}
		sessionDriver.UpdateGroupParticipantIDs(participantIDs)
	}
}

/*
RemoveUserFromGroup removes self from the group if id matches, otherwise user from the group's participant list.
*/
func (csu *ClientSideUser) RemoveUserFromGroup(groupID string, groupType pb.GroupType, removedID string, participantIDs []string, mlsRemoveSerialized []byte, mlsRemoveCommitSerialized []byte) {
	logger.Info("Removing user: ", removedID, " from group: ", groupID)
	switch groupType {
	case pb.GroupType_SERVER_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}
		if removedID == csu.userID {
			logger.Info("Leaving group: ", groupID)
			csu.Client.LeaveGroup(groupID, groupType)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs + removedID

		sessionDriver.RemoveUserSession(removedID)
		sessionDriver.UpdateGroupParticipantIDs(participantIDs)
	case pb.GroupType_CLIENT_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetClientSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}
		if removedID == csu.userID {
			logger.Info("Leaving group: ", groupID)
			csu.Client.LeaveGroup(groupID, groupType)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs + removedID

		sessionDriver.RemoveUserSession(removedID)
		sessionDriver.UpdateGroupParticipantIDs(participantIDs)

	case pb.GroupType_MLS:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID)
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

		if removedID == csu.userID {
			logger.Info("Leaving group: ", groupID)
			csu.Client.LeaveGroup(groupID, groupType)
			return
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
AddChatbotToGroup adds a user to the group.
*/
func (csu *ClientSideUser) AddChatbotToGroup(groupID string, groupType pb.GroupType, senderID string, addedChatbotID string, chatbotIDs []string, isIGA bool, isPseudo bool, chatbotCipherText treekem.ECKEMCipherText, mlsUserAddSerialized []byte, mlsCommitSerialized []byte) {
	logger.Info("Adding chatbot: ", addedChatbotID, " to group: ", groupID)
	switch groupType {
	case pb.GroupType_SERVER_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs - addedID

		sessionDriver.UpdateGroupChatbotIDs(chatbotIDs)
		sessionDriver.SetChatbotIsIGA(addedChatbotID, isIGA)
		sessionDriver.SetChatbotIsPseudo(addedChatbotID, isPseudo)

		// Add to MultiTreeKEM's external node
		if chatbotCipherText.CipherText != nil {
			err = sessionDriver.AddExternalNodeToMultiTreeKEM(addedChatbotID, chatbotCipherText)
			if err != nil {
				logger.Warning("Failed to add external node to MultiTreeKEM: ", err)
			}
		}
	case pb.GroupType_CLIENT_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetClientSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs - addedID

		sessionDriver.UpdateGroupChatbotIDs(chatbotIDs)

		// Add to MultiTreeKEM's external node
		if chatbotCipherText.CipherText != nil {
			err = sessionDriver.AddExternalNodeToMultiTreeKEM(addedChatbotID, chatbotCipherText)
			if err != nil {
				logger.Warning("Failed to add external node to MultiTreeKEM: ", err)
			}
		}
	case pb.GroupType_MLS:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs - addedID

		if senderID != csu.userID {
			// Add to MultiTreeKEM's external node
			if chatbotCipherText.CipherText != nil {
				//err = sessionDriver.GetMlsMultiTree().UpdateTreeKEMRootFromHash(sessionDriver.GetGroupState().Tree.RootHash())
				//if err != nil {
				//	logger.Error("Failed to update TreeKEMRootFromHash: ", err)
				//}
				err = sessionDriver.AddExternalNodeToMlsMultiTree(addedChatbotID, chatbotCipherText)
				if err != nil {
					logger.Error("Failed to add external node to MlsMultiTree: ", err)
				}
			}

			if mlsUserAddSerialized != nil && mlsCommitSerialized != nil {
				// Deserialize MLSUserAdd and MLSCommit
				mlsUserAdd := mls.MLSPlaintext{}
				mlsCommit := mls.MLSPlaintext{}
				_, err = syntax.Unmarshal(mlsUserAddSerialized, &mlsUserAdd)
				_, err = syntax.Unmarshal(mlsCommitSerialized, &mlsCommit)

				err = sessionDriver.AddUser(&mlsUserAdd, &mlsCommit)
				if err != nil {
					logger.Error("Failed to add user to MLS group: ", err)
					return
				}
			}
		}

		sessionDriver.UpdateGroupChatbotIDs(chatbotIDs)
		sessionDriver.SetChatbotIsIGA(addedChatbotID, isIGA)
		sessionDriver.SetChatbotIsPseudo(addedChatbotID, isPseudo)
	}
}

/*
RemoveChatbotFromGroup removes chatbot from the group's chatbot list.
*/
func (csu *ClientSideUser) RemoveChatbotFromGroup(groupID string, groupType pb.GroupType, removedChatbotID string, chatbotIDs []string) {
	logger.Info("Removing chatbot: ", removedChatbotID, " from group: ", groupID)
	switch groupType {
	case pb.GroupType_SERVER_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs + removedID

		sessionDriver.RemoveUserSession(removedChatbotID)
		sessionDriver.UpdateGroupChatbotIDs(chatbotIDs)
	case pb.GroupType_CLIENT_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetClientSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs + removedID

		sessionDriver.RemoveUserSession(removedChatbotID)
		sessionDriver.UpdateGroupChatbotIDs(chatbotIDs)
	case pb.GroupType_MLS:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return
		}

		// Todo: Assert that current sessionDriver.participantIDs = participantIDs + removedID

		sessionDriver.UpdateGroupChatbotIDs(chatbotIDs)
	}
}

/*
RequestInviteUserToGroup invites a group member to the group. This function both send request and add the member.
*/
func (csu *ClientSideUser) RequestInviteUserToGroup(groupID string, groupType pb.GroupType, invitedID string) {
	// For TreeKEM's UserAdd, GroupInitKey, and chatbots external join messages.
	var ua treekem.UserAdd
	var gik treekem.GroupInitKey
	var initLeaf []byte
	var chatbotPubKeys map[string][]byte
	var chatbotSignPubKeys map[string][]byte
	var lastTreeKemRootCiphertexts map[string]treekem.ECKEMCipherText

	// For MLS's welcome message
	var welcomeSerialized []byte
	var addSerialized []byte
	var addCommitSerialized []byte
	var keyPackageID uint32
	var keyPackage mls.KeyPackage

	if groupType != pb.GroupType_MLS {
		ua, gik, initLeaf, chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts = csu.TreeKEMUserAdd(groupID, groupType)
	} else {
		// Get MLS Welcome message
		sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID)
		if err != nil {
			logger.Error("Error getting MLS group session driver: ", err)
			panic("")
		}

		// Get key package
		keyPackage, keyPackageID, err = csu.Client.GetOthersMLSKeyPackage(invitedID)
		if err != nil {
			logger.Error("Error getting MLS key package: ", err)
			panic("")
		}
		welcome, add, addCommit, err := sessionDriver.GetWelcomeMessage(keyPackage)
		if err != nil {
			logger.Error("Error getting welcome message: ", err)
			panic("")
		}
		welcomeSerialized, err = syntax.Marshal(welcome)
		addSerialized, err = syntax.Marshal(add)
		addCommitSerialized, err = syntax.Marshal(addCommit)
	}

	// Send group invitation request to server
	res, err := csu.chatServiceClient.InviteMember(csu.chatServiceClientCtx, &pb.InviteMemberRequest{
		GroupID:                    groupID,
		InitiatorID:                csu.userID,
		InvitedID:                  invitedID,
		TreeKEMUserAdd:             treekem.TreeKEMUserAddPbConvert(ua),
		TreeKEMGroupInitKey:        treekem.TreeKEMGroupInitKeyPbConvert(gik),
		TreeKEMInitLeaf:            initLeaf,
		ChatbotPubKeys:             chatbotPubKeys,
		ChatbotSignPubKeys:         chatbotSignPubKeys,
		LastTreeKemRootCiphertexts: treekem.ECKEMCipherTextStringMapPbConvert(lastTreeKemRootCiphertexts),
		MlsWelcomeMessage:          welcomeSerialized,
		MlsUserAdd:                 addSerialized,
		MlsAddCommit:               addCommitSerialized,
		MlsKeyPackageID:            keyPackageID,
	})

	if err != nil || res.ErrorMessage != "" {
		logger.Error("Error inviting member to group: ", err)
		panic("")
	}

	logger.Info("Received response for group invitation: ", res.String())
}

/*
RequestRemoveUserFromGroup requests to remove a group member from the group. This function only send request but does not remove the member.
*/
func (csu *ClientSideUser) RequestRemoveUserFromGroup(groupID string, removedID string) {
	// Generate Remove for MLS group if needed
	var removeSerialized []byte
	var removeCommitSerialized []byte
	if sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID); err == nil {
		remove, removeCommit, err := sessionDriver.GetRemoveMessage(removedID)
		if err != nil {
			logger.Error("Error getting remove message: ", err)
			panic("")
		}
		removeSerialized, err = syntax.Marshal(remove)
		removeCommitSerialized, err = syntax.Marshal(removeCommit)
	}

	// Send group removal request to server
	res, err := csu.chatServiceClient.RemoveMember(csu.chatServiceClientCtx, &pb.RemoveMemberRequest{
		GroupID:         groupID,
		InitiatorID:     csu.userID,
		RemovedID:       removedID,
		MlsRemove:       removeSerialized,
		MlsRemoveCommit: removeCommitSerialized,
	})

	if err != nil || res.ErrorMessage != "" {
		logger.Error("Error removing member from group: ", err)
		panic("")
	}

	logger.Info("Received response for group removal: ", res.String())
}

/*
RequestInviteChatbotToGroup invites a chatbot to the group. This function only send request but does not add the chatbot.
*/
func (csu *ClientSideUser) RequestInviteChatbotToGroup(groupID string, groupType pb.GroupType, invitedID string, isIGA bool, isPseudo bool) {

	// For TreeKEM
	var chatbotCipherText treekem.ECKEMCipherText
	var initLeaf []byte
	var err error

	// For MLS's welcome message
	var welcomeSerialized []byte
	var addSerialized []byte
	var addCommitSerialized []byte
	var keyPackageID uint32
	var keyPackage mls.KeyPackage

	if groupType != pb.GroupType_MLS || isIGA {
		chatbotCipherText, initLeaf, err = csu.TreeKEMChatbotAdd(groupID, groupType, invitedID)
		if err != nil {
			logger.Error("Error adding chatbot to group: ", err)
			panic("")
		}
	}
	if groupType == pb.GroupType_MLS && !isIGA {
		// Get MLS Welcome message
		sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID)
		if err != nil {
			logger.Error("Error getting MLS group session driver: ", err)
			panic("")
		}

		// Get key package
		keyPackage, keyPackageID, err = csu.Client.GetOthersMLSKeyPackage(invitedID)
		if err != nil {
			logger.Error("Error getting MLS key package: ", err)
			panic("")
		}
		welcome, add, addCommit, err := sessionDriver.GetWelcomeMessage(keyPackage)
		if err != nil {
			logger.Error("Error getting welcome message: ", err)
			panic("")
		}
		welcomeSerialized, err = syntax.Marshal(welcome)
		addSerialized, err = syntax.Marshal(add)
		addCommitSerialized, err = syntax.Marshal(addCommit)
	}

	// Send group invitation request to server
	rootPub, rootSignPub := csu.GetTreeKEMRootPublicKey(groupID, groupType)
	res, err := csu.chatServiceClient.InviteChatbot(csu.chatServiceClientCtx, &pb.InviteChatbotRequest{
		GroupID:            groupID,
		InitiatorID:        csu.userID,
		InvitedID:          invitedID,
		IsIGA:              isIGA,
		IsPseudo:           isPseudo,
		TreekemRootPub:     rootPub,
		TreekemRootSignPub: rootSignPub,
		ChatbotInitLeaf:    initLeaf,
		ChatbotCipherText:  treekem.ECKEMCipherTextPbConvert(&chatbotCipherText),
		MlsWelcomeMessage:  welcomeSerialized,
		MlsUserAdd:         addSerialized,
		MlsAddCommit:       addCommitSerialized,
		MlsKeyPackageID:    keyPackageID,
	})

	if err != nil || res.ErrorMessage != "" {
		logger.Error("Error inviting chatbot to group: ", err)
		panic("")
	}

	logger.Info("Received response for group invitation: ", res.String())
}

/*
RequestRemoveChatbotFromGroup requests to remove a chatbot from the group. This function only send request but does not remove the chatbot.
*/
func (csu *ClientSideUser) RequestRemoveChatbotFromGroup(groupID string, removedID string) {
	// Send group removal request to server
	res, err := csu.chatServiceClient.RemoveChatbot(csu.chatServiceClientCtx, &pb.RemoveChatbotRequest{
		GroupID:     groupID,
		InitiatorID: csu.userID,
		RemovedID:   removedID,
	})

	if err != nil || res.ErrorMessage != "" {
		logger.Error("Error removing chatbot from group: ", err)
		panic("")
	}

	logger.Info("Received response for group removal: ", res.String())
}

/*
TreeKEMUserAdd add a new member to the treekem of the group and returns UserAdd, GroupInitKey, and chatbots external join messages.
*/
func (csu *ClientSideUser) TreeKEMUserAdd(groupID string, groupType pb.GroupType) (treekem.UserAdd, treekem.GroupInitKey, []byte, map[string][]byte, map[string][]byte, map[string]treekem.ECKEMCipherText) {
	switch groupType {
	case pb.GroupType_SERVER_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return treekem.UserAdd{}, treekem.GroupInitKey{}, nil, nil, nil, nil
		}

		leaf, err := treekem.GenerateRandomBytes(32)
		if err != nil {
			logger.Error("TreeKEMUserAdd failed:", err)
			return treekem.UserAdd{}, treekem.GroupInitKey{}, nil, nil, nil, nil
		}

		gik := sessionDriver.GetTreeKEMState().GroupInitKey()
		ua, err := treekem.TreeKEMStateJoin(leaf, gik)
		if err != nil {
			logger.Error("TreeKEMUserAdd failed:", err)
			return treekem.UserAdd{}, treekem.GroupInitKey{}, nil, nil, nil, nil
		}

		// Get chatbot external join messages.
		// workaround: ua.Nodes[(ua.Size-1)*2].Public is the public key of the newly added node
		chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts, err := sessionDriver.GenerateExternalNodeJoinsWithoutUpdate(ua.Nodes[(ua.Size-1)*2].Public)
		if err != nil {
			logger.Error("TreeKEMUserAdd failed:", err)
			return treekem.UserAdd{}, treekem.GroupInitKey{}, nil, nil, nil, nil
		}

		return ua, gik, leaf, chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts
	case pb.GroupType_CLIENT_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetClientSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return treekem.UserAdd{}, treekem.GroupInitKey{}, nil, nil, nil, nil
		}

		leaf, err := treekem.GenerateRandomBytes(32)
		if err != nil {
			logger.Error("TreeKEMUserAdd failed:", err)
			return treekem.UserAdd{}, treekem.GroupInitKey{}, nil, nil, nil, nil
		}

		gik := sessionDriver.GetTreeKEMState().GroupInitKey()
		ua, err := treekem.TreeKEMStateJoin(leaf, gik)
		if err != nil {
			logger.Error("TreeKEMUserAdd failed:", err)
			return treekem.UserAdd{}, treekem.GroupInitKey{}, nil, nil, nil, nil
		}

		// Get chatbot external join messages.
		// workaround: ua.Nodes[(ua.Size-1)*2].Public is the public key of the newly added node
		chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts, err := sessionDriver.GenerateExternalNodeJoinsWithoutUpdate(ua.Nodes[(ua.Size-1)*2].Public)
		if err != nil {
			logger.Error("TreeKEMUserAdd failed:", err)
			return treekem.UserAdd{}, treekem.GroupInitKey{}, nil, nil, nil, nil
		}

		return ua, gik, leaf, chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts
	}
	return treekem.UserAdd{}, treekem.GroupInitKey{}, nil, nil, nil, nil
}

/*
TreeKEMChatbotAdd add a new chatbot to the treekem of the group and returns initLeaf and chatbotCipherText.
*/
func (csu *ClientSideUser) TreeKEMChatbotAdd(groupID string, groupType pb.GroupType, chatbotId string) (treekem.ECKEMCipherText, []byte, error) {
	switch groupType {
	case pb.GroupType_SERVER_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return treekem.ECKEMCipherText{}, nil, err
		}

		return sessionDriver.GenerateExternalNodeJoin(chatbotId)
	case pb.GroupType_CLIENT_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetClientSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return treekem.ECKEMCipherText{}, nil, err
		}

		return sessionDriver.GenerateExternalNodeJoin(chatbotId)
	case pb.GroupType_MLS:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return treekem.ECKEMCipherText{}, nil, err
		}

		return sessionDriver.GenerateExternalNodeJoin(chatbotId)
	}
	return treekem.ECKEMCipherText{}, nil, nil
}

/*
GetTreeKEMRootPublicKey returns the root public key of the treekem of the group.
*/
func (csu *ClientSideUser) GetTreeKEMRootPublicKey(groupID string, groupType pb.GroupType) ([]byte, []byte) {
	switch groupType {
	case pb.GroupType_SERVER_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return nil, nil
		}

		return sessionDriver.GetTreeKEMState().RootPublic(), sessionDriver.GetTreeKEMState().RootSignPublic()
	case pb.GroupType_CLIENT_SIDE:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetClientSideGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return nil, nil
		}

		return sessionDriver.GetTreeKEMState().RootPublic(), sessionDriver.GetTreeKEMState().RootSignPublic()
	case pb.GroupType_MLS:
		// Check if already in the group
		sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID)
		if err != nil {
			logger.Info("Not in the group: ", groupID)
			return nil, nil
		}

		treekemRoot, err := sessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		if err != nil {
			logger.Warning("Failed to get TreeKEMRoot: ", err)
			return nil, nil
		}
		return treekemRoot.Public, treekemRoot.SignPublic
	}
	return nil, nil
}

/*
CreatePseudoUser creates a pseudonym for the user.
*/
func (csu *ClientSideUser) CreatePseudoUser(groupId string, groupType pb.GroupType, chatbotId string) (string, []byte, error) {
	pseudonymId := csu.userID + groupId + "-pseudonym"

	// Create pseudonym for the user.
	signSecret, err := treekem.GenerateRandomBytes(32)
	if err != nil {
		logger.Error("Failed to generate random bytes: ", err)
		return "", nil, err
	}
	pseudoSigningKp, err := treekem.NewSigningKeyPairFromSecret(signSecret)
	pseudoUser := &PseudoUser{
		PseudoUserID:   pseudonymId,
		SigningKeyPair: *pseudoSigningKp,
		SignSecret:     signSecret,
	}

	if _, exists := csu.pseudoUsers[groupId]; !exists {
		csu.pseudoUsers[groupId] = make(map[string]*PseudoUser)
	}
	csu.pseudoUsers[groupId][chatbotId] = pseudoUser

	return pseudonymId, pseudoSigningKp.Public.Bytes(), nil

}

/*
GetPseudoUser returns the pseudonym of the user in a given group for a given chatbot.
*/
func (csu *ClientSideUser) GetPseudoUser(groupId string, chatbotId string) *PseudoUser {
	if _, exists := csu.pseudoUsers[groupId]; !exists {
		return nil
	}
	return csu.pseudoUsers[groupId][chatbotId]
}

/*
RemovePseudoUser removes the pseudonym of the user.
*/
func (csu *ClientSideUser) RemovePseudoUser(groupId string) {
	delete(csu.pseudoUsers, groupId)
}
