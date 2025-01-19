package chatbot

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
func (csc *ClientSideChatbot) PostCreateClientSideGroup(groupID string, treekemRootPub []byte, treekemRootSignPub []byte, initLeaf []byte) (*client.ClientSideGroupSessionDriver, error) {
	sessionDriver, err := csc.Client.GetClientSideGroupSessionDriver(groupID)
	if err != nil {
		return nil, err
	}

	// Set up MultiTreeKEMExternal
	err = sessionDriver.InitiateMultiTreeKEMExternal(treekemRootPub, treekemRootSignPub, initLeaf)
	if err != nil {
		logger.Error("Failed to initiate MultiTreeKEMExternal: ", err)
		return nil, err
	}

	return sessionDriver, nil
}

func (csc *ClientSideChatbot) SendClientSideGroupMessage(groupID string, message []byte, messageType pb.MessageType) error {
	groupDriver, err := csc.Client.GetClientSideGroupSessionDriver(groupID)
	if err != nil {
		logger.Error(err)
		return err
	}

	// Update TreeKEM
	chatbotUpdate, newCbPubKey, newCbSignPubKey, err := groupDriver.GenerateMultiTreeKEMExternalKeyUpdate()
	if err != nil {
		logger.Error(err)
		return err
	}

	messages := make(map[string]*pb.MessageWrapper)
	receivingIDs := groupDriver.GetGroupParticipants()
	for _, participantID := range receivingIDs {
		// Create ClientSideGroupMessage
		packedCSGMsg := &pb.ClientSideGroupMessage{
			GroupID:     groupID,
			MessageType: messageType,
			Message:     message,
		}

		packedCSGMsgMarshalled, err := proto.Marshal(packedCSGMsg)
		if err != nil {
			logger.Error("Failed to encode client side group message", err)
			return err
		}

		packedMessage := &pb.Message{
			Message:     packedCSGMsgMarshalled,
			MessageType: pb.MessageType_CLIENT_SIDE_GROUP_MESSAGE,
		}

		packedMessageMarshal, err := proto.Marshal(packedMessage)
		if err != nil {
			logger.Error(err)
			return err
		}

		// Get user session
		sessionDriver, err := csc.Client.GetSessionDriver(protocol.NewSignalAddress(participantID, 1))
		if err != nil {
			logger.Info("Sending message to user without session ", participantID)
			sessionDriver, err = csc.CreateIndividualSession(protocol.NewSignalAddress(participantID, 1))
			if err != nil {
				logger.Error("Failed to create session with ", participantID, ": ", err)
				return err
			}
		}

		encryptedMsg := sessionDriver.EncryptMessage(packedMessageMarshal)
		messageWrapper := &pb.MessageWrapper{
			SenderID:         csc.chatbotID,
			RecipientID:      participantID,
			EncryptedMessage: encryptedMsg.Serialize(),
			HasPreKey:        encryptedMsg.Type() == protocol.PREKEY_TYPE,
			ChatbotKeyUpdatePack: &pb.MultiTreeKEMExternalKeyUpdatePack{
				ChatbotUpdate:   treekem.ECKEMCipherTextPbConvert(&chatbotUpdate),
				NewCbPubKey:     newCbPubKey,
				NewCbSignPubKey: newCbSignPubKey,
			},
		}

		messages[participantID] = messageWrapper
	}

	return csc.Client.SendClientSideGroupMessage(groupID, messages)
}

func (csc *ClientSideChatbot) HandleClientSideGroupMessage(message []byte, senderID string, treeKEMKeyUpdatePack *pb.TreeKEMKeyUpdatePack) ([]byte, pb.MessageType) {
	groupId, groupMessage, groupMessageType := csc.Client.ParseClientSideGroupMessage(message, senderID)

	// Handle treekem key update
	sessionDriver, err := csc.Client.GetClientSideGroupSessionDriver(groupId)
	if err != nil {
		logger.Error("Not in the group: ", groupId)
	}

	if treeKEMKeyUpdatePack != nil {
		logger.Info("Received TreeKEM update from ", senderID, " for client-side group ", groupId)

		updateMessage := treekem.PbECKEMCipherTextConvert(treeKEMKeyUpdatePack.GetChatbotUpdateCiphertexts().GetCiphertexts()[csc.chatbotID])
		err = sessionDriver.HandleTreeKEMUserKeyUpdate(updateMessage, treeKEMKeyUpdatePack.GetNewRootPubKey(), treeKEMKeyUpdatePack.GetNewRootSignPubKey())
		if err != nil {
			logger.Error("UpdateTreeKEMUserKey failed: ", groupId, err)
		}
	}

	return groupMessage, groupMessageType
}

//func (csc *ClientSideChatbot) HandleClientSideGroupIGAMessage(messageWrapper *pb.MessageWrapper) ([]byte, pb.MessageType) {
//
//}
