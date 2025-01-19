package chatbot

import (
	"chatbot-poc-go/pkg/client"
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/treekem"
	"fmt"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/libsignal/protocol"
	"google.golang.org/protobuf/proto"
)

/*
PostCreateServerSideGroup is called when a new client-side group is created.
*/
func (csc *ClientSideChatbot) PostCreateServerSideGroup(groupID string, isIGA bool, isPseudo bool, treekemRootPub []byte, treekemRootSignPub []byte, initLeaf []byte) (*client.ServerSideGroupSessionDriver, error) {
	sessionDriver, err := csc.Client.GetServerSideGroupSessionDriver(groupID)
	if err != nil {
		return nil, err
	}

	// Set up MultiTreeKEMExternal
	err = sessionDriver.InitiateMultiTreeKEMExternal(treekemRootPub, treekemRootSignPub, initLeaf)
	if err != nil {
		logger.Error("Failed to initiate MultiTreeKEMExternal: ", err)
		return nil, err
	}

	// Setup IGA config
	sessionDriver.SetChatbotIsIGA(csc.chatbotID, isIGA)
	sessionDriver.SetChatbotIsPseudo(csc.chatbotID, isPseudo)

	return sessionDriver, nil
}

/*
HandleServerSideGroupMessage handles the incoming server-side group message.
Todo: modularize this function. It's too big now.
*/
func (csc *ClientSideChatbot) HandleServerSideGroupMessage(messageWrapper *pb.MessageWrapper) ([]byte, pb.MessageType) {
	sessionDriver, err := csc.Client.GetServerSideGroupSessionDriver(messageWrapper.RecipientID)
	if err != nil {
		logger.Debug("Received message from server-side fanout group without session ", messageWrapper.RecipientID)
		sessionDriver = csc.Client.CreateServerSideGroupSessionAndDriver(messageWrapper.RecipientID, []string{}, []string{})
	}

	treeKEMKeyUpdatePack := messageWrapper.GetTreeKEMKeyUpdatePack()
	if treeKEMKeyUpdatePack != nil && treeKEMKeyUpdatePack.GetNewRootPubKey() != nil {
		logger.Info("Chatbot Received TreeKEM update from ", messageWrapper.SenderID, " for server-side group ", messageWrapper.RecipientID)

		updateMessage := treekem.PbECKEMCipherTextConvert(treeKEMKeyUpdatePack.GetChatbotUpdateCiphertexts().GetCiphertexts()[csc.chatbotID])
		err = sessionDriver.HandleTreeKEMUserKeyUpdate(updateMessage, treeKEMKeyUpdatePack.GetNewRootPubKey(), treeKEMKeyUpdatePack.GetNewRootSignPubKey())
		if err != nil {
			logger.Warning("UpdateTreeKEMUserKey failed, maybe this is a skipped message? ", messageWrapper.RecipientID, err)

			/* We no longer need to send a validation message after the protocol update.
			// Send a validation message to the group broadcasting that the message is invalid and therefore is skipped.
			if messageWrapper.GetIsIGA() || messageWrapper.GetIsPseudo() {
				err = csc.SendServerSideValidationMessage(messageWrapper.RecipientID, []byte("Invalid message"), pb.MessageType_SKIP)
				if err != nil {
					logger.Error("Error sending validation message to group ", messageWrapper.RecipientID, ": ", err)
					return nil, -1
				}
			}
			*/

			return []byte("Invalid message"), pb.MessageType_SKIP
		}
	}

	if messageWrapper.GetIsPseudo() {
		if !sessionDriver.GetChatbotIsPseudo(csc.chatbotID) {
			logger.Error("Received pseudonym message for chatbot ", csc.chatbotID, " in group ", messageWrapper.RecipientID, " but chatbot is not pseudo.")
			return nil, -1
		}

		pseudoUser, exists := csc.groupPseudonyms[messageWrapper.RecipientID][messageWrapper.SenderID]
		if !exists {
			logger.Error("Received pseudonym message for chatbot ", csc.chatbotID, " in group ", messageWrapper.RecipientID, " using pseudonym ", messageWrapper.SenderID, " but the pseudonym is not registered.")
			return nil, -1
		}

		message, messageType := sessionDriver.ParseEncryptedIGAMessage(messageWrapper.EncryptedMessage, pseudoUser.SigningPubKey)
		logger.Info(fmt.Sprintf("Received pseudonym message in server-side group %v from pseudoUser %v: %v", messageWrapper.RecipientID, messageWrapper.SenderID, string(message)))

		/* We no longer need to send a validation message after the protocol update.
		// Send a validation message to the group.
		err = csc.SendServerSideValidationMessage(messageWrapper.RecipientID, message, messageType)
		if err != nil {
			logger.Error("Error sending validation message to group ", messageWrapper.RecipientID, ": ", err)
			return nil, -1
		}
		*/

		return message, messageType
	} else if messageWrapper.GetIsIGA() {
		if !sessionDriver.GetChatbotIsIGA(csc.chatbotID) {
			logger.Error("Received IGA message for chatbot ", csc.chatbotID, " in group ", messageWrapper.RecipientID, " but chatbot is not IGA.")
			return nil, -1
		}

		message, messageType := sessionDriver.ParseEncryptedIGAMessage(messageWrapper.EncryptedMessage, nil)
		logger.Info(fmt.Sprintf("Received IGA message in server-side group %v: %v", messageWrapper.RecipientID, string(message)))

		// Pseudonym registration message would be sent in IGA channel. This is the only case when user send IGA message to pseudonymity-enabled bot.
		if messageType == pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE {
			logger.Info("Received pseudonym registration message for chatbot ", csc.chatbotID, " in group ", messageWrapper.RecipientID)

			if !sessionDriver.GetChatbotIsPseudo(csc.chatbotID) {
				logger.Error("Received pseudonym registration message for chatbot ", csc.chatbotID, " in group ", messageWrapper.RecipientID, " but chatbot is not pseudo.")
				return nil, -1
			}

			// Unpack message as PseudonymRegistrationMessage
			pseudonymRegistrationMessage := &pb.PseudonymRegistrationMessage{}
			err := proto.Unmarshal(message, pseudonymRegistrationMessage)
			if err != nil {
				logger.Error("Error unmarshalling message: ", err)
				return nil, -1
			}

			// Update group participants
			sessionDriver.UpdateGroupParticipantIDs(append(sessionDriver.GetGroupParticipants(), pseudonymRegistrationMessage.GetPseudoUserID()))

			if csc.groupPseudonyms[messageWrapper.RecipientID] == nil {
				csc.groupPseudonyms[messageWrapper.RecipientID] = make(map[string]*PseudoUser)
			}
			csc.groupPseudonyms[messageWrapper.RecipientID][pseudonymRegistrationMessage.GetPseudoUserID()] = &PseudoUser{
				PseudoUserID:  pseudonymRegistrationMessage.GetPseudoUserID(),
				SigningPubKey: pseudonymRegistrationMessage.GetSigningKeyPub(),
			}
		}

		/* We no longer need to send a validation message after the protocol update.
		// Send a validation message to the group. Pseudonym registration message does not need to be verified.
		if messageType != pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE {
			err = csc.SendServerSideValidationMessage(messageWrapper.RecipientID, message, messageType)
			if err != nil {
				logger.Error("Error sending validation message to group ", messageWrapper.RecipientID, ": ", err)
				return nil, -1
			}
		}
		*/

		return message, messageType
	} else {
		// Forward the message to the server-side group handler.
		msg, err := csc.Client.ParseSenderKeyMessage(messageWrapper.EncryptedMessage) // Convert to SenderKeyMessage
		if err != nil {
			logger.Error("Failed to decode sender key message ", err)
			panic("")
		}
		message, messageType, _ := sessionDriver.ParseEncryptedMessage(messageWrapper.SenderID, msg)
		logger.Info(fmt.Sprintf("Received message from %v in server-side group %v with type %v: %v", messageWrapper.SenderID, messageWrapper.RecipientID, messageType.String(), string(message)))
		return message, messageType
	}
}

/*
SendServerSideGroupMessage sends a message to a server-side group.
*/
func (csc *ClientSideChatbot) SendServerSideGroupMessage(groupID string, messageRaw []byte, messageType pb.MessageType) error {
	logger.Info("Sending message to server-side group ", groupID, ": ", string(messageRaw))
	sessionDriver, err := csc.Client.GetServerSideGroupSessionDriver(groupID)
	if err != nil {
		return err
	}

	// Update TreeKEM
	var chatbotUpdate treekem.ECKEMCipherText
	var newCbPubKey []byte
	var newCbSignPubKey []byte
	var chatbotKeyUpdatePack *pb.MultiTreeKEMExternalKeyUpdatePack

	var ct []byte
	if sessionDriver.GetChatbotIsIGA(csc.chatbotID) {
		chatbotUpdate, newCbPubKey, newCbSignPubKey, err = sessionDriver.GenerateMultiTreeKEMExternalKeyUpdate()
		if err != nil {
			logger.Error(err)
			return err
		}
		ct = sessionDriver.EncryptMessageByMultiTreeKEMExternalRoot(messageRaw, messageType, sessionDriver.GetMultiTreeKEMExternal().GetSelfNode().SignPrivate).Serialize()
		chatbotKeyUpdatePack = &pb.MultiTreeKEMExternalKeyUpdatePack{
			ChatbotUpdate:   treekem.ECKEMCipherTextPbConvert(&chatbotUpdate),
			NewCbPubKey:     newCbPubKey,
			NewCbSignPubKey: newCbSignPubKey,
		}
	} else {
		ct = sessionDriver.EncryptMessageBySendingSession(messageRaw, messageType, nil).SignedSerialize()
	}

	messageWrapper := &pb.MessageWrapper{
		SenderID:             csc.chatbotID,
		RecipientID:          groupID,
		EncryptedMessage:     ct,
		ChatbotMessages:      nil,
		HasPreKey:            false,
		IsIGA:                sessionDriver.GetChatbotIsIGA(csc.chatbotID),
		ChatbotKeyUpdatePack: chatbotKeyUpdatePack,
	}

	return csc.Client.SendServerSideGroupMessage(groupID, messageWrapper)
}

/*
DistributeSelfSenderKeyToUserID sends the own sender key to the given userID.
*/
func (csc *ClientSideChatbot) DistributeSelfSenderKeyToUserID(userID string, groupID string, bounceBack bool) error {
	sessionDriver, err := csc.Client.GetServerSideGroupSessionDriver(groupID)
	if err != nil {
		return err
	}

	if sessionDriver.GetChatbotIsIGA(csc.chatbotID) {
		logger.Error("Chatbot ", csc.chatbotID, " is IGA in group ", groupID, ". Skip distributing self sender key.")
		return nil
	}

	msg := pb.SenderKeyDistributionMessage{
		GroupID:                      groupID,
		SenderKeyDistributionMessage: sessionDriver.GetSelfSenderKey().Serialize(),
		BounceBack:                   bounceBack,
	}
	msgMarshal, err := proto.Marshal(&msg)
	if err != nil {
		logger.Error("Error marshalling message: ", err)
		return err
	}

	return csc.SendIndividualMessage(protocol.NewSignalAddress(userID, 1), msgMarshal, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE)
}

/*
DistributeSelfSenderKeyToAll sends the own sender key to all group participants.
*/
func (csc *ClientSideChatbot) DistributeSelfSenderKeyToAll(groupID string) error {
	sessionDriver, err := csc.Client.GetServerSideGroupSessionDriver(groupID)
	if err != nil {
		return err
	}

	if sessionDriver.GetChatbotIsIGA(csc.chatbotID) {
		logger.Error("Chatbot ", csc.chatbotID, " is IGA in group ", groupID, ". Skip distributing self sender key.")
		return nil
	}

	for _, userID := range sessionDriver.GetGroupParticipants() {
		err = csc.DistributeSelfSenderKeyToUserID(userID, groupID, true)
		if err != nil {
			logger.Error("Error sending sender key distribution message to ", userID, ": ", err)
			return err
		}
	}

	return nil
}

/*
SendServerSideValidationMessage sends the input message to the group with the message type of pb.MessageType_VALIDATION_MESSAGE.
*/
func (csc *ClientSideChatbot) SendServerSideValidationMessage(groupID string, message []byte, messageType pb.MessageType) error {
	logger.Info("Sending validation message to group ", groupID, ": ", string(message))

	validationMessage := &pb.ValidationMessage{
		GroupID:             groupID,
		PreviousMessage:     message,
		PreviousMessageType: messageType,
	}

	validationMessageMarshal, err := proto.Marshal(validationMessage)
	if err != nil {
		logger.Error("Error marshalling message: ", err)
		return err
	}

	return csc.SendServerSideGroupMessage(groupID, validationMessageMarshal, pb.MessageType_VALIDATION_MESSAGE)
}
