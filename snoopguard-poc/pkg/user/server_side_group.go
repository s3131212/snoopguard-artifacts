package user

import (
	"chatbot-poc-go/pkg/client"
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/treekem"
	"chatbot-poc-go/pkg/util"
	"fmt"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/libsignal/protocol"
	"google.golang.org/protobuf/proto"
)

/*
PostCreateServerSideGroup is called when a new server-side group is created.
*/
func (csu *ClientSideUser) PostCreateServerSideGroup(groupID string, chatbotIsIGA map[string]bool, chatbotIsPseudo map[string]bool, treekemGroupInitKey treekem.GroupInitKey, treekemInitLeaf []byte, chatbotPubKeys map[string][]byte, chatbotSignPubKeys map[string][]byte, lastTreeKemRootCiphertexts map[string]treekem.ECKEMCipherText) (*client.ServerSideGroupSessionDriver, error) {
	sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
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

	// Set Chatbot Pseudo status.
	for chatbotID, isPseudo := range chatbotIsPseudo {
		sessionDriver.SetChatbotIsPseudo(chatbotID, isPseudo)
	}

	// Distribute sender key to all users
	//logger.Info("Distributing sender key to all users in group ", groupID)
	//senderKeyDistributionMessage := pb.SenderKeyDistributionMessage{
	//	GroupID:                      groupID,
	//	SenderKeyDistributionMessage: sessionDriver.GetSelfSenderKey().Serialize(),
	//}
	//senderKeyDistributionMessageMarshal, err := proto.Marshal(&senderKeyDistributionMessage)
	//if err != nil {
	//	logger.Error("Error marshalling message: ", err)
	//	return nil, err
	//}
	//
	//for _, userID := range sessionDriver.GetGroupParticipants() {
	//	if userID == csu.userID {
	//		continue
	//	}
	//	err = csu.SendIndividualMessage(protocol.NewSignalAddress(userID, 1), senderKeyDistributionMessageMarshal, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE)
	//	if err != nil {
	//		logger.Error("Error sending sender key distribution message to ", userID, ": ", err)
	//		return nil, err
	//	}
	//}
	//
	//for _, chatbotID := range sessionDriver.GetGroupChatbots() {
	//	if !sessionDriver.GetChatbotIsIGA(chatbotID) {
	//		err = csu.SendIndividualMessage(protocol.NewSignalAddress(chatbotID, 1), senderKeyDistributionMessageMarshal, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE)
	//		if err != nil {
	//			logger.Error("Error sending sender key distribution message to ", chatbotID, ": ", err)
	//			return nil, err
	//		}
	//	}
	//}

	return sessionDriver, nil
}

/*
HandleServerSideGroupMessage handles the incoming server-side group message.
Todo: modularize this function. It's too big now.
*/
func (csu *ClientSideUser) HandleServerSideGroupMessage(messageWrapper *pb.MessageWrapper) ([]byte, pb.MessageType) {
	groupId, senderId := messageWrapper.RecipientID, messageWrapper.SenderID

	sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupId)
	if err != nil {
		logger.Debug("Received message from server-side fanout group without session ", groupId)
		sessionDriver = csu.Client.CreateServerSideGroupSessionAndDriver(groupId, []string{}, []string{})
	}

	if !messageWrapper.GetIsIGA() && !messageWrapper.GetIsPseudo() {
		msg, err := csu.Client.ParseSenderKeyMessage(messageWrapper.EncryptedMessage) // Convert to SenderKeyMessage
		if err != nil {
			logger.Error("Failed to decode sender key message ", err)
			panic("")
		}
		message, messageType, receivingChatbotIDs := sessionDriver.ParseEncryptedMessage(senderId, msg)

		logger.Info(fmt.Sprintf("Received message from %v in server-side group %v with type %v: %v", messageWrapper.SenderID, messageWrapper.RecipientID, messageType.String(), string(message)))

		// Handle treekem key update. This only happens when the message comes from other users and is also intended for other chatbots.
		treeKEMKeyUpdatePack := messageWrapper.GetTreeKEMKeyUpdatePack()
		if treeKEMKeyUpdatePack != nil {
			logger.Info("Received TreeKEM update from ", senderId, " for server-side group ", groupId)

			userUpdate := treekem.PbTreeKEMUserUpdateConvert(treeKEMKeyUpdatePack.GetUserUpdate())
			err = sessionDriver.UpdateTreeKEMUserKey(&userUpdate, receivingChatbotIDs)
			if err != nil {
				logger.Error("UpdateTreeKEMUserKey failed: ", groupId)
			}
		}

		return message, messageType
	}

	// Handle chatbot key update. This only happens when the message comes from IGA chatbot.
	chatbotKeyUpdatePack := messageWrapper.GetChatbotKeyUpdatePack()
	if chatbotKeyUpdatePack != nil {
		logger.Info("Received MultiTreeKEM update from ", senderId, " for server-side group ", groupId)

		chatbotUpdate := treekem.PbECKEMCipherTextConvert(chatbotKeyUpdatePack.GetChatbotUpdate())
		newCbPubKey := chatbotKeyUpdatePack.GetNewCbPubKey()
		newCbSignPubKey := chatbotKeyUpdatePack.GetNewCbSignPubKey()
		err = sessionDriver.HandleMultiTreeKEMExternalKeyUpdate(senderId, chatbotUpdate, newCbPubKey, newCbSignPubKey)
		if err != nil {
			logger.Error("UpdateMultiTreeKEMExternalKey failed: ", groupId, err)
		}
	}

	if messageWrapper.GetIsIGA() {
		message, messageType := sessionDriver.ParseEncryptedExternalIGAMessage(messageWrapper.EncryptedMessage, messageWrapper.SenderID)
		logger.Info(fmt.Sprintf("%v Received IGA message in server-side group %v from %v: %v", csu.userID, messageWrapper.RecipientID, messageWrapper.SenderID, string(message)))

		// Validation message always comes from IGA channel.
		if messageType == pb.MessageType_VALIDATION_MESSAGE {
			logger.Info("Received validation message from ", messageWrapper.SenderID, " in server-side group ", messageWrapper.RecipientID, " with content: ", string(message))
		}

		return message, messageType
	} else {
		// Forward the message to the server-side group handler.
		msg, err := csu.Client.ParseSenderKeyMessage(messageWrapper.EncryptedMessage) // Convert to SenderKeyMessage
		if err != nil {
			logger.Error("Failed to decode sender key message ", err)
			panic("")
		}
		message, messageType, _ := sessionDriver.ParseEncryptedMessage(senderId, msg)

		logger.Info(fmt.Sprintf("Received message from %v in server-side group %v with type %v: %v", messageWrapper.SenderID, messageWrapper.RecipientID, messageType.String(), string(message)))
		return message, messageType
	}

}

/*
SendServerSideGroupMessage sends a message to a server-side group.
*/
func (csu *ClientSideUser) SendServerSideGroupMessage(groupID string, messageRaw []byte, messageType pb.MessageType, receivingChatbotIDs []string, hideTrigger bool) error {
	messageWrapper, err := csu.GenerateServerSideGroupMessageCipherText(groupID, messageRaw, messageType, receivingChatbotIDs, hideTrigger)
	if err != nil {
		return err
	}

	return csu.Client.SendServerSideGroupMessage(groupID, messageWrapper)
}

/*
GenerateServerSideGroupMessageCipherText generate the ciphertext for server-side group without actually sending it.

func (csu *ClientSideUser) GenerateServerSideGroupMessageCipherText(groupID string, messageRaw []byte, messageType pb.MessageType, receivingChatbotIDs []string, hideTrigger bool) (*pb.MessageWrapper, error) {
	logger.Info("Sending message to server-side group ", groupID, ": ", string(messageRaw))
	sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	cipherText := sessionDriver.EncryptMessageBySendingSession(messageRaw, messageType).SignedSerialize()

	chatbotMessages, userUpdate, receivingChatbotIDs, err := csu.GetServerSideChatbotEncryptedMessage(groupID, messageRaw, messageType, receivingChatbotIDs, hideTrigger, cipherText)
	if err != nil {
		return nil, err
	}

	var treeKEMKeyUpdatePack *pb.TreeKEMKeyUpdatePack
	if userUpdate != nil {
		treeKEMKeyUpdatePack = &pb.TreeKEMKeyUpdatePack{
			UserUpdate: treekem.TreeKEMUserUpdatePbConvert(userUpdate),
		}
	}

	messageWrapper := &pb.MessageWrapper{
		SenderID:             csu.userID,
		RecipientID:          groupID,
		EncryptedMessage:     cipherText,
		ChatbotMessages:      chatbotMessages,
		HasPreKey:            false,
		ChatbotIds:           receivingChatbotIDs,
		TreeKEMKeyUpdatePack: treeKEMKeyUpdatePack,
	}

	return messageWrapper, nil
}
*/

/*
GenerateServerSideGroupMessageCipherText generate the ciphertext for server-side group without actually sending it.
*/
func (csu *ClientSideUser) GenerateServerSideGroupMessageCipherText(groupID string, messageRaw []byte, messageType pb.MessageType, receivingChatbotIDs []string, hideTrigger bool) (*pb.MessageWrapper, error) {
	logger.Info("Sending message to server-side group ", groupID, ": ", string(messageRaw))
	sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	cipherTextForUser := sessionDriver.EncryptMessageBySendingSession(messageRaw, messageType, receivingChatbotIDs).SignedSerialize()

	var chatbotMessages []*pb.ChatbotMessage
	var treeKEMKeyUpdatePackChatbot *pb.TreeKEMKeyUpdatePack
	var userUpdate *treekem.UserUpdate
	var actuallySentChatbotIDs []string

	// Determine if any receiving chatbot has IGA or is pseudonymous
	hasIGAOrPseudo := false
	var exampleChatbotId string
	if hideTrigger {
		for _, chatbotID := range sessionDriver.GetGroupChatbots() {
			if sessionDriver.GetChatbotIsIGA(chatbotID) || sessionDriver.GetChatbotIsPseudo(chatbotID) {
				hasIGAOrPseudo = true
				if util.ContainString(chatbotID, receivingChatbotIDs) {
					exampleChatbotId = chatbotID
					break
				}
			}
		}
		// If no receiving chatbot is IGA but there are non-receiving chatbot that is IGA, we simply fake a chatbot ID, as no one is going to use the wrong ciphertext.
		if exampleChatbotId == "" {
			for _, chatbotID := range sessionDriver.GetGroupChatbots() {
				if sessionDriver.GetChatbotIsIGA(chatbotID) || sessionDriver.GetChatbotIsPseudo(chatbotID) {
					exampleChatbotId = chatbotID
					break
				}
			}
		}
		actuallySentChatbotIDs = sessionDriver.GetGroupChatbots()
	} else {
		for _, chatbotID := range receivingChatbotIDs {
			if sessionDriver.GetChatbotIsIGA(chatbotID) || sessionDriver.GetChatbotIsPseudo(chatbotID) {
				hasIGAOrPseudo = true
				exampleChatbotId = chatbotID
				break
			}
		}
		actuallySentChatbotIDs = receivingChatbotIDs
	}

	if hasIGAOrPseudo {

		// Generate TreeKEM update first
		var chatbotUpdateCiphertexts map[string]treekem.ECKEMCipherText
		var newTreeKemRootPubKey []byte
		var newTreeKemRootSignPubKey []byte

		userUpdate, chatbotUpdateCiphertexts, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err = sessionDriver.GenerateMultiTreeKEMKeyUpdate(receivingChatbotIDs)
		if err != nil {
			logger.Error("Failed to generate MultiTreeKEM key update: ", err)
			return nil, err
		}

		// When hide triggers, add fake chatbot update ciphertexts for IGA chatbots not receiving the message
		if hideTrigger {
			for _, chatbotID := range sessionDriver.GetGroupChatbots() {
				if (sessionDriver.GetChatbotIsIGA(chatbotID) || sessionDriver.GetChatbotIsPseudo(chatbotID)) && !util.ContainString(chatbotID, receivingChatbotIDs) {
					// Create a fake treekem.ECKEMCipherText filled with random bytes
					chatbotUpdateCiphertexts[chatbotID] = treekem.ECKEMCipherText{
						CipherText: util.RandomBytes(32),
						IV:         util.RandomBytes(16),
						Public:     util.RandomBytes(32),
					}
				}
			}
		}

		// Encrypt the message using MultiTreeKEM
		cipherTextForIGAChatbot := sessionDriver.EncryptMessageByMultiTreeKEMRoot(messageRaw, messageType, nil, exampleChatbotId, nil)

		// Prepare TreeKEMKeyUpdatePack
		treeKEMKeyUpdatePackChatbot = &pb.TreeKEMKeyUpdatePack{
			ChatbotUpdateCiphertexts: treekem.ECKEMCipherTextStringMapPbConvert(chatbotUpdateCiphertexts),
			NewRootPubKey:            newTreeKemRootPubKey,
			NewRootSignPubKey:        newTreeKemRootSignPubKey,
		}

		// Create ChatbotMessages with the same ciphertext
		for _, chatbotID := range actuallySentChatbotIDs {
			if sessionDriver.GetChatbotIsIGA(chatbotID) || sessionDriver.GetChatbotIsPseudo(chatbotID) {
				senderID := csu.userID
				var cipherTextTmp []byte
				if sessionDriver.GetChatbotIsPseudo(chatbotID) {
					// Sign the message with the pseudo user's signing key
					pseudoUser := csu.GetPseudoUser(groupID, chatbotID)
					if pseudoUser == nil {
						logger.Error("Failed to get pseudo user. Register a pseudonym before sending any message.")
						return nil, err
					}

					senderID = pseudoUser.PseudoUserID
					sig, err := util.Sign(cipherTextForIGAChatbot.CipherText, pseudoUser.SigningKeyPair.Private.Bytes())

					if err != nil {
						logger.Error("Failed to sign message: ", err)
						return nil, err
					}
					cipherTextTmp = util.CipherText{
						IV:         cipherTextForIGAChatbot.IV,
						CipherText: cipherTextForIGAChatbot.CipherText,
						Signature:  sig,
					}.Serialize()
				} else {
					cipherTextTmp = util.CipherText{
						IV:         cipherTextForIGAChatbot.IV,
						CipherText: cipherTextForIGAChatbot.CipherText,
						Signature:  nil,
					}.Serialize()
				}

				messageWrapper := &pb.MessageWrapper{
					SenderID:             senderID,
					RecipientID:          groupID,
					EncryptedMessage:     cipherTextTmp,
					HasPreKey:            false,
					ChatbotIds:           receivingChatbotIDs,
					IsIGA:                sessionDriver.GetChatbotIsIGA(chatbotID),
					IsPseudo:             sessionDriver.GetChatbotIsPseudo(chatbotID),
					TreeKEMKeyUpdatePack: treeKEMKeyUpdatePackChatbot,
				}
				chatbotMessage := &pb.ChatbotMessage{
					ChatbotID:        chatbotID,
					MessageWrapper:   messageWrapper,
					UseNormalMessage: false,
				}
				chatbotMessages = append(chatbotMessages, chatbotMessage)
			}
		}

		// Handle non-IGA/Pseudonymous chatbots if any
		nonIGAChatbots := []string{}
		for _, chatbotID := range receivingChatbotIDs {
			if !sessionDriver.GetChatbotIsIGA(chatbotID) && !sessionDriver.GetChatbotIsPseudo(chatbotID) && util.ContainString(chatbotID, receivingChatbotIDs) {
				nonIGAChatbots = append(nonIGAChatbots, chatbotID)
			}
		}

		if len(nonIGAChatbots) > 0 {
			// Create a MessageWrapper for non-IGA/Pseudonymous chatbots
			chatbotMessageWrapper := &pb.MessageWrapper{
				SenderID:             csu.userID,
				RecipientID:          groupID,
				EncryptedMessage:     cipherTextForUser,
				HasPreKey:            false,
				ChatbotIds:           receivingChatbotIDs,
				IsIGA:                false,
				IsPseudo:             false,
				TreeKEMKeyUpdatePack: treeKEMKeyUpdatePackChatbot,
			}

			// Add to chatbotMessages
			for _, chatbotID := range nonIGAChatbots {
				chatbotMessages = append(chatbotMessages, &pb.ChatbotMessage{
					ChatbotID:        chatbotID,
					MessageWrapper:   chatbotMessageWrapper,
					UseNormalMessage: false,
				})
			}
		}

		// Handle non-IGA/Pseudonymous skipped chatbots if any
		// Non-IGA/Pseudonymous chatbots cannot hide triggers, according to our paper.
		// For compatibility, we send a dummy message instead of the real message. This is not intended to be a secure solution.
		if hideTrigger {
			nonIGAnonReceivingChatbots := []string{}
			for _, chatbotID := range sessionDriver.GetGroupChatbots() {
				if !util.ContainString(chatbotID, receivingChatbotIDs) && !sessionDriver.GetChatbotIsIGA(chatbotID) && !sessionDriver.GetChatbotIsPseudo(chatbotID) {
					nonIGAnonReceivingChatbots = append(nonIGAnonReceivingChatbots, chatbotID)
				}
			}

			if len(nonIGAnonReceivingChatbots) > 0 {
				// Encrypt message for users
				dummyMessage := []byte(util.RandomString(len(messageRaw)))
				chatbotCipherText := sessionDriver.EncryptMessageBySendingSession(dummyMessage, pb.MessageType_SKIP, nil).SignedSerialize()

				// Create a MessageWrapper for non-IGA/Pseudonymous chatbots
				chatbotMessageWrapper := &pb.MessageWrapper{
					SenderID:             csu.userID,
					RecipientID:          groupID,
					EncryptedMessage:     chatbotCipherText,
					HasPreKey:            false,
					ChatbotIds:           receivingChatbotIDs,
					IsIGA:                false,
					IsPseudo:             false,
					TreeKEMKeyUpdatePack: treeKEMKeyUpdatePackChatbot,
				}

				// Add to chatbotMessages
				for _, chatbotID := range nonIGAnonReceivingChatbots {
					chatbotMessages = append(chatbotMessages, &pb.ChatbotMessage{
						ChatbotID:        chatbotID,
						MessageWrapper:   chatbotMessageWrapper,
						UseNormalMessage: false,
					})
				}
			}
		}
	} else {
		// Encrypt for chatbots without IGA or pseudonymity
		chatbotMessages = make([]*pb.ChatbotMessage, 0)
		for _, chatbotID := range receivingChatbotIDs {
			chatbotMessageWrapper := &pb.MessageWrapper{
				SenderID:         csu.userID,
				RecipientID:      groupID,
				EncryptedMessage: cipherTextForUser,
				ChatbotIds:       []string{chatbotID},
				// No TreeKEM update for non-IGA chatbots
			}

			chatbotMessages = append(chatbotMessages, &pb.ChatbotMessage{
				ChatbotID:        chatbotID,
				MessageWrapper:   chatbotMessageWrapper,
				UseNormalMessage: false,
			})
		}

		// Again, chatbot without IGA/Pseudonymity cannot hide triggers.
		// For compatibility, we send a dummy message instead of the real message.
		if hideTrigger {
			for _, chatbotID := range sessionDriver.GetGroupChatbots() {
				if !util.ContainString(chatbotID, receivingChatbotIDs) {
					dummyMessage := []byte(util.RandomString(len(messageRaw)))
					chatbotCipherText := sessionDriver.EncryptMessageBySendingSession(dummyMessage, pb.MessageType_SKIP, nil).SignedSerialize()
					chatbotMessageWrapper := &pb.MessageWrapper{
						SenderID:         csu.userID,
						RecipientID:      groupID,
						EncryptedMessage: chatbotCipherText,
						ChatbotIds:       []string{chatbotID},
						// No TreeKEM update for non-IGA chatbots
					}
					chatbotMessages = append(chatbotMessages, &pb.ChatbotMessage{
						ChatbotID:        chatbotID,
						MessageWrapper:   chatbotMessageWrapper,
						UseNormalMessage: false,
					})
				}
			}
		}
	}

	var treeKEMKeyUpdatePackUser *pb.TreeKEMKeyUpdatePack
	if userUpdate != nil {
		treeKEMKeyUpdatePackUser = &pb.TreeKEMKeyUpdatePack{
			UserUpdate: treekem.TreeKEMUserUpdatePbConvert(userUpdate),
		}
	}

	messageWrapper := &pb.MessageWrapper{
		SenderID:             csu.userID,
		RecipientID:          groupID,
		EncryptedMessage:     cipherTextForUser,
		IsIGA:                false,
		ChatbotMessages:      chatbotMessages,
		HasPreKey:            false,
		ChatbotIds:           actuallySentChatbotIDs,
		TreeKEMKeyUpdatePack: treeKEMKeyUpdatePackUser,
	}

	logger.Info("messageWrapper: ", messageWrapper)

	return messageWrapper, nil
}

/*
GetServerSideChatbotEncryptedMessage returns the encrypted message for the given chatbotID.
*/
func (csu *ClientSideUser) GetServerSideChatbotEncryptedMessage(groupID string, message []byte, messageType pb.MessageType, receivingChatbotIDs []string, hideTrigger bool, originalCipherText []byte) ([]*pb.ChatbotMessage, *treekem.UserUpdate, []string, error) {
	sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
	if err != nil {
		return nil, nil, nil, err
	}

	// If hiding triggers is true, for chatbots not in the receivingChatbotIDs, they will still receive the message.
	receivingChatbotIDsReal := receivingChatbotIDs
	if hideTrigger {
		for _, chatbotID := range sessionDriver.GetGroupChatbots() {
			if !util.ContainString(chatbotID, receivingChatbotIDs) {
				receivingChatbotIDs = append(receivingChatbotIDs, chatbotID)
			}
		}
	}

	// Update TreeKEM
	var userUpdate *treekem.UserUpdate
	var chatbotUpdateCiphertexts map[string]treekem.ECKEMCipherText
	var newTreeKemRootPubKey []byte
	var newTreeKemRootSignPubKey []byte

	for _, chatbotID := range receivingChatbotIDs {
		if sessionDriver.GetChatbotIsIGA(chatbotID) {
			userUpdate, chatbotUpdateCiphertexts, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err = sessionDriver.GenerateMultiTreeKEMKeyUpdate(receivingChatbotIDs)
			if err != nil {
				return nil, nil, nil, err
			}
			break
		}
	}

	chatbotMessages := make([]*pb.ChatbotMessage, 0)
	var encryptedMessageCache util.CipherText
	var encryptedSkipMessageCache util.CipherText

	for _, chatbotID := range receivingChatbotIDs {
		sendSkip := false

		if hideTrigger && !util.ContainString(chatbotID, receivingChatbotIDsReal) {
			sendSkip = true
		}

		var ct []byte
		var senderID string
		var hasPreKey bool
		if sessionDriver.GetChatbotIsPseudo(chatbotID) {
			pseudoUser := csu.GetPseudoUser(groupID, chatbotID)
			if pseudoUser == nil {
				logger.Error("Failed to get pseudo user. Register a pseudonym before sending any message.")
				return nil, nil, nil, err
			}

			if !sendSkip {
				if encryptedMessageCache.CipherText == nil {
					encryptedMessageCache = sessionDriver.EncryptMessageByMultiTreeKEMRoot(message, messageType, nil, chatbotID, nil)
				}
				sig, err := util.Sign(encryptedMessageCache.CipherText, pseudoUser.SigningKeyPair.Private.Bytes())
				if err != nil {
					logger.Error("Failed to sign message: ", err)
					return nil, nil, nil, err
				}
				ct = util.CipherText{
					IV:         encryptedMessageCache.IV,
					CipherText: encryptedMessageCache.CipherText,
					Signature:  sig,
				}.Serialize()
			} else {
				if encryptedSkipMessageCache.CipherText == nil {
					encryptedSkipMessageCache = sessionDriver.EncryptMessageByMultiTreeKEMRoot([]byte(util.RandomString(len(message))), pb.MessageType_SKIP, nil, chatbotID, nil)
				}
				sig, err := util.Sign(encryptedSkipMessageCache.CipherText, pseudoUser.SigningKeyPair.Private.Bytes())
				if err != nil {
					logger.Error("Failed to sign message: ", err)
					return nil, nil, nil, err
				}
				ct = util.CipherText{
					IV:         encryptedSkipMessageCache.IV,
					CipherText: encryptedSkipMessageCache.CipherText,
					Signature:  sig,
				}.Serialize()
			}
			senderID = pseudoUser.PseudoUserID
			hasPreKey = false
		} else if sessionDriver.GetChatbotIsIGA(chatbotID) {
			if !sendSkip {
				if encryptedMessageCache.CipherText == nil {
					encryptedMessageCache = sessionDriver.EncryptMessageByMultiTreeKEMRoot(message, messageType, nil, chatbotID, nil)
				}
				ct = encryptedMessageCache.Serialize()
			} else {
				if encryptedSkipMessageCache.CipherText == nil {
					encryptedSkipMessageCache = sessionDriver.EncryptMessageByMultiTreeKEMRoot([]byte(util.RandomString(len(message))), pb.MessageType_SKIP, nil, chatbotID, nil)
				}
				ct = encryptedSkipMessageCache.Serialize()
			}
			senderID = ""
			hasPreKey = false
		} else {
			if !sendSkip {
				ct = originalCipherText
			} else {
				ct = sessionDriver.EncryptMessageBySendingSession([]byte(util.RandomString(len(message))), pb.MessageType_SKIP, nil).SignedSerialize()
			}
			senderID = csu.userID
			hasPreKey = false
		}

		messageWrapper := &pb.MessageWrapper{
			SenderID:         senderID,
			RecipientID:      groupID,
			EncryptedMessage: ct,
			HasPreKey:        hasPreKey,
			ChatbotIds:       receivingChatbotIDs,
			IsIGA:            sessionDriver.GetChatbotIsIGA(chatbotID),
			IsPseudo:         sessionDriver.GetChatbotIsPseudo(chatbotID),
			TreeKEMKeyUpdatePack: &pb.TreeKEMKeyUpdatePack{
				ChatbotUpdateCiphertexts: treekem.ECKEMCipherTextStringMapPbConvert(map[string]treekem.ECKEMCipherText{chatbotID: chatbotUpdateCiphertexts[chatbotID]}),
				NewRootPubKey:            newTreeKemRootPubKey,
				NewRootSignPubKey:        newTreeKemRootSignPubKey,
			},
		}
		chatbotMessage := &pb.ChatbotMessage{
			ChatbotID:        chatbotID,
			MessageWrapper:   messageWrapper,
			UseNormalMessage: false,
		}
		chatbotMessages = append(chatbotMessages, chatbotMessage)
	}

	if hideTrigger {
		logger.Info("Sending message to server-side group ", groupID, " with trigger hidden: ", string(message))
	}

	return chatbotMessages, userUpdate, receivingChatbotIDs, nil
}

/*
DistributeSelfSenderKeyToUserID sends the own sender key to the given userID.
*/
func (csu *ClientSideUser) DistributeSelfSenderKeyToUserID(userID string, groupID string, bounceBack bool) error {
	sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
	if err != nil {
		return err
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

	return csu.SendIndividualMessage(protocol.NewSignalAddress(userID, 1), msgMarshal, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE)
}

/*
DistributeSelfSenderKeyToAll sends the own sender key to all group participants.
*/
func (csu *ClientSideUser) DistributeSelfSenderKeyToAll(groupID string) error {
	sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
	if err != nil {
		return err
	}

	for _, userID := range sessionDriver.GetGroupParticipants() {
		if userID == csu.userID {
			continue
		}
		err = csu.DistributeSelfSenderKeyToUserID(userID, groupID, true)
		if err != nil {
			logger.Error("Error sending sender key distribution message to ", userID, ": ", err)
			return err
		}
	}

	for _, chatbotID := range sessionDriver.GetGroupChatbots() {
		if !sessionDriver.GetChatbotIsIGA(chatbotID) {
			err = csu.DistributeSelfSenderKeyToUserID(chatbotID, groupID, true)
			if err != nil {
				logger.Error("Error sending sender key distribution message to ", chatbotID, ": ", err)
				return err
			}
		}
	}

	return nil
}

/*
CreateAndRegisterServerSidePseudonym creates a pseudonym for the given chatbotID and send it to the chatbot through the IGA channel.
*/
func (csu *ClientSideUser) CreateAndRegisterServerSidePseudonym(groupID string, chatbotID string) error {
	pseudoUserID, signingKeyPub, err := csu.CreatePseudoUser(groupID, pb.GroupType_SERVER_SIDE, chatbotID)
	if err != nil {
		logger.Error("Failed to create pseudo user: ", err)
		return err
	}

	logger.Info("Creating pseudo user for chatbot ", chatbotID, " in group ", groupID)

	pseudonymRegistrationMessage := &pb.PseudonymRegistrationMessage{
		GroupID:       groupID,
		PseudoUserID:  pseudoUserID,
		SigningKeyPub: signingKeyPub,
	}

	// Pack the message.
	pseudonymRegistrationMessageMarshalled, err := proto.Marshal(pseudonymRegistrationMessage)
	if err != nil {
		logger.Error("Failed to encode pseudonym registration message", err)
		return err
	}

	// Get session driver of the group
	sessionDriver, err := csu.Client.GetServerSideGroupSessionDriver(groupID)
	if err != nil {
		logger.Error("Failed to get session driver: ", err)
		return err
	}

	// Update TreeKEM
	userUpdate, chatbotUpdateCiphertexts, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err := sessionDriver.GenerateMultiTreeKEMKeyUpdate([]string{chatbotID})

	// Encrypt the message.
	encryptedMessage := sessionDriver.EncryptMessageByMultiTreeKEMRoot(pseudonymRegistrationMessageMarshalled, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, []string{chatbotID}, chatbotID, nil).Serialize()

	chatbotMessage := &pb.ChatbotMessage{
		ChatbotID: chatbotID,
		MessageWrapper: &pb.MessageWrapper{
			SenderID:         "",
			RecipientID:      groupID,
			EncryptedMessage: encryptedMessage,
			HasPreKey:        false,
			ChatbotIds:       []string{chatbotID},
			IsIGA:            true,
			TreeKEMKeyUpdatePack: &pb.TreeKEMKeyUpdatePack{
				ChatbotUpdateCiphertexts: treekem.ECKEMCipherTextStringMapPbConvert(map[string]treekem.ECKEMCipherText{chatbotID: chatbotUpdateCiphertexts[chatbotID]}),
				NewRootPubKey:            newTreeKemRootPubKey,
				NewRootSignPubKey:        newTreeKemRootSignPubKey,
			},
		},
		UseNormalMessage: false,
	}

	messageWrapper := &pb.MessageWrapper{
		SenderID:         csu.userID,
		RecipientID:      groupID,
		EncryptedMessage: sessionDriver.EncryptMessageBySendingSession(pseudonymRegistrationMessageMarshalled, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, []string{chatbotID}).SignedSerialize(),
		ChatbotMessages:  []*pb.ChatbotMessage{chatbotMessage},
		HasPreKey:        false,
		ChatbotIds:       []string{chatbotID},
		TreeKEMKeyUpdatePack: &pb.TreeKEMKeyUpdatePack{
			UserUpdate: treekem.TreeKEMUserUpdatePbConvert(userUpdate),
		},
	}

	return csu.Client.SendServerSideGroupMessage(groupID, messageWrapper)
}
