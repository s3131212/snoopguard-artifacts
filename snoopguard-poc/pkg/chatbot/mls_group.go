package chatbot

import (
	"chatbot-poc-go/pkg/client"
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/treekem"
	"chatbot-poc-go/pkg/util"
	"fmt"
	syntax "github.com/cisco/go-tls-syntax"
	"github.com/s3131212/go-mls"
	"go.mau.fi/libsignal/logger"
	"google.golang.org/protobuf/proto"
)

/*
PostCreateMlsGroup is called when a new MLS group is created.
*/
func (csc *ClientSideChatbot) PostCreateMlsGroup(groupID string, isIGA bool, isPseudo bool, treekemRootPub []byte, treekemRootSignPub []byte, initLeaf []byte, welcome mls.Welcome, keyPackageId uint32) (*client.MlsGroupSessionDriver, error) {
	sessionDriver, err := csc.Client.GetMlsGroupSessionDriver(groupID)
	if err != nil {
		return nil, err
	}

	if !isIGA {
		// If the welcome message is not set, set up the group state from empty
		if welcome.Secrets == nil {
			err = csc.Client.SetMlsGroupStateFromEmpty(groupID, keyPackageId)
		} else {
			err = csc.Client.SetMlsGroupStateFromWelcome(groupID, welcome, keyPackageId)
		}
		if err != nil {
			logger.Error("Failed to set MLS group state: ", err)
			return nil, err
		}
	}

	// Setup MlsMultiTree
	err = sessionDriver.InitiateMlsMultiTreeExternal(treekemRootPub, treekemRootSignPub, initLeaf)

	// Setup IGA config
	sessionDriver.SetChatbotIsIGA(csc.chatbotID, isIGA)
	sessionDriver.SetChatbotIsPseudo(csc.chatbotID, isPseudo)

	return sessionDriver, nil
}

/*
HandleMlsGroupMessage handles the incoming MLS group message.
*/
func (csc *ClientSideChatbot) HandleMlsGroupMessage(messageWrapper *pb.MessageWrapper) ([]byte, pb.MessageType) {
	groupId, senderId := messageWrapper.RecipientID, messageWrapper.SenderID

	sessionDriver, err := csc.Client.GetMlsGroupSessionDriver(groupId)
	if err != nil {
		logger.Error("Received message from MlS group without session ", groupId)
		return nil, -1
	}

	// Update CMRT
	if sessionDriver.GetChatbotIsIGA(csc.chatbotID) {
		treeKEMKeyUpdatePack := messageWrapper.GetTreeKEMKeyUpdatePack()
		if treeKEMKeyUpdatePack != nil && treeKEMKeyUpdatePack.GetNewRootPubKey() != nil {
			logger.Info("Chatbot Received TreeKEM update from ", messageWrapper.SenderID, " for MLS group ", messageWrapper.RecipientID)

			updateMessage := treekem.PbECKEMCipherTextConvert(treeKEMKeyUpdatePack.GetChatbotUpdateCiphertexts().GetCiphertexts()[csc.chatbotID])
			err = sessionDriver.HandleTreeKEMUserKeyUpdate(updateMessage, treeKEMKeyUpdatePack.GetNewRootPubKey(), treeKEMKeyUpdatePack.GetNewRootSignPubKey())
			if err != nil {
				logger.Warning("UpdateTreeKEMUserKey failed, maybe this is a skipped message? ", messageWrapper.RecipientID, err)

				/* We no longer need to send a validation message after the protocol update.
				// Send a validation message to the group broadcasting that the message is invalid and therefore is skipped.
				if messageWrapper.GetIsIGA() || messageWrapper.GetIsPseudo() {
					err = csc.SendMlsValidationMessage(messageWrapper.RecipientID, []byte("Invalid message"), pb.MessageType_SKIP)
					if err != nil {
						logger.Error("Error sending validation message to group ", messageWrapper.RecipientID, ": ", err)
						return nil, -1
					}
				}
				*/
				return []byte("Invalid message"), pb.MessageType_SKIP
			}
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
		logger.Info(fmt.Sprintf("Received pseudonym message in MLS group %v from pseudoUser %v: %v", messageWrapper.RecipientID, messageWrapper.SenderID, string(message)))

		/* We no longer need to send a validation message after the protocol update.
		// Send a validation message to the group.
		err = csc.SendMlsValidationMessage(messageWrapper.RecipientID, message, messageType)
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
		logger.Info(fmt.Sprintf("Received IGA message in MLS group %v: %v", messageWrapper.RecipientID, string(message)))

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

		// Send a validation message to the group. Pseudonym registration message does not need to be verified.
		if messageType != pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE {
			if messageType == pb.MessageType_SKIP {
				logger.Info("Received SKIP message in MLS group ", messageWrapper.RecipientID)
			}
			/* We no longer need to send a validation message after the protocol update.
			err = csc.SendMlsValidationMessage(messageWrapper.RecipientID, message, messageType)
			if err != nil {
				logger.Error("Error sending validation message to group ", messageWrapper.RecipientID, ": ", err)
				return nil, -1
			}
			*/
		}

		return message, messageType
	} else {
		deserializedCiphertext, err := util.DeserializeMLSCiphertext(messageWrapper.EncryptedMessage)
		if err != nil {
			return nil, -1
		}

		// Forward the message to the MLS group handler.
		message, messageType, _ := sessionDriver.ParseEncryptedMessage(senderId, deserializedCiphertext)
		logger.Info(fmt.Sprintf("Received message from %v in MLS group %v with type %v: %v", messageWrapper.SenderID, messageWrapper.RecipientID, messageType.String(), string(message)))

		// Handle the commit
		commit := &mls.MLSPlaintext{}
		_, err = syntax.Unmarshal(messageWrapper.GetMlsCommit(), commit)
		if err != nil {
			logger.Error(err)
			return nil, -1
		}
		err = sessionDriver.HandleCommit(commit, messageWrapper.SenderID)
		if err != nil {
			logger.Error(err)
			return nil, -1
		}

		return message, messageType
	}

}

/*
SendMlsGroupMessage sends a message to an MLS group.
*/
func (csc *ClientSideChatbot) SendMlsGroupMessage(groupID string, messageRaw []byte, messageType pb.MessageType) error {
	logger.Info("Sending message to MLS group ", groupID, ": ", string(messageRaw))
	sessionDriver, err := csc.Client.GetMlsGroupSessionDriver(groupID)
	if err != nil {
		logger.Error(err)
		return err
	}

	// Update TreeKEM
	var chatbotUpdate treekem.ECKEMCipherText
	var newCbPubKey []byte
	var newCbSignPubKey []byte
	var chatbotKeyUpdatePack *pb.MultiTreeKEMExternalKeyUpdatePack

	var serializedCipherText []byte
	var serializedCommit []byte

	if sessionDriver.GetChatbotIsIGA(csc.chatbotID) {
		chatbotUpdate, newCbPubKey, newCbSignPubKey, err = sessionDriver.GenerateMlsMultiTreeExternalKeyUpdate()
		if err != nil {
			logger.Error(err)
			return err
		}
		serializedCipherText = sessionDriver.EncryptMessageByMlsMultiTreeExternalRoot(messageRaw, messageType, sessionDriver.GetMlsMultiTreeExternal().GetSelfNode().SignPrivate).Serialize()
		chatbotKeyUpdatePack = &pb.MultiTreeKEMExternalKeyUpdatePack{
			ChatbotUpdate:   treekem.ECKEMCipherTextPbConvert(&chatbotUpdate),
			NewCbPubKey:     newCbPubKey,
			NewCbSignPubKey: newCbSignPubKey,
		}
	} else {
		ct, commit, err := sessionDriver.EncryptMessage(messageRaw, messageType, nil)
		if err != nil {
			logger.Error("Failed to encrypt message: ", err)
			return err
		}
		serializedCipherText, err = util.SerializeMLSCiphertext(*ct)
		if err != nil {
			logger.Error("Failed to serialize MLS ciphertext: ", err)
			return err
		}

		serializedCommit, err = syntax.Marshal(commit)
		if err != nil {
			return err
		}
	}

	messageWrapper := &pb.MessageWrapper{
		SenderID:             csc.chatbotID,
		RecipientID:          groupID,
		EncryptedMessage:     serializedCipherText,
		ChatbotMessages:      nil,
		HasPreKey:            false,
		IsIGA:                sessionDriver.GetChatbotIsIGA(csc.chatbotID),
		ChatbotKeyUpdatePack: chatbotKeyUpdatePack,
		MlsCommit:            serializedCommit,
	}

	return csc.Client.SendMlsGroupMessage(groupID, messageWrapper)
}

/*
SendMlsValidationMessage sends the input message to the group with the message type of pb.MessageType_VALIDATION_MESSAGE.
*/
func (csc *ClientSideChatbot) SendMlsValidationMessage(groupID string, message []byte, messageType pb.MessageType) error {
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

	return csc.SendMlsGroupMessage(groupID, validationMessageMarshal, pb.MessageType_VALIDATION_MESSAGE)
}
