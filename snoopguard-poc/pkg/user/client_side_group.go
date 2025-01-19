package user

import (
	"chatbot-poc-go/pkg/client"
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/treekem"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/libsignal/protocol"
	"google.golang.org/protobuf/proto"
)

/*
PostCreateClientSideGroup is called when a new client-side group is created.
*/
func (csu *ClientSideUser) PostCreateClientSideGroup(groupID string, chatbotIsIGA map[string]bool, treekemGroupInitKey treekem.GroupInitKey, treekemInitLeaf []byte, chatbotPubKeys map[string][]byte, chatbotSignPubKeys map[string][]byte, lastTreeKemRootCiphertexts map[string]treekem.ECKEMCipherText) (*client.ClientSideGroupSessionDriver, error) {
	sessionDriver, err := csu.Client.GetClientSideGroupSessionDriver(groupID)
	if err != nil {
		return nil, err
	}

	// Set up treekem
	err = sessionDriver.InitiateTreeKEM(treekemGroupInitKey, treekemInitLeaf)
	if err != nil {
		logger.Error("Failed to initiate TreeKEM: ", err)
		return nil, err
	}

	err = sessionDriver.InitiateMultiTreeKEM()
	if err != nil {
		logger.Error("Failed to initiate MultiTreeKEM: ", err)
		return nil, err
	}

	if chatbotPubKeys != nil && lastTreeKemRootCiphertexts != nil {
		err = sessionDriver.SetExternalNodeJoinsWithoutUpdate(chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts)
		if err != nil {
			logger.Error("Failed to set external node joins: ", err)
			return nil, err
		}
	}

	// Set Chatbot IGA status.
	for chatbotID, isIGA := range chatbotIsIGA {
		sessionDriver.SetChatbotIsIGA(chatbotID, isIGA)
	}

	return sessionDriver, nil
}

/*
SendClientSideGroupMessage sends a client-side group message to the group.
*/
func (csu *ClientSideUser) SendClientSideGroupMessage(groupID string, message []byte, messageType pb.MessageType, receivingChatbotIDs []string) error {
	messages, err := csu.GenerateClientSideGroupMessageCipherText(groupID, message, messageType, receivingChatbotIDs)
	if err != nil {
		logger.Error("Failed to generate client side group message: ", err)
		return err
	}
	return csu.Client.SendClientSideGroupMessage(groupID, messages)
}

/*
GenerateClientSideGroupMessageCipherText generates the client-side group message ciphertexts for all group participants.
*/
func (csu *ClientSideUser) GenerateClientSideGroupMessageCipherText(groupID string, message []byte, messageType pb.MessageType, receivingChatbotIDs []string) (map[string]*pb.MessageWrapper, error) {
	groupDriver, err := csu.Client.GetClientSideGroupSessionDriver(groupID)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	// Update TreeKEM
	userUpdate, chatbotUpdateCiphertexts, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err := groupDriver.GenerateMultiTreeKEMKeyUpdate(receivingChatbotIDs)

	// Create ClientSideGroupMessage
	packedCSGMsg := &pb.ClientSideGroupMessage{
		GroupID:     groupID,
		MessageType: messageType,
		Message:     message,
	}

	packedCSGMsgMarshalled, err := proto.Marshal(packedCSGMsg)
	if err != nil {
		logger.Error("Failed to encode client side group message", err)
		return nil, err
	}

	packedMessage := &pb.Message{
		Message:     packedCSGMsgMarshalled,
		MessageType: pb.MessageType_CLIENT_SIDE_GROUP_MESSAGE,
	}

	packedMessageMarshal, err := proto.Marshal(packedMessage)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	messages := make(map[string]*pb.MessageWrapper)

	for _, participantID := range groupDriver.GetGroupParticipants() {
		// Get user session
		sessionDriver, err := csu.Client.GetSessionDriver(protocol.NewSignalAddress(participantID, 1))
		if err != nil {
			logger.Info("Sending message to user without session ", participantID)
			sessionDriver, err = csu.CreateIndividualSession(protocol.NewSignalAddress(participantID, 1))
			if err != nil {
				logger.Error("Failed to create session with ", participantID, ": ", err)
				return nil, err
			}
		}

		encryptedMsg := sessionDriver.EncryptMessage(packedMessageMarshal)
		messageWrapper := &pb.MessageWrapper{
			SenderID:         csu.userID,
			RecipientID:      participantID,
			EncryptedMessage: encryptedMsg.Serialize(),
			HasPreKey:        encryptedMsg.Type() == protocol.PREKEY_TYPE,
			ChatbotIds:       receivingChatbotIDs,
			IsIGA:            false,
			TreeKEMKeyUpdatePack: &pb.TreeKEMKeyUpdatePack{
				UserUpdate:               treekem.TreeKEMUserUpdatePbConvert(userUpdate),
				ChatbotUpdateCiphertexts: treekem.ECKEMCipherTextStringMapPbConvert(chatbotUpdateCiphertexts),
				NewRootPubKey:            newTreeKemRootPubKey,
				NewRootSignPubKey:        newTreeKemRootSignPubKey,
			},
		}

		messages[participantID] = messageWrapper
	}

	for _, chatbotID := range receivingChatbotIDs {
		var encryptedMsg []byte
		var hasPreKey bool
		var senderID string

		//if isIGA {
		//	ct, err := util.Encrypt(packedMessageMarshal, groupDriver.GetMultiTreeKEM().GetRootSecret(chatbotID))
		//	if err != nil {
		//		logger.Error("Failed to encrypt message for chatbot ", chatbotID)
		//		return err
		//	}
		//	encryptedMsg = ct.Serialize()
		//	hasPreKey = false
		//	senderID = ""
		//} else {
		// Get user session
		sessionDriver, err := csu.Client.GetSessionDriver(protocol.NewSignalAddress(chatbotID, 1))
		if err != nil {
			logger.Info("Sending message to user without session ", chatbotID)
			sessionDriver, err = csu.CreateIndividualSession(protocol.NewSignalAddress(chatbotID, 1))
			if err != nil {
				logger.Error("Failed to create session with ", chatbotID, ": ", err)
				return nil, err
			}
		}
		encryptedCT := sessionDriver.EncryptMessage(packedMessageMarshal)
		encryptedMsg = encryptedCT.Serialize()
		hasPreKey = encryptedCT.Type() == protocol.PREKEY_TYPE
		senderID = csu.userID
		//}

		messageWrapper := &pb.MessageWrapper{
			SenderID:         senderID,
			RecipientID:      chatbotID,
			EncryptedMessage: encryptedMsg,
			HasPreKey:        hasPreKey,
			ChatbotIds:       receivingChatbotIDs,
			IsIGA:            false,
			TreeKEMKeyUpdatePack: &pb.TreeKEMKeyUpdatePack{
				UserUpdate:               treekem.TreeKEMUserUpdatePbConvert(userUpdate),
				ChatbotUpdateCiphertexts: treekem.ECKEMCipherTextStringMapPbConvert(chatbotUpdateCiphertexts),
				NewRootPubKey:            newTreeKemRootPubKey,
				NewRootSignPubKey:        newTreeKemRootSignPubKey,
			},
		}

		messages[chatbotID] = messageWrapper
	}

	return messages, nil
}

func (csu *ClientSideUser) HandleClientSideGroupMessage(message []byte, senderID string, chatbotIds []string, treeKEMKeyUpdatePack *pb.TreeKEMKeyUpdatePack, chatbotKeyUpdatePack *pb.MultiTreeKEMExternalKeyUpdatePack) ([]byte, pb.MessageType) {
	groupId, groupMessage, groupMessageType := csu.Client.ParseClientSideGroupMessage(message, senderID)

	// Handle treekem key update
	sessionDriver, err := csu.Client.GetClientSideGroupSessionDriver(groupId)
	if err != nil {
		logger.Error("Not in the group: ", groupId)
	}

	if treeKEMKeyUpdatePack != nil {
		logger.Info("Received TreeKEM update from ", senderID, " for client-side group ", groupId)

		userUpdate := treekem.PbTreeKEMUserUpdateConvert(treeKEMKeyUpdatePack.GetUserUpdate())
		err = sessionDriver.UpdateTreeKEMUserKey(&userUpdate, chatbotIds)
		if err != nil {
			logger.Error("UpdateTreeKEMUserKey failed: ", groupId)
		}
	}

	if chatbotKeyUpdatePack != nil {
		logger.Info("Received MultiTreeKEM update from ", senderID, " for client-side group ", groupId)

		chatbotUpdate := treekem.PbECKEMCipherTextConvert(chatbotKeyUpdatePack.GetChatbotUpdate())
		newCbPubKey := chatbotKeyUpdatePack.GetNewCbPubKey()
		newCbSignPubKey := chatbotKeyUpdatePack.GetNewCbSignPubKey()
		err = sessionDriver.HandleMultiTreeKEMExternalKeyUpdate(senderID, chatbotUpdate, newCbPubKey, newCbSignPubKey)
		if err != nil {
			logger.Error("UpdateMultiTreeKEMExternalKey failed: ", groupId, err)
		}
	}

	return groupMessage, groupMessageType
}
