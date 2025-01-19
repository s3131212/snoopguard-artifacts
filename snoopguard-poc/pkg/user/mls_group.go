package user

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
func (csu *ClientSideUser) PostCreateMlsGroup(groupID string, chatbotIsIGA map[string]bool, chatbotIsPseudo map[string]bool, welcome mls.Welcome, keyPackageId uint32) (*client.MlsGroupSessionDriver, error) {
	sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID)
	if err != nil {
		return nil, err
	}

	// If the welcome message is not set, set up the group state from empty
	if welcome.Secrets == nil {
		err = csu.Client.SetMlsGroupStateFromEmpty(groupID, keyPackageId)
	} else {
		err = csu.Client.SetMlsGroupStateFromWelcome(groupID, welcome, keyPackageId)
	}
	if err != nil {
		logger.Error("Failed to set MLS group state: ", err)
		return nil, err
	}

	// Setup MlsMultiTree
	err = sessionDriver.InitiateMlsMultiTree()

	// Set Chatbot IGA status.
	for chatbotID, isIGA := range chatbotIsIGA {
		sessionDriver.SetChatbotIsIGA(chatbotID, isIGA)
	}

	// Set Chatbot Pseudo status.
	for chatbotID, isPseudo := range chatbotIsPseudo {
		sessionDriver.SetChatbotIsPseudo(chatbotID, isPseudo)
	}

	return sessionDriver, nil
}

/*
HandleMlsGroupMessage handles the incoming MLS group message.
*/
func (csu *ClientSideUser) HandleMlsGroupMessage(messageWrapper *pb.MessageWrapper) ([]byte, pb.MessageType) {
	groupId, senderId := messageWrapper.RecipientID, messageWrapper.SenderID

	sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupId)
	if err != nil {
		logger.Error("Received message from Mls fanout group without session ", groupId)
		return nil, -1
	}

	// Todo: Update CMRT

	err = sessionDriver.UpdateTreeKEMUserKey(messageWrapper.GetChatbotIds())
	if err != nil {
		logger.Error("UpdateTreeKEMUserKey failed: ", groupId)
	}

	chatbotKeyUpdatePack := messageWrapper.GetChatbotKeyUpdatePack()
	if chatbotKeyUpdatePack != nil {
		logger.Info("Received MultiTreeKEM update from ", senderId, " for MLS group ", groupId)

		chatbotUpdate := treekem.PbECKEMCipherTextConvert(chatbotKeyUpdatePack.GetChatbotUpdate())
		newCbPubKey := chatbotKeyUpdatePack.GetNewCbPubKey()
		newCbSignPubKey := chatbotKeyUpdatePack.GetNewCbSignPubKey()
		err = sessionDriver.HandleMlsMultiTreeExternalKeyUpdate(senderId, chatbotUpdate, newCbPubKey, newCbSignPubKey)
		if err != nil {
			logger.Error("UpdateMultiTreeKEMExternalKey failed: ", groupId, err)
		}
	}

	if messageWrapper.GetIsIGA() {
		message, messageType := sessionDriver.ParseEncryptedExternalIGAMessage(messageWrapper.EncryptedMessage, messageWrapper.SenderID)
		logger.Info(fmt.Sprintf("Received IGA message in MLS group %v: %v", messageWrapper.RecipientID, string(message)))

		// Validation message always comes from IGA channel.
		if messageType == pb.MessageType_VALIDATION_MESSAGE {
			logger.Info("Received validation message from ", messageWrapper.SenderID, " in MLS group ", messageWrapper.RecipientID, " with content: ", string(message))
		}

		return message, messageType
	} else {
		deserializedCiphertext, err := util.DeserializeMLSCiphertext(messageWrapper.EncryptedMessage)
		if err != nil {
			return nil, -1
		}

		// Forward the message to the MLS group handler.
		message, messageType, receivingChatbotIDs := sessionDriver.ParseEncryptedMessage(senderId, deserializedCiphertext)
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

		err = sessionDriver.UpdateTreeKEMUserKey(receivingChatbotIDs)
		if err != nil {
			logger.Error("UpdateTreeKEMUserKey failed: ", groupId)
		}

		return message, messageType
	}

}

/*
SendMlsGroupMessage sends a message to an MLS group.
*/
func (csu *ClientSideUser) SendMlsGroupMessage(groupID string, messageRaw []byte, messageType pb.MessageType, receivingChatbotIDs []string, hideTrigger bool) error {
	messageWrapper, err := csu.GenerateMlsGroupMessageCipherText(groupID, messageRaw, messageType, receivingChatbotIDs, hideTrigger)
	if err != nil {
		logger.Error(err)
		return err
	}

	return csu.Client.SendMlsGroupMessage(groupID, messageWrapper)
}

/*
GenerateMlsGroupMessageCipherText generate the ciphertext for MLS group without actually sending it.

func (csu *ClientSideUser) GenerateMlsGroupMessageCipherText(groupID string, messageRaw []byte, messageType pb.MessageType, receivingChatbotIDs []string, hideTrigger bool) (*pb.MessageWrapper, error) {
	logger.Info("Sending message to MLS group ", groupID, ": ", string(messageRaw))
	sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	ct, commit, err := sessionDriver.EncryptMessage(messageRaw, messageType)
	if err != nil {
		logger.Error("Failed to encrypt message: ", err)
		return nil, err
	}
	serializedCipherText, err := util.SerializeMLSCiphertext(*ct)
	serializedCommit, err := syntax.Marshal(commit)
	if err != nil {
		return nil, err
	}

	chatbotMessages, receivingChatbotIDs, err := csu.GetMlsChatbotEncryptedMessage(groupID, messageRaw, messageType, receivingChatbotIDs, hideTrigger, serializedCipherText, serializedCommit)
	if err != nil {
		logger.Error("Failed to get chatbot encrypted message: ", err)
		return nil, err
	}

	messageWrapper := &pb.MessageWrapper{
		SenderID:             csu.userID,
		RecipientID:          groupID,
		EncryptedMessage:     serializedCipherText,
		ChatbotMessages:      chatbotMessages,
		HasPreKey:            false,
		ChatbotIds:           receivingChatbotIDs,
		TreeKEMKeyUpdatePack: nil,
		MlsCommit:            serializedCommit,
	}

	return messageWrapper, nil
}
*/

/*
GenerateMlsGroupMessageCipherText generates the ciphertext for MLS group without actually sending it.
*/
func (csu *ClientSideUser) GenerateMlsGroupMessageCipherText(groupID string, messageRaw []byte, messageType pb.MessageType, receivingChatbotIDs []string, hideTrigger bool) (*pb.MessageWrapper, error) {
	logger.Info("Sending message to MLS group ", groupID, ": ", string(messageRaw))
	sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	// Encrypt message for user (note the updated signature to include receivingChatbotIDs)
	ct, commit, err := sessionDriver.EncryptMessage(messageRaw, messageType, receivingChatbotIDs)
	if err != nil {
		logger.Error("Failed to encrypt message: ", err)
		return nil, err
	}

	serializedCipherText, err := util.SerializeMLSCiphertext(*ct)
	if err != nil {
		return nil, err
	}
	serializedCommit, err := syntax.Marshal(commit)
	if err != nil {
		return nil, err
	}

	var chatbotMessages []*pb.ChatbotMessage
	var treeKEMKeyUpdatePackChatbot *pb.TreeKEMKeyUpdatePack
	var actuallySentChatbotIDs []string

	// Determine if any receiving chatbot is IGA or pseudonymous.
	hasIGAOrPseudo := false
	var exampleChatbotID string
	if hideTrigger {
		// When hiding triggers, make sure every chatbot in the group gets a message.
		for _, chatbotID := range sessionDriver.GetGroupChatbots() {
			if sessionDriver.GetChatbotIsIGA(chatbotID) || sessionDriver.GetChatbotIsPseudo(chatbotID) {
				hasIGAOrPseudo = true
				if util.ContainString(chatbotID, receivingChatbotIDs) {
					exampleChatbotID = chatbotID
					break
				}
			}
		}
		// If none of the receiving chatbots is IGA/Pseudonymous, pick one arbitrarily from group chatbots.
		if exampleChatbotID == "" {
			for _, chatbotID := range sessionDriver.GetGroupChatbots() {
				if sessionDriver.GetChatbotIsIGA(chatbotID) || sessionDriver.GetChatbotIsPseudo(chatbotID) {
					exampleChatbotID = chatbotID
					hasIGAOrPseudo = true
					break
				}
			}
		}
		actuallySentChatbotIDs = sessionDriver.GetGroupChatbots()
	} else {
		// Only send to the chatbots that were specified.
		for _, chatbotID := range receivingChatbotIDs {
			if sessionDriver.GetChatbotIsIGA(chatbotID) || sessionDriver.GetChatbotIsPseudo(chatbotID) {
				hasIGAOrPseudo = true
				exampleChatbotID = chatbotID
				break
			}
		}
		actuallySentChatbotIDs = receivingChatbotIDs
	}

	if hasIGAOrPseudo {
		// Generate MLS MultiTree key update for the chatbots (IGA/pseudonymous).
		var chatbotUpdateCiphertexts map[string]treekem.ECKEMCipherText
		var newTreeKemRootPubKey []byte
		var newTreeKemRootSignPubKey []byte

		chatbotUpdateCiphertexts, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err = sessionDriver.GenerateMlsMultiTreeKeyUpdate(receivingChatbotIDs)
		if err != nil {
			logger.Error("Failed to generate MlsMultiTree key update: ", err)
			return nil, err
		}

		// If hiding triggers, add fake update ciphertexts for IGA/pseudonymous chatbots not in the original list.
		if hideTrigger {
			for _, chatbotID := range sessionDriver.GetGroupChatbots() {
				if (sessionDriver.GetChatbotIsIGA(chatbotID) || sessionDriver.GetChatbotIsPseudo(chatbotID)) && !util.ContainString(chatbotID, receivingChatbotIDs) {
					chatbotUpdateCiphertexts[chatbotID] = treekem.ECKEMCipherText{
						CipherText: util.RandomBytes(32),
						IV:         util.RandomBytes(16),
						Public:     util.RandomBytes(32),
					}
				}
			}
		}

		// Encrypt the message (using MLS MultiTree) for IGA/pseudonymous chatbots.
		cipherTextForIGAChatbot := sessionDriver.EncryptMessageByMlsMultiTreeRoot(messageRaw, messageType, exampleChatbotID, nil)

		// Prepare the TreeKEMKeyUpdatePack
		treeKEMKeyUpdatePackChatbot = &pb.TreeKEMKeyUpdatePack{
			ChatbotUpdateCiphertexts: treekem.ECKEMCipherTextStringMapPbConvert(chatbotUpdateCiphertexts),
			NewRootPubKey:            newTreeKemRootPubKey,
			NewRootSignPubKey:        newTreeKemRootSignPubKey,
		}

		// Create chatbot messages for all chatbots in actuallySentChatbotIDs
		for _, chatbotID := range actuallySentChatbotIDs {
			if sessionDriver.GetChatbotIsIGA(chatbotID) || sessionDriver.GetChatbotIsPseudo(chatbotID) {
				var cipherTextTmp []byte
				senderID := csu.userID

				if sessionDriver.GetChatbotIsPseudo(chatbotID) {
					pseudoUser := csu.GetPseudoUser(groupID, chatbotID)
					if pseudoUser == nil {
						logger.Error("Failed to get pseudo user. Register a pseudonym before sending any message.")
						return nil, err
					}
					// For a pseudonymous user, sign the ciphertext.
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
					MlsCommit:            serializedCommit,
				}
				chatbotMessage := &pb.ChatbotMessage{
					ChatbotID:        chatbotID,
					MessageWrapper:   messageWrapper,
					UseNormalMessage: false,
				}
				chatbotMessages = append(chatbotMessages, chatbotMessage)
			}
		}

		// Handle non-IGA/Pseudonymous chatbots if present in the original list.
		var nonIGAchatbots []string
		for _, chatbotID := range receivingChatbotIDs {
			if !sessionDriver.GetChatbotIsIGA(chatbotID) && !sessionDriver.GetChatbotIsPseudo(chatbotID) {
				nonIGAchatbots = append(nonIGAchatbots, chatbotID)
			}
		}
		if len(nonIGAchatbots) > 0 {
			// For non-IGA chatbots, we create a separate message wrapper using the user ciphertext.
			chatbotMessageWrapper := &pb.MessageWrapper{
				SenderID:             csu.userID,
				RecipientID:          groupID,
				EncryptedMessage:     serializedCipherText,
				HasPreKey:            false,
				ChatbotIds:           receivingChatbotIDs,
				IsIGA:                false,
				IsPseudo:             false,
				TreeKEMKeyUpdatePack: treeKEMKeyUpdatePackChatbot,
				MlsCommit:            serializedCommit,
			}
			for _, chatbotID := range nonIGAchatbots {
				chatbotMessages = append(chatbotMessages, &pb.ChatbotMessage{
					ChatbotID:        chatbotID,
					MessageWrapper:   chatbotMessageWrapper,
					UseNormalMessage: false,
				})
			}
		}

		// Handle non-IGA/Pseudonymous chatbots that did not receive the real message (when hideTrigger is true).
		if hideTrigger {
			var nonIGAnonReceivingChatbots []string
			for _, chatbotID := range sessionDriver.GetGroupChatbots() {
				if !util.ContainString(chatbotID, receivingChatbotIDs) && !sessionDriver.GetChatbotIsIGA(chatbotID) && !sessionDriver.GetChatbotIsPseudo(chatbotID) {
					nonIGAnonReceivingChatbots = append(nonIGAnonReceivingChatbots, chatbotID)
				}
			}
			if len(nonIGAnonReceivingChatbots) > 0 {
				// Create a dummy ciphertext for these chatbots.
				dummyMessage := []byte(util.RandomString(len(messageRaw)))
				dummyCT, dummyCommit, err := sessionDriver.EncryptMessage(dummyMessage, pb.MessageType_SKIP, receivingChatbotIDs)
				if err != nil {
					logger.Error("Failed to encrypt dummy message for hiding trigger: ", err)
					return nil, err
				}
				serializedDummyCT, err := util.SerializeMLSCiphertext(*dummyCT)
				if err != nil {
					return nil, err
				}
				dummyCommitSerialized, err := syntax.Marshal(dummyCommit)
				if err != nil {
					return nil, err
				}
				chatbotMessageWrapper := &pb.MessageWrapper{
					SenderID:         csu.userID,
					RecipientID:      groupID,
					EncryptedMessage: serializedDummyCT,
					HasPreKey:        false,
					ChatbotIds:       receivingChatbotIDs,
					IsIGA:            false,
					IsPseudo:         false,
					// No TreeKEM update needed for non-IGA dummy message.
					MlsCommit: dummyCommitSerialized,
				}
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
		// If there is no IGA/Pseudonymous chatbot, simply prepare chatbot messages for each receiving chatbot.
		chatbotMessages = make([]*pb.ChatbotMessage, 0)
		for _, chatbotID := range receivingChatbotIDs {
			chatbotMessageWrapper := &pb.MessageWrapper{
				SenderID:         csu.userID,
				RecipientID:      groupID,
				EncryptedMessage: serializedCipherText,
				ChatbotIds:       []string{chatbotID},
				// No MultiTree update for non-IGA chatbots.
				MlsCommit: serializedCommit,
			}
			chatbotMessages = append(chatbotMessages, &pb.ChatbotMessage{
				ChatbotID:        chatbotID,
				MessageWrapper:   chatbotMessageWrapper,
				UseNormalMessage: false,
			})
		}
		// For non-IGA/Pseudonymous chatbots, if hideTrigger is true, send a dummy message for those not in receivingChatbotIDs.
		if hideTrigger {
			for _, chatbotID := range sessionDriver.GetGroupChatbots() {
				if !util.ContainString(chatbotID, receivingChatbotIDs) {
					dummyMessage := []byte(util.RandomString(len(messageRaw)))
					dummyCT, dummyCommit, err := sessionDriver.EncryptMessage(dummyMessage, pb.MessageType_SKIP, receivingChatbotIDs)
					if err != nil {
						logger.Error("Failed to encrypt dummy message for hiding trigger: ", err)
						return nil, err
					}
					serializedDummyCT, err := util.SerializeMLSCiphertext(*dummyCT)
					if err != nil {
						return nil, err
					}
					dummyCommitSerialized, err := syntax.Marshal(dummyCommit)
					if err != nil {
						return nil, err
					}
					chatbotMessageWrapper := &pb.MessageWrapper{
						SenderID:         csu.userID,
						RecipientID:      groupID,
						EncryptedMessage: serializedDummyCT,
						ChatbotIds:       []string{chatbotID},
						MlsCommit:        dummyCommitSerialized,
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

	// Prepare final MessageWrapper.
	messageWrapper := &pb.MessageWrapper{
		SenderID:         csu.userID,
		RecipientID:      groupID,
		EncryptedMessage: serializedCipherText,
		ChatbotMessages:  chatbotMessages,
		HasPreKey:        false,
		ChatbotIds:       actuallySentChatbotIDs,
		// No user-level MultiTree update exists here so we only attach the MLS commit.
		MlsCommit: serializedCommit,
	}

	logger.Info("MLs messageWrapper: ", messageWrapper)
	return messageWrapper, nil
}

/*
GetMlsChatbotEncryptedMessage returns the encrypted message for the given chatbotID.
*/
func (csu *ClientSideUser) GetMlsChatbotEncryptedMessage(groupID string, message []byte, messageType pb.MessageType, receivingChatbotIDs []string, hideTrigger bool, originalCipherText []byte, originalCommit []byte) ([]*pb.ChatbotMessage, []string, error) {
	sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID)
	if err != nil {
		return nil, nil, err
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

	// Update MlsMultiTree
	var chatbotUpdateCiphertexts map[string]treekem.ECKEMCipherText
	var newTreeKemRootPubKey []byte
	var newTreeKemRootSignPubKey []byte
	for _, chatbotID := range receivingChatbotIDs {
		if sessionDriver.GetChatbotIsIGA(chatbotID) {
			chatbotUpdateCiphertexts, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err = sessionDriver.GenerateMlsMultiTreeKeyUpdate(receivingChatbotIDs)
			if err != nil {
				logger.Error("Failed to generate MlsMultiTree key update: ", err)
				return nil, nil, err
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

		var cipherText []byte
		var senderID string
		if sessionDriver.GetChatbotIsPseudo(chatbotID) {
			pseudoUser := csu.GetPseudoUser(groupID, chatbotID)
			if pseudoUser == nil {
				logger.Error("Failed to get pseudo user. Register a pseudonym before sending any message.")
				return nil, nil, err
			}

			if !sendSkip {
				if encryptedMessageCache.CipherText == nil {
					encryptedMessageCache = sessionDriver.EncryptMessageByMlsMultiTreeRoot(message, messageType, chatbotID, nil)
				}
				sig, err := util.Sign(encryptedMessageCache.CipherText, pseudoUser.SigningKeyPair.Private.Bytes())
				if err != nil {
					logger.Error("Failed to sign message: ", err)
					return nil, nil, err
				}
				cipherText = util.CipherText{
					IV:         encryptedMessageCache.IV,
					CipherText: encryptedMessageCache.CipherText,
					Signature:  sig,
				}.Serialize()
			} else {
				if encryptedSkipMessageCache.CipherText == nil {
					encryptedSkipMessageCache = sessionDriver.EncryptMessageByMlsMultiTreeRoot([]byte(util.RandomString(len(message))), pb.MessageType_SKIP, chatbotID, nil)
				}
				sig, err := util.Sign(encryptedSkipMessageCache.CipherText, pseudoUser.SigningKeyPair.Private.Bytes())
				if err != nil {
					logger.Error("Failed to sign message: ", err)
					return nil, nil, err
				}
				cipherText = util.CipherText{
					IV:         encryptedSkipMessageCache.IV,
					CipherText: encryptedSkipMessageCache.CipherText,
					Signature:  sig,
				}.Serialize()
			}
			senderID = pseudoUser.PseudoUserID
		} else if sessionDriver.GetChatbotIsIGA(chatbotID) {
			if !sendSkip {
				if encryptedMessageCache.CipherText == nil {
					encryptedMessageCache = sessionDriver.EncryptMessageByMlsMultiTreeRoot(message, messageType, chatbotID, nil)
				}
				cipherText = encryptedMessageCache.Serialize()
			} else {
				if encryptedSkipMessageCache.CipherText == nil {
					encryptedSkipMessageCache = sessionDriver.EncryptMessageByMlsMultiTreeRoot([]byte(util.RandomString(len(message))), pb.MessageType_SKIP, chatbotID, nil)
				}
				cipherText = encryptedSkipMessageCache.Serialize()
			}
			senderID = ""
			senderID = ""
		} else {
			cipherText = originalCipherText
			senderID = csu.userID
		}

		messageWrapper := &pb.MessageWrapper{
			SenderID:         senderID,
			RecipientID:      groupID,
			EncryptedMessage: cipherText,
			HasPreKey:        false,
			ChatbotIds:       receivingChatbotIDs,
			IsIGA:            sessionDriver.GetChatbotIsIGA(chatbotID),
			IsPseudo:         sessionDriver.GetChatbotIsPseudo(chatbotID),
			TreeKEMKeyUpdatePack: &pb.TreeKEMKeyUpdatePack{
				ChatbotUpdateCiphertexts: treekem.ECKEMCipherTextStringMapPbConvert(map[string]treekem.ECKEMCipherText{chatbotID: chatbotUpdateCiphertexts[chatbotID]}),
				NewRootPubKey:            newTreeKemRootPubKey,
				NewRootSignPubKey:        newTreeKemRootSignPubKey,
			},
			MlsCommit: originalCommit,
		}
		chatbotMessage := &pb.ChatbotMessage{
			ChatbotID:        chatbotID,
			MessageWrapper:   messageWrapper,
			UseNormalMessage: false,
		}
		chatbotMessages = append(chatbotMessages, chatbotMessage)
	}

	if hideTrigger {
		logger.Info("Sending message to MLS group ", groupID, " with trigger hidden: ", string(message))
	}

	return chatbotMessages, receivingChatbotIDs, nil
}

/*
CreateAndRegisterMlsPseudonym creates a pseudonym for the given chatbotID and send it to the chatbot through the IGA channel.
*/
func (csu *ClientSideUser) CreateAndRegisterMlsPseudonym(groupID string, chatbotID string) error {
	//logger.Error("CreateAndRegisterMlsPseudonym is not implemented yet.")
	pseudoUserID, signingKeyPub, err := csu.CreatePseudoUser(groupID, pb.GroupType_MLS, chatbotID)
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
	sessionDriver, err := csu.Client.GetMlsGroupSessionDriver(groupID)
	if err != nil {
		logger.Error("Failed to get session driver: ", err)
		return err
	}

	// Update TreeKEM
	chatbotUpdateCiphertexts, newTreeKemRootPubKey, newTreeKemRootSignPubKey, err := sessionDriver.GenerateMlsMultiTreeKeyUpdate([]string{chatbotID})

	// Encrypt the message.
	encryptedMessage := sessionDriver.EncryptMessageByMlsMultiTreeRoot(pseudonymRegistrationMessageMarshalled, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, chatbotID, nil).Serialize()

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

	//cipherText := sessionDriver.EncryptMessageByMlsMultiTreeRoot(pseudonymRegistrationMessageMarshalled, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, chatbotID, nil).Serialize()

	messageWrapper := &pb.MessageWrapper{
		SenderID:         csu.userID,
		RecipientID:      groupID,
		EncryptedMessage: nil,
		ChatbotMessages:  []*pb.ChatbotMessage{chatbotMessage},
		HasPreKey:        false,
		ChatbotIds:       []string{chatbotID},
	}

	return csu.Client.SendMlsGroupMessage(groupID, messageWrapper)
}
