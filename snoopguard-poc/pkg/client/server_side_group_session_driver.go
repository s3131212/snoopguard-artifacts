package client

import (
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/treekem"
	"chatbot-poc-go/pkg/util"
	"context"
	"fmt"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/libsignal/protocol"
	"google.golang.org/protobuf/proto"
)

/*
ServerSideGroupSessionDriver handles the server-side group session.
*/
type ServerSideGroupSessionDriver struct {
	userID            string
	groupID           string
	groupChatHandler  *util.GroupChatServerSideFanout
	groupParticipants []string
	groupChatbots     []string
	chatbotIsIGA      map[string]bool
	chatbotIsPseudo   map[string]bool

	treekemState         *treekem.TreeKEMState
	multiTreeKEM         *treekem.MultiTreeKEM
	multiTreeKEMExternal *treekem.MultiTreeKEMExternal

	sendIndividualMessage func(recipientAddress *protocol.SignalAddress, messageWrapper *pb.MessageWrapper) error

	chatServiceClient    *pb.ChatServiceClient
	chatServiceClientCtx *context.Context
}

/*
NewServerSideGroupSessionDriver creates a new ServerSideGroupSessionDriver.
*/
func NewServerSideGroupSessionDriver(userID string, groupID string, groupChatHandler *util.GroupChatServerSideFanout, groupParticipants []string, groupChatbotIDs []string) *ServerSideGroupSessionDriver {
	serverSideGroupSessionDriver := &ServerSideGroupSessionDriver{
		userID:            userID,
		groupID:           groupID,
		groupChatHandler:  groupChatHandler,
		groupParticipants: groupParticipants,
		groupChatbots:     groupChatbotIDs,
		chatbotIsIGA:      make(map[string]bool),
		chatbotIsPseudo:   make(map[string]bool),
		sendIndividualMessage: func(recipientAddress *protocol.SignalAddress, messageWrapper *pb.MessageWrapper) error {
			// Print not implemented error
			logger.Error("Not implemented: sendIndividualMessage")
			return fmt.Errorf("not implemented: sendIndividualMessage")
		},
	}

	groupChatHandler.CreateSendingGroupSession()

	return serverSideGroupSessionDriver
}

/*
SetChatServiceClient inject the chatServiceClient and chatServiceClientCtx into the ServerSideGroupSessionDriver.
*/
func (ssgsd *ServerSideGroupSessionDriver) SetChatServiceClient(chatServiceClient *pb.ChatServiceClient, chatServiceClientCtx *context.Context) {
	ssgsd.chatServiceClient = chatServiceClient
	ssgsd.chatServiceClientCtx = chatServiceClientCtx
}

/*
SetSendIndividualMessage inject the sendIndividualMessage function into the ServerSideGroupSessionDriver.
*/
func (ssgsd *ServerSideGroupSessionDriver) SetSendIndividualMessage(sendIndividualMessage func(recipientAddress *protocol.SignalAddress, messageWrapper *pb.MessageWrapper) error) {
	ssgsd.sendIndividualMessage = sendIndividualMessage
}

/*
GetGroupParticipants returns the group participants.
*/
func (ssgsd *ServerSideGroupSessionDriver) GetGroupParticipants() []string {
	return ssgsd.groupParticipants
}

/*
GetGroupChatbots returns the group chatbots.
*/
func (ssgsd *ServerSideGroupSessionDriver) GetGroupChatbots() []string {
	return ssgsd.groupChatbots
}

/*
GetChatbotIsIGA returns whether the chatbot has IGA enabled.
*/
func (ssgsd *ServerSideGroupSessionDriver) GetChatbotIsIGA(chatbotId string) bool {
	if _, ok := ssgsd.chatbotIsIGA[chatbotId]; !ok {
		return false
	}
	return ssgsd.chatbotIsIGA[chatbotId]
}

/*
SetChatbotIsIGA sets whether the chatbot has IGA enabled.
*/
func (ssgsd *ServerSideGroupSessionDriver) SetChatbotIsIGA(chatbotId string, isIGA bool) {
	ssgsd.chatbotIsIGA[chatbotId] = isIGA
}

/*
GetChatbotIsPseudo returns whether the chatbot is a pseudo chatbot.
*/
func (ssgsd *ServerSideGroupSessionDriver) GetChatbotIsPseudo(chatbotId string) bool {
	if _, ok := ssgsd.chatbotIsPseudo[chatbotId]; !ok {
		return false
	}
	return ssgsd.chatbotIsPseudo[chatbotId]
}

/*
SetChatbotIsPseudo sets whether the chatbot is a pseudo chatbot.
*/
func (ssgsd *ServerSideGroupSessionDriver) SetChatbotIsPseudo(chatbotId string, isPsuedo bool) {
	ssgsd.chatbotIsPseudo[chatbotId] = isPsuedo
}

/*
HasUserReceivingSession returns if the given user id has already established a receiving session.
*/
func (ssgsd *ServerSideGroupSessionDriver) HasUserReceivingSession(userID string) bool {
	return ssgsd.groupChatHandler.GetReceivingGroupSession(userID) != nil
}

/*
SendMessage sends a Message to the server-side group.
*/
func (ssgsd *ServerSideGroupSessionDriver) SendMessage(messageRaw *pb.MessageWrapper) error {
	logger.Debug("Sending Message to server-side group: ", messageRaw.String())

	// Send Message
	res, err := (*ssgsd.chatServiceClient).SendMessage(*ssgsd.chatServiceClientCtx, messageRaw)
	if err != nil {
		logger.Error("Error sending Message to server-side group: ", err)
		return err
	}

	if res.ErrorMessage != "" {
		logger.Error("Error sending Message to server-side group: ", res.ErrorMessage)
		return fmt.Errorf(res.ErrorMessage)
	}

	logger.Info("Received response from server-side group: ", res.String())
	return nil
}

/*
EncryptMessageBySendingSession encrypts the given Message by the sending session.
*/
func (ssgsd *ServerSideGroupSessionDriver) EncryptMessageBySendingSession(messageRaw []byte, messageType pb.MessageType, receivingChatbotIds []string) protocol.GroupCiphertextMessage {
	message := &pb.Message{
		Message:     messageRaw,
		MessageType: messageType,
		ChatbotIDs:  receivingChatbotIds,
	}
	messageMarshal, err := proto.Marshal(message)
	if err != nil {
		logger.Error("Error marshalling Message: ", err)
		panic("")
	}

	return ssgsd.groupChatHandler.GetSendingGroupSession().EncryptGroupMessage(messageMarshal)
}

/*
EncryptMessageByMultiTreeKEMRoot encrypts the given message by the multi-treekem's root. (For IGA)
*/
func (ssgsd *ServerSideGroupSessionDriver) EncryptMessageByMultiTreeKEMRoot(messageRaw []byte, messageType pb.MessageType, receivingChatbotIds []string, externalId string, signPrivKey []byte) util.CipherText {
	message := &pb.Message{
		Message:     messageRaw,
		MessageType: messageType,
		ChatbotIDs:  receivingChatbotIds,
	}
	messageMarshal, err := proto.Marshal(message)
	if err != nil {
		logger.Error("Error marshalling Message: ", err)
		panic("")
	}

	ct, err := util.Encrypt(messageMarshal, ssgsd.GetMultiTreeKEM().GetRootSecret(externalId), signPrivKey)
	if err != nil {
		logger.Error("Error encrypting Message: ", err)
		panic("")
	}

	return ct
}

/*
EncryptMessageByMultiTreeKEMExternalRoot encrypts the given message by the multi-treekem's external root. (For IGA)
*/
func (ssgsd *ServerSideGroupSessionDriver) EncryptMessageByMultiTreeKEMExternalRoot(messageRaw []byte, messageType pb.MessageType, signPrivKey []byte) util.CipherText {
	message := &pb.Message{
		Message:     messageRaw,
		MessageType: messageType,
	}
	messageMarshal, err := proto.Marshal(message)
	if err != nil {
		logger.Error("Error marshalling Message: ", err)
		panic("")
	}

	ct, err := util.Encrypt(messageMarshal, ssgsd.GetMultiTreeKEMExternal().GetRootSecret(), signPrivKey)
	if err != nil {
		logger.Error("Error encrypting Message: ", err)
		panic("")
	}

	return ct
}

/*
ParseEncryptedMessage parses the given messageRaw and handles it (either do the specific task or output the Message) as well as return the Message.
*/
func (ssgsd *ServerSideGroupSessionDriver) ParseEncryptedMessage(senderID string, encryptedMessage protocol.GroupCiphertextMessage) ([]byte, pb.MessageType, []string) {
	if ssgsd.groupChatHandler.GetReceivingGroupSession(senderID) == nil {
		logger.Error("No receiving group session for senderID: ", senderID)
		return nil, -1, nil
	}

	decryptedMessage := ssgsd.groupChatHandler.GetReceivingGroupSession(senderID).DecryptGroupMessage(encryptedMessage)
	packedMessage := &pb.Message{}

	logger.Debug("Received Message", packedMessage)

	if err := proto.Unmarshal(decryptedMessage, packedMessage); err != nil {
		logger.Error("Failed to decode Message", err)
	}

	return packedMessage.Message, packedMessage.MessageType, packedMessage.ChatbotIDs
}

/*
ParseEncryptedIGAMessage parses the given messageRaw and handles it as an IGA message.
*/
func (ssgsd *ServerSideGroupSessionDriver) ParseEncryptedIGAMessage(encryptedMessageRaw []byte, signPubKey []byte) ([]byte, pb.MessageType) {
	if ssgsd.GetMultiTreeKEMExternal() == nil {
		logger.Error("No MultiTreeKEMExternal")
		return nil, -1
	}

	decryptedMessage, err := util.Decrypt(util.DeserializeCipherText(encryptedMessageRaw), ssgsd.GetMultiTreeKEMExternal().GetRootSecret(), signPubKey)
	if err != nil {
		logger.Error("Failed to decrypt the IGA message", err)
		return nil, -1
	}

	packedMessage := &pb.Message{}

	logger.Debug("Received Message", packedMessage)

	if err := proto.Unmarshal(decryptedMessage, packedMessage); err != nil {
		logger.Error("Failed to decode Message", err)
	}

	if packedMessage.MessageType == pb.MessageType_TEXT_MESSAGE {
		logger.Debug("Parsed text Message: ", packedMessage.Message)
		return packedMessage.Message, packedMessage.MessageType
	} else if packedMessage.MessageType == pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE {
		logger.Debug("Parsed pseudonym registration Message: ", packedMessage.Message)
		return packedMessage.Message, packedMessage.MessageType
	} else if packedMessage.MessageType == pb.MessageType_SKIP {
		logger.Debug("Parsed skipped message.")
		return packedMessage.Message, packedMessage.MessageType
	} else {
		logger.Error("Unknown Message type: ", packedMessage.MessageType)
		return nil, -1
	}
}

/*
ParseEncryptedExternalIGAMessage parses the given messageRaw and handles it as an IGA message from the chatbot.
*/
func (ssgsd *ServerSideGroupSessionDriver) ParseEncryptedExternalIGAMessage(encryptedMessageRaw []byte, externalId string) ([]byte, pb.MessageType) {
	if ssgsd.GetMultiTreeKEM() == nil {
		logger.Error("No MultiTreeKEM")
		return nil, -1
	}

	decryptedMessage, err := util.Decrypt(util.DeserializeCipherText(encryptedMessageRaw), ssgsd.GetMultiTreeKEM().GetRootSecret(externalId), ssgsd.GetMultiTreeKEM().GetExternalNode(externalId).SignPublic)
	if err != nil {
		logger.Error("Failed to decrypt the IGA message", err)
		return nil, -1
	}

	packedMessage := &pb.Message{}

	logger.Debug("Received Message", packedMessage)

	if err := proto.Unmarshal(decryptedMessage, packedMessage); err != nil {
		logger.Error("Failed to decode Message", err)
	}

	return packedMessage.Message, packedMessage.MessageType
}

/*
AddSenderKey adds the given senderKey to the server-side group session.
*/
func (ssgsd *ServerSideGroupSessionDriver) AddSenderKey(senderID string, senderKey *protocol.SenderKeyDistributionMessage) {
	ssgsd.groupChatHandler.CreateReceivingGroupSession(senderID, senderKey)
}

/*
GetSelfSenderKey returns the user's sender key of the server-side group session.
*/
func (ssgsd *ServerSideGroupSessionDriver) GetSelfSenderKey() *protocol.SenderKeyDistributionMessage {
	return ssgsd.groupChatHandler.GetSendingGroupSession().DistributeSenderKey()
}

/*
UpdateGroupParticipantIDs updates the group participant IDs.
*/
func (ssgsd *ServerSideGroupSessionDriver) UpdateGroupParticipantIDs(groupParticipantIDs []string) {
	ssgsd.groupParticipants = groupParticipantIDs
}

/*
UpdateGroupChatbotIDs updates the group chatbots.
*/
func (ssgsd *ServerSideGroupSessionDriver) UpdateGroupChatbotIDs(groupChatbotIDs []string) {
	ssgsd.groupChatbots = groupChatbotIDs
}

/*
RemoveUserSession removes user's session.
*/
func (ssgsd *ServerSideGroupSessionDriver) RemoveUserSession(removedID string) {
	ssgsd.groupChatHandler.RemoveReceivingGroupSession(removedID)
}

/*
InitiateTreeKEM initiates the TreeKEM.
*/
func (ssgsd *ServerSideGroupSessionDriver) InitiateTreeKEM(gik treekem.GroupInitKey, initLeaf []byte) error {
	logger.Info("Initiating TreeKEM for group: ", ssgsd.groupID)

	if gik.Size != 0 {
		// Reconstruct an existing TreeKEM
		logger.Info("Reconstructing TreeKEM from GroupInitKey")
		joiner, err := treekem.TreeKEMStateFromUserAdd(initLeaf, gik)
		if err != nil {
			logger.Error("Error initiating TreeKEM: ", err)
			return err
		}
		ssgsd.treekemState = joiner
	} else {
		// Create a new TreeKEM
		logger.Info("Creating new TreeKEM")
		ssgsd.treekemState = treekem.TreeKEMStateOneMemberGroup(initLeaf)
	}
	return nil
}

/*
InitiateMultiTreeKEM initiates the MultiTreeKEM.
*/
func (ssgsd *ServerSideGroupSessionDriver) InitiateMultiTreeKEM() error {
	if ssgsd.treekemState == nil {
		return fmt.Errorf("treekemState is nil")
	}
	ssgsd.multiTreeKEM = treekem.NewMultiTreeKEM(ssgsd.treekemState)
	return nil
}

/*
InviteUserToTreeKEM generates a UserAdd.
*/
func (ssgsd *ServerSideGroupSessionDriver) InviteUserToTreeKEM() (*treekem.UserAdd, error) {
	leaf, err := treekem.GenerateRandomBytes(32)
	if err != nil {
		return nil, err
	}

	gik := ssgsd.treekemState.GroupInitKey()
	ua, err := treekem.TreeKEMStateJoin(leaf, gik)
	if err != nil {
		return nil, err
	}

	return &ua, nil
}

/*
AddUserToTreeKEM adds a user to the TreeKEM.
*/
func (ssgsd *ServerSideGroupSessionDriver) AddUserToTreeKEM(userAdd *treekem.UserAdd) {
	ssgsd.treekemState.HandleUserAdd(*userAdd)
}

/*
UpdateTreeKEMUserKey handles the TreeKEM key update request.
*/
func (ssgsd *ServerSideGroupSessionDriver) UpdateTreeKEMUserKey(userUpdate *treekem.UserUpdate, chatbotIds []string) error {
	return ssgsd.multiTreeKEM.HandleTreeKEMUpdate(userUpdate, chatbotIds)
}

/*
GenerateMultiTreeKEMKeyUpdate generates the TreeKEM key update request.
*/
func (ssgsd *ServerSideGroupSessionDriver) GenerateMultiTreeKEMKeyUpdate(chatbotIds []string) (*treekem.UserUpdate, map[string]treekem.ECKEMCipherText, []byte, []byte, error) {
	return ssgsd.multiTreeKEM.UpdateTreeKEM(chatbotIds)
}

/*
GenerateExternalNodeJoin generates the MultiTreeKEMExternal join request.
*/
func (ssgsd *ServerSideGroupSessionDriver) GenerateExternalNodeJoin(id string) (treekem.ECKEMCipherText, []byte, error) {
	return ssgsd.multiTreeKEM.GetExternalNodeJoin(id)
}

/*
GenerateExternalNodeJoinsWithoutUpdate generates the MultiTreeKEMExternal join request for all chatbots.
*/
func (ssgsd *ServerSideGroupSessionDriver) GenerateExternalNodeJoinsWithoutUpdate(pubKey []byte) (map[string][]byte, map[string][]byte, map[string]treekem.ECKEMCipherText, error) {
	return ssgsd.multiTreeKEM.GetExternalNodeJoinsWithoutUpdate(pubKey)
}

/*
SetExternalNodeJoinsWithoutUpdate sets the external node join message for all chatbots without updating any existing node.
*/
func (ssgsd *ServerSideGroupSessionDriver) SetExternalNodeJoinsWithoutUpdate(chatbotPubKeys map[string][]byte, chatbotSignPubKeys map[string][]byte, lastTreeKemRootCiphertexts map[string]treekem.ECKEMCipherText) error {
	return ssgsd.multiTreeKEM.SetExternalNodeJoinsWithoutUpdate(chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts)
}

/*
AddExternalNodeToMultiTreeKEM adds an external node to the MultiTreeKEM.
*/
func (ssgsd *ServerSideGroupSessionDriver) AddExternalNodeToMultiTreeKEM(id string, ct treekem.ECKEMCipherText) error {
	return ssgsd.multiTreeKEM.AddExternalNode(id, ct)
}

/*
GetTreeKEMState returns the TreeKEM state.
*/
func (ssgsd *ServerSideGroupSessionDriver) GetTreeKEMState() *treekem.TreeKEMState {
	return ssgsd.treekemState
}

/*
GetMultiTreeKEM returns the MultiTreeKEM.
*/
func (ssgsd *ServerSideGroupSessionDriver) GetMultiTreeKEM() *treekem.MultiTreeKEM {
	return ssgsd.multiTreeKEM
}

/*
InitiateMultiTreeKEMExternal initiates the MultiTreeKEMExternal.
*/
func (ssgsd *ServerSideGroupSessionDriver) InitiateMultiTreeKEMExternal(treekemRootPub []byte, treekemRootSignPub []byte, initLeaf []byte) error {
	ssgsd.multiTreeKEMExternal = treekem.NewMultiTreeKEMExternal(treekemRootPub, treekemRootSignPub, initLeaf)
	return nil
}

/*
GetMultiTreeKEMExternal returns the MultiTreeKEMExternal.
*/
func (ssgsd *ServerSideGroupSessionDriver) GetMultiTreeKEMExternal() *treekem.MultiTreeKEMExternal {
	return ssgsd.multiTreeKEMExternal
}

/*
GenerateMultiTreeKEMExternalKeyUpdate issues a key update from the MultiTreeKEMExternal.
*/
func (ssgsd *ServerSideGroupSessionDriver) GenerateMultiTreeKEMExternalKeyUpdate() (treekem.ECKEMCipherText, []byte, []byte, error) {
	return ssgsd.multiTreeKEMExternal.UpdateExternalNode()
}

/*
HandleMultiTreeKEMExternalKeyUpdate handles the key update from the MultiTreeKEMExternal.
*/
func (ssgsd *ServerSideGroupSessionDriver) HandleMultiTreeKEMExternalKeyUpdate(chatbotId string, chatbotUpdate treekem.ECKEMCipherText, newCbPubKey []byte, newCbSignPubKey []byte) error {
	return ssgsd.multiTreeKEM.HandleExternalNodeUpdate(chatbotId, chatbotUpdate, newCbPubKey, newCbSignPubKey)
}

/*
HandleTreeKEMUserKeyUpdate handles the TreeKEM key update request for MultiTreeKEM.
*/
func (ssgsd *ServerSideGroupSessionDriver) HandleTreeKEMUserKeyUpdate(updateMessage treekem.ECKEMCipherText, newPubKey []byte, newSignPubKey []byte) error {
	return ssgsd.multiTreeKEMExternal.HandleTreeKEMUpdate(updateMessage, newPubKey, newSignPubKey)
}
