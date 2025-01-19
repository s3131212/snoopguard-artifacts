package client

import (
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/treekem"
	"chatbot-poc-go/pkg/util"
	"context"
	"fmt"
	"github.com/s3131212/go-mls"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/libsignal/protocol"
	"google.golang.org/protobuf/proto"
)

/*
MlsGroupSessionDriver handles the server-side group session.
*/
type MlsGroupSessionDriver struct {
	userID            string
	groupID           string
	groupParticipants []string
	groupChatbots     []string
	chatbotIsIGA      map[string]bool
	chatbotIsPseudo   map[string]bool

	groupChatState       *mls.State
	memberToLeafIndex    map[string]uint32
	mlsMultiTree         *treekem.MlsMultiTree
	mlsMultiTreeExternal *treekem.MlsMultiTreeExternal

	sendIndividualMessage func(recipientAddress *protocol.SignalAddress, messageWrapper *pb.MessageWrapper) error

	chatServiceClient    *pb.ChatServiceClient
	chatServiceClientCtx *context.Context
}

/*
NewMlsGroupSessionDriver creates a new MlsGroupSessionDriver.
*/
func NewMlsGroupSessionDriver(userID string, groupID string, groupParticipants []string, groupChatbotIDs []string) *MlsGroupSessionDriver {
	mlsGroupSessionDriver := &MlsGroupSessionDriver{
		userID:            userID,
		groupID:           groupID,
		groupParticipants: groupParticipants,
		groupChatbots:     groupChatbotIDs,
		chatbotIsIGA:      make(map[string]bool),
		chatbotIsPseudo:   make(map[string]bool),
		groupChatState:    nil,
		memberToLeafIndex: make(map[string]uint32),
		sendIndividualMessage: func(recipientAddress *protocol.SignalAddress, messageWrapper *pb.MessageWrapper) error {
			// Print not implemented error
			logger.Error("Not implemented: sendIndividualMessage")
			return fmt.Errorf("not implemented: sendIndividualMessage")
		},
	}

	return mlsGroupSessionDriver
}

/*
SetChatServiceClient inject the chatServiceClient and chatServiceClientCtx into the ServerSideGroupSessionDriver.
*/
func (mgsd *MlsGroupSessionDriver) SetChatServiceClient(chatServiceClient *pb.ChatServiceClient, chatServiceClientCtx *context.Context) {
	mgsd.chatServiceClient = chatServiceClient
	mgsd.chatServiceClientCtx = chatServiceClientCtx
}

/*
SetSendIndividualMessage inject the sendIndividualMessage function into the ServerSideGroupSessionDriver.
*/
func (mgsd *MlsGroupSessionDriver) SetSendIndividualMessage(sendIndividualMessage func(recipientAddress *protocol.SignalAddress, messageWrapper *pb.MessageWrapper) error) {
	mgsd.sendIndividualMessage = sendIndividualMessage
}

/*
SetGroupState sets the group state.
*/
func (mgsd *MlsGroupSessionDriver) SetGroupState(groupChatState *mls.State) {
	mgsd.groupChatState = groupChatState
}

/*
GetGroupState returns the group state. This should be used for debug purpose only.
*/
func (mgsd *MlsGroupSessionDriver) GetGroupState() *mls.State {
	return mgsd.groupChatState
}

/*
GetGroupParticipants returns the group participants.
*/
func (mgsd *MlsGroupSessionDriver) GetGroupParticipants() []string {
	return mgsd.groupParticipants
}

/*
GetGroupChatbots returns the group chatbots.
*/
func (mgsd *MlsGroupSessionDriver) GetGroupChatbots() []string {
	return mgsd.groupChatbots
}

/*
GetChatbotIsIGA returns whether the chatbot has IGA enabled.
*/
func (mgsd *MlsGroupSessionDriver) GetChatbotIsIGA(chatbotId string) bool {
	if _, ok := mgsd.chatbotIsIGA[chatbotId]; !ok {
		return false
	}
	return mgsd.chatbotIsIGA[chatbotId]
}

/*
SetChatbotIsIGA sets whether the chatbot has IGA enabled.
*/
func (mgsd *MlsGroupSessionDriver) SetChatbotIsIGA(chatbotId string, isIGA bool) {
	mgsd.chatbotIsIGA[chatbotId] = isIGA
}

/*
GetChatbotIsPseudo returns whether the chatbot is a pseudo chatbot.
*/
func (mgsd *MlsGroupSessionDriver) GetChatbotIsPseudo(chatbotId string) bool {
	if _, ok := mgsd.chatbotIsPseudo[chatbotId]; !ok {
		return false
	}
	return mgsd.chatbotIsPseudo[chatbotId]
}

/*
SetChatbotIsPseudo sets whether the chatbot is a pseudo chatbot.
*/
func (mgsd *MlsGroupSessionDriver) SetChatbotIsPseudo(chatbotId string, isPsuedo bool) {
	mgsd.chatbotIsPseudo[chatbotId] = isPsuedo
}

/*
SendMessage sends a Message to the server-side group.
*/
func (mgsd *MlsGroupSessionDriver) SendMessage(messageRaw *pb.MessageWrapper) error {
	logger.Debug("Sending Message to MLS group: ", messageRaw.String())

	// Send Message
	res, err := (*mgsd.chatServiceClient).SendMessage(*mgsd.chatServiceClientCtx, messageRaw)
	if err != nil {
		logger.Error("Error sending Message to MLS group: ", err)
		return err
	}

	if res.ErrorMessage != "" {
		logger.Error("Error sending Message to MLS group: ", res.ErrorMessage)
		return fmt.Errorf(res.ErrorMessage)
	}

	logger.Info("Received response from MLS group: ", res.String())
	return nil
}

/*
HandleCommit handles the commit.
*/
func (mgsd *MlsGroupSessionDriver) HandleCommit(commit *mls.MLSPlaintext, senderId string) error {
	nextState, err := mgsd.groupChatState.Handle(commit)
	if err != nil {
		logger.Error("Error handling commit", err)
		return err
	}
	mgsd.groupChatState = nextState

	// Update memberToLeafIndex
	mgsd.memberToLeafIndex[senderId] = commit.Sender.Sender

	return nil
}

/*
EncryptMessage encrypts (protects) the given Message by the sending session.
*/
func (mgsd *MlsGroupSessionDriver) EncryptMessage(messageRaw []byte, messageType pb.MessageType, receivingChatbotIds []string) (*mls.MLSCiphertext, *mls.MLSPlaintext, error) {
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

	ct, err := mgsd.groupChatState.Protect(messageMarshal)
	if err != nil {
		logger.Error("Error protecting Message: ", err)
		return nil, nil, err
	}

	// Commit
	secret := util.RandomBytes(32)
	commit, _, nextState, err := mgsd.groupChatState.Commit(secret)
	if err != nil {
		logger.Error("Error committing secret: ", err)
		return nil, nil, err
	}
	mgsd.groupChatState = nextState

	return ct, commit, nil
}

/*
GetWelcomeMessage returns the welcome Message.
*/
func (mgsd *MlsGroupSessionDriver) GetWelcomeMessage(kp mls.KeyPackage) (*mls.Welcome, *mls.MLSPlaintext, *mls.MLSPlaintext, error) {
	add, err := mgsd.groupChatState.Add(kp)
	if err != nil {
		logger.Error("Error adding KeyPackage to groupChatState: ", err)
		return nil, nil, nil, err
	}
	_, err = mgsd.groupChatState.Handle(add)
	if err != nil {
		logger.Error("Error handling KeyPackage: ", err)
		return nil, nil, nil, err
	}

	secret := util.RandomBytes(32)
	addCommit, welcome, nextState, err := mgsd.groupChatState.Commit(secret)
	if err != nil {
		logger.Error("Error committing secret: ", err)
		return nil, nil, nil, err
	}
	mgsd.groupChatState = nextState

	return welcome, add, addCommit, nil
}

/*
GetRemoveMessage returns the remove Message.
*/
func (mgsd *MlsGroupSessionDriver) GetRemoveMessage(removedID string) (*mls.MLSPlaintext, *mls.MLSPlaintext, error) {
	removedLeafIndex, exist := mgsd.memberToLeafIndex[removedID]
	if !exist {
		logger.Error("RemovedLeafIndex does not exist.")
		return nil, nil, fmt.Errorf("RemovedLeafIndex does not exist.")
	}

	remove, err := mgsd.groupChatState.Remove(mls.LeafIndex(removedLeafIndex))
	if err != nil {
		logger.Error("Error generating remove message from groupChatState: ", err)
		return nil, nil, err
	}

	_, err = mgsd.groupChatState.Handle(remove)
	if err != nil {
		logger.Error("Error handling remove message: ", err)
		return nil, nil, err
	}

	secret := util.RandomBytes(32)
	removeCommit, _, nextState, err := mgsd.groupChatState.Commit(secret)
	if err != nil {
		logger.Error("Error committing secret: ", err)
		return nil, nil, err
	}
	mgsd.groupChatState = nextState

	return remove, removeCommit, nil
}

/*
EncryptMessageByMlsMultiTreeRoot encrypts the given message by the multi-treekem's root. (For IGA)
*/
func (mgsd *MlsGroupSessionDriver) EncryptMessageByMlsMultiTreeRoot(messageRaw []byte, messageType pb.MessageType, externalId string, signPrivKey []byte) util.CipherText {
	//logger.Error("EncryptMessageByMlsMultiTreeRoot is not implemented yet.")
	message := &pb.Message{
		Message:     messageRaw,
		MessageType: messageType,
	}
	messageMarshal, err := proto.Marshal(message)
	if err != nil {
		logger.Error("Error marshalling Message: ", err)
		panic("")
	}

	logger.Info("Encrypting Message: ", messageMarshal, " using key", mgsd.GetMlsMultiTree().GetRootSecret(externalId))

	ct, err := util.Encrypt(messageMarshal, mgsd.GetMlsMultiTree().GetRootSecret(externalId), signPrivKey)
	if err != nil {
		logger.Error("Error encrypting Message: ", err)
		panic("")
	}

	return ct
}

/*
EncryptMessageByMlsMultiTreeExternalRoot encrypts the given message by the multi-treekem's external root. (For IGA)
*/
func (mgsd *MlsGroupSessionDriver) EncryptMessageByMlsMultiTreeExternalRoot(messageRaw []byte, messageType pb.MessageType, signPrivKey []byte) util.CipherText {
	message := &pb.Message{
		Message:     messageRaw,
		MessageType: messageType,
	}
	messageMarshal, err := proto.Marshal(message)
	if err != nil {
		logger.Error("Error marshalling Message: ", err)
		panic("")
	}
	logger.Info("Encrypting Message: ", messageMarshal, " using key", mgsd.GetMlsMultiTreeExternal().GetRootSecret())
	ct, err := util.Encrypt(messageMarshal, mgsd.GetMlsMultiTreeExternal().GetRootSecret(), signPrivKey)
	if err != nil {
		logger.Error("Error encrypting Message: ", err)
		panic("")
	}

	return ct
}

/*
ParseEncryptedMessage parses the given messageRaw and handles it (either do the specific task or output the Message) as well as return the Message.
*/
func (mgsd *MlsGroupSessionDriver) ParseEncryptedMessage(senderID string, encryptedMessage mls.MLSCiphertext) ([]byte, pb.MessageType, []string) {
	decryptedMessage, err := mgsd.groupChatState.Unprotect(&encryptedMessage)
	if err != nil {
		logger.Error("Failed to decrypt the Message", err)
		return nil, -1, nil
	}

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
func (mgsd *MlsGroupSessionDriver) ParseEncryptedIGAMessage(encryptedMessageRaw []byte, signPubKey []byte) ([]byte, pb.MessageType) {
	if mgsd.GetMlsMultiTreeExternal() == nil {
		logger.Error("No MlsMultiTreeExternal")
		return nil, -1
	}

	logger.Info("Decrypting IGA Message: ", encryptedMessageRaw, " using key", mgsd.GetMlsMultiTreeExternal().GetRootSecret())

	decryptedMessage, err := util.Decrypt(util.DeserializeCipherText(encryptedMessageRaw), mgsd.GetMlsMultiTreeExternal().GetRootSecret(), signPubKey)
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

	return nil, -1
}

/*
ParseEncryptedExternalIGAMessage parses the given messageRaw and handles it as an IGA message from the chatbot.
*/
func (mgsd *MlsGroupSessionDriver) ParseEncryptedExternalIGAMessage(encryptedMessageRaw []byte, externalId string) ([]byte, pb.MessageType) {
	if mgsd.GetMlsMultiTree() == nil {
		logger.Error("No MlsMultiTree")
		return nil, -1
	}

	logger.Info("Decrypting IGA Message: ", encryptedMessageRaw, " using key", mgsd.GetMlsMultiTree().GetRootSecret(externalId))
	decryptedMessage, err := util.Decrypt(util.DeserializeCipherText(encryptedMessageRaw), mgsd.GetMlsMultiTree().GetRootSecret(externalId), mgsd.GetMlsMultiTree().GetExternalNode(externalId).SignPublic)
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
AddUser add the user using Add and Commit. It does not update the group participant IDs.
*/
func (mgsd *MlsGroupSessionDriver) AddUser(add *mls.MLSPlaintext, addCommit *mls.MLSPlaintext) error {
	_, err := mgsd.groupChatState.Handle(add)
	if err != nil {
		logger.Error("Error handling add: ", err)
		return err
	}

	nextState, err := mgsd.groupChatState.Handle(addCommit)
	if err != nil {
		logger.Error("Error handling addCommit: ", err)
		return err
	}

	mgsd.groupChatState = nextState

	return nil
}

/*
RemoveUser remove the user using Remove and Commit. It does not update the group participant IDs.
*/
func (mgsd *MlsGroupSessionDriver) RemoveUser(remove *mls.MLSPlaintext, removeCommit *mls.MLSPlaintext) error {
	_, err := mgsd.groupChatState.Handle(remove)
	if err != nil {
		logger.Error("Error handling remove: ", err)
		return err
	}

	nextState, err := mgsd.groupChatState.Handle(removeCommit)
	if err != nil {
		logger.Error("Error handling removeCommit: ", err)
		return err
	}

	mgsd.groupChatState = nextState
	return nil
}

/*
UpdateGroupParticipantIDs updates the group participant IDs.
*/
func (mgsd *MlsGroupSessionDriver) UpdateGroupParticipantIDs(groupParticipantIDs []string) {
	mgsd.groupParticipants = groupParticipantIDs
}

/*
UpdateGroupChatbotIDs updates the group chatbots.
*/
func (mgsd *MlsGroupSessionDriver) UpdateGroupChatbotIDs(groupChatbotIDs []string) {
	mgsd.groupChatbots = groupChatbotIDs
}

/*
InitiateMlsMultiTree initiates the MlsMultiTree.
*/
func (mgsd *MlsGroupSessionDriver) InitiateMlsMultiTree() error {
	mgsd.mlsMultiTree = treekem.NewMlsMultiTree(&mgsd.groupChatState)
	return nil
}

/*
UpdateTreeKEMUserKey handles the TreeKEM key update request.
*/
func (mgsd *MlsGroupSessionDriver) UpdateTreeKEMUserKey(chatbotIds []string) error {
	return mgsd.mlsMultiTree.HandleTreeKEMUpdate(chatbotIds)
}

/*
GenerateMlsMultiTreeKeyUpdate generates the TreeKEM key update request.
*/
func (mgsd *MlsGroupSessionDriver) GenerateMlsMultiTreeKeyUpdate(chatbotIds []string) (map[string]treekem.ECKEMCipherText, []byte, []byte, error) {
	return mgsd.mlsMultiTree.UpdateTreeKEM(chatbotIds)
}

/*
GenerateExternalNodeJoin generates the MlsMultiTreeExternal join request.
*/
func (mgsd *MlsGroupSessionDriver) GenerateExternalNodeJoin(id string) (treekem.ECKEMCipherText, []byte, error) {
	return mgsd.mlsMultiTree.GetExternalNodeJoin(id)
}

/*
GenerateExternalNodeJoinsWithoutUpdate generates the MlsMultiTreeExternal join request for all chatbots.
*/
func (mgsd *MlsGroupSessionDriver) GenerateExternalNodeJoinsWithoutUpdate(pubKey []byte) (map[string][]byte, map[string][]byte, map[string]treekem.ECKEMCipherText, error) {
	return mgsd.mlsMultiTree.GetExternalNodeJoinsWithoutUpdate(pubKey)
}

/*
SetExternalNodeJoinsWithoutUpdate sets the external node join message for all chatbots without updating any existing node.
*/
func (mgsd *MlsGroupSessionDriver) SetExternalNodeJoinsWithoutUpdate(chatbotPubKeys map[string][]byte, chatbotSignPubKeys map[string][]byte, lastTreeKemRootCiphertexts map[string]treekem.ECKEMCipherText) error {
	return mgsd.mlsMultiTree.SetExternalNodeJoinsWithoutUpdate(chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts)
}

/*
AddExternalNodeToMlsMultiTree adds an external node to the MlsMultiTree.
*/
func (mgsd *MlsGroupSessionDriver) AddExternalNodeToMlsMultiTree(id string, ct treekem.ECKEMCipherText) error {
	return mgsd.mlsMultiTree.AddExternalNode(id, ct)
}

/*
GetMlsMultiTree returns the MlsMultiTree.
*/
func (mgsd *MlsGroupSessionDriver) GetMlsMultiTree() *treekem.MlsMultiTree {
	return mgsd.mlsMultiTree
}

/*
InitiateMlsMultiTreeExternal initiates the MlsMultiTreeExternal.
*/
func (mgsd *MlsGroupSessionDriver) InitiateMlsMultiTreeExternal(treekemRootPub []byte, treekemRootSignPub []byte, initLeaf []byte) error {
	mgsd.mlsMultiTreeExternal = treekem.NewMlsMultiTreeExternal(treekemRootPub, treekemRootSignPub, initLeaf)
	return nil
}

/*
GetMlsMultiTreeExternal returns the MlsMultiTreeExternal.
*/
func (mgsd *MlsGroupSessionDriver) GetMlsMultiTreeExternal() *treekem.MlsMultiTreeExternal {
	return mgsd.mlsMultiTreeExternal
}

/*
GenerateMlsMultiTreeExternalKeyUpdate issues a key update from the MlsMultiTreeExternal.
*/
func (mgsd *MlsGroupSessionDriver) GenerateMlsMultiTreeExternalKeyUpdate() (treekem.ECKEMCipherText, []byte, []byte, error) {
	return mgsd.mlsMultiTreeExternal.UpdateExternalNode()
}

/*
HandleMlsMultiTreeExternalKeyUpdate handles the key update from the MlsMultiTreeExternal.
*/
func (mgsd *MlsGroupSessionDriver) HandleMlsMultiTreeExternalKeyUpdate(chatbotId string, chatbotUpdate treekem.ECKEMCipherText, newCbPubKey []byte, newCbSignPubKey []byte) error {
	return mgsd.mlsMultiTree.HandleExternalNodeUpdate(chatbotId, chatbotUpdate, newCbPubKey, newCbSignPubKey)
}

/*
HandleTreeKEMUserKeyUpdate handles the TreeKEM key update request for MlsMultiTree.
*/
func (mgsd *MlsGroupSessionDriver) HandleTreeKEMUserKeyUpdate(updateMessage treekem.ECKEMCipherText, newPubKey []byte, newSignPubKey []byte) error {
	return mgsd.mlsMultiTreeExternal.HandleTreeKEMUpdate(updateMessage, newPubKey, newSignPubKey)
}
