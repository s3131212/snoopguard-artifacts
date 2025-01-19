package client

import (
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/treekem"
	"chatbot-poc-go/pkg/util"
	"context"
	"fmt"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/libsignal/protocol"
)

type ClientSideGroupSessionDriver struct {
	userID            string
	groupID           string
	groupChatHandler  *util.GroupChatClientSideFanout
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
NewClientSideGroupSessionDriver creates a new ClientSideGroupSessionDriver.
*/
func NewClientSideGroupSessionDriver(userID string, groupID string, groupChatHandler *util.GroupChatClientSideFanout, groupParticipants []string, groupChatbots []string) *ClientSideGroupSessionDriver {
	clientSideGroupSessionDriver := &ClientSideGroupSessionDriver{
		userID:            userID,
		groupID:           groupID,
		groupChatHandler:  groupChatHandler,
		groupParticipants: groupParticipants,
		groupChatbots:     groupChatbots,
		chatbotIsIGA:      make(map[string]bool),
		chatbotIsPseudo:   make(map[string]bool),
		sendIndividualMessage: func(recipientAddress *protocol.SignalAddress, messageWrapper *pb.MessageWrapper) error {
			// Print not implemented error
			logger.Error("Not implemented: sendIndividualMessage")
			return fmt.Errorf("not implemented: sendIndividualMessage")
		},
	}

	return clientSideGroupSessionDriver
}

/*
SetChatServiceClient inject the chatServiceClient and chatServiceClientCtx into the ClientSideGroupSessionDriver.
*/
func (csgsd *ClientSideGroupSessionDriver) SetChatServiceClient(chatServiceClient *pb.ChatServiceClient, chatServiceClientCtx *context.Context) {
	csgsd.chatServiceClient = chatServiceClient
	csgsd.chatServiceClientCtx = chatServiceClientCtx
}

/*
SetSendIndividualMessage inject the sendIndividualMessage function into the ClientSideGroupSessionDriver.
*/
func (csgsd *ClientSideGroupSessionDriver) SetSendIndividualMessage(sendIndividualMessage func(recipientAddress *protocol.SignalAddress, messageWrapper *pb.MessageWrapper) error) {
	csgsd.sendIndividualMessage = sendIndividualMessage
}

/*
GetGroupParticipants returns the group participants.
*/
func (csgsd *ClientSideGroupSessionDriver) GetGroupParticipants() []string {
	return csgsd.groupParticipants
}

/*
GetGroupChatbots returns the group chatbots.
*/
func (csgsd *ClientSideGroupSessionDriver) GetGroupChatbots() []string {
	return csgsd.groupChatbots
}

/*
SetChatbotIsIGA sets whether the chatbot has IGA enabled.
*/
func (csgsd *ClientSideGroupSessionDriver) SetChatbotIsIGA(chatbotId string, isIGA bool) {
	csgsd.chatbotIsIGA[chatbotId] = isIGA
}

/*
GetChatbotIsPseudo returns whether the chatbot is pseudo.
*/
func (csgsd *ClientSideGroupSessionDriver) GetChatbotIsPseudo(chatbotId string) bool {
	if _, exist := csgsd.chatbotIsPseudo[chatbotId]; !exist {
		return false
	}
	return csgsd.chatbotIsPseudo[chatbotId]
}

/*
SetChatbotIsPseudo sets whether the chatbot is pseudo.
*/
func (csgsd *ClientSideGroupSessionDriver) SetChatbotIsPseudo(chatbotId string, isPseudo bool) {
	csgsd.chatbotIsPseudo[chatbotId] = isPseudo
}

/*
SendMessage creates an ClientSideGroupMessage and sends it using sendIndividualMessage.
*/
func (csgsd *ClientSideGroupSessionDriver) SendMessage(messages map[string]*pb.MessageWrapper) error {
	// Send it to all participants
	for _, pid := range csgsd.groupParticipants {
		if pid == csgsd.userID {
			continue
		}

		message, exist := messages[pid]
		if !exist {
			continue
		}

		// Create SignalAddress
		recipientAddress := protocol.NewSignalAddress(pid, 1)

		// Send Message
		err := csgsd.sendIndividualMessage(recipientAddress, message)
		if err != nil {
			return err
		}
	}

	// Send it to all chatbots
	for _, pid := range csgsd.groupChatbots {
		if pid == csgsd.userID {
			continue
		}

		message, exist := messages[pid]
		if !exist {
			continue
		}

		// Create SignalAddress
		recipientAddress := protocol.NewSignalAddress(pid, 1)

		// Send Message
		err := csgsd.sendIndividualMessage(recipientAddress, message)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
ParseDecryptedMessage parse the incoming Message.
*/
func (csgsd *ClientSideGroupSessionDriver) ParseDecryptedMessage(message *pb.ClientSideGroupMessage) ([]byte, pb.MessageType) {
	logger.Debug("Received Message to client-side group: ", message.Message, message.MessageType)

	return message.Message, message.MessageType
}

/*
UpdateGroupParticipantIDs updates the group participant IDs.
*/
func (csgsd *ClientSideGroupSessionDriver) UpdateGroupParticipantIDs(groupParticipantIDs []string) {
	csgsd.groupParticipants = groupParticipantIDs
}

/*
UpdateGroupChatbotIDs updates the group chatbots.
*/
func (csgsd *ClientSideGroupSessionDriver) UpdateGroupChatbotIDs(groupChatbotIDs []string) {
	csgsd.groupChatbots = groupChatbotIDs
}

/*
RemoveUserSession removes user's session.
*/
func (csgsd *ClientSideGroupSessionDriver) RemoveUserSession(removedID string) {
	// NOP
}

/*
InitiateTreeKEM initiates the TreeKEM.
*/
func (csgsd *ClientSideGroupSessionDriver) InitiateTreeKEM(gik treekem.GroupInitKey, initLeaf []byte) error {
	logger.Info("Initiating TreeKEM for group: ", csgsd.groupID)

	if gik.Size != 0 {
		// Reconstruct an existing TreeKEM
		logger.Info("Reconstructing TreeKEM from GroupInitKey")
		joiner, err := treekem.TreeKEMStateFromUserAdd(initLeaf, gik)
		if err != nil {
			logger.Error("Error initiating TreeKEM: ", err)
			return err
		}
		csgsd.treekemState = joiner
	} else {
		// Create a new TreeKEM
		logger.Info("Creating new TreeKEM")
		csgsd.treekemState = treekem.TreeKEMStateOneMemberGroup(initLeaf)
	}
	return nil
}

/*
InitiateMultiTreeKEM initiates the MultiTreeKEM.
*/
func (csgsd *ClientSideGroupSessionDriver) InitiateMultiTreeKEM() error {
	if csgsd.treekemState == nil {
		return fmt.Errorf("treekemState is nil")
	}
	csgsd.multiTreeKEM = treekem.NewMultiTreeKEM(csgsd.treekemState)
	return nil
}

/*
InviteUserToTreeKEM generates a UserAdd.
*/
func (csgsd *ClientSideGroupSessionDriver) InviteUserToTreeKEM() (*treekem.UserAdd, error) {
	leaf, err := treekem.GenerateRandomBytes(32)
	if err != nil {
		return nil, err
	}

	gik := csgsd.treekemState.GroupInitKey()
	ua, err := treekem.TreeKEMStateJoin(leaf, gik)
	if err != nil {
		return nil, err
	}

	return &ua, nil
}

/*
AddUserToTreeKEM adds a user to the TreeKEM.
*/
func (csgsd *ClientSideGroupSessionDriver) AddUserToTreeKEM(userAdd *treekem.UserAdd) {
	csgsd.treekemState.HandleUserAdd(*userAdd)
}

/*
UpdateTreeKEMUserKey handles the TreeKEM key update request.
*/
func (csgsd *ClientSideGroupSessionDriver) UpdateTreeKEMUserKey(userUpdate *treekem.UserUpdate, chatbotIds []string) error {
	return csgsd.multiTreeKEM.HandleTreeKEMUpdate(userUpdate, chatbotIds)
}

/*
GenerateMultiTreeKEMKeyUpdate generates the TreeKEM key update request.
*/
func (csgsd *ClientSideGroupSessionDriver) GenerateMultiTreeKEMKeyUpdate(chatbotIds []string) (*treekem.UserUpdate, map[string]treekem.ECKEMCipherText, []byte, []byte, error) {
	return csgsd.multiTreeKEM.UpdateTreeKEM(chatbotIds)
}

/*
GenerateExternalNodeJoin generates the MultiTreeKEMExternal join request.
*/
func (csgsd *ClientSideGroupSessionDriver) GenerateExternalNodeJoin(id string) (treekem.ECKEMCipherText, []byte, error) {
	return csgsd.multiTreeKEM.GetExternalNodeJoin(id)
}

/*
GenerateExternalNodeJoinsWithoutUpdate generates the MultiTreeKEMExternal join request for all chatbots.
*/
func (csgsd *ClientSideGroupSessionDriver) GenerateExternalNodeJoinsWithoutUpdate(pubKey []byte) (map[string][]byte, map[string][]byte, map[string]treekem.ECKEMCipherText, error) {
	return csgsd.multiTreeKEM.GetExternalNodeJoinsWithoutUpdate(pubKey)
}

/*
SetExternalNodeJoinsWithoutUpdate sets the external node join message for all chatbots without updating any existing node.
*/
func (csgsd *ClientSideGroupSessionDriver) SetExternalNodeJoinsWithoutUpdate(chatbotPubKeys map[string][]byte, chatbotSignPubKeys map[string][]byte, lastTreeKemRootCiphertexts map[string]treekem.ECKEMCipherText) error {
	return csgsd.multiTreeKEM.SetExternalNodeJoinsWithoutUpdate(chatbotPubKeys, chatbotSignPubKeys, lastTreeKemRootCiphertexts)
}

/*
AddExternalNodeToMultiTreeKEM adds an external node to the MultiTreeKEM.
*/
func (csgsd *ClientSideGroupSessionDriver) AddExternalNodeToMultiTreeKEM(id string, ct treekem.ECKEMCipherText) error {
	return csgsd.multiTreeKEM.AddExternalNode(id, ct)
}

/*
GetTreeKEMState returns the TreeKEM state.
*/
func (csgsd *ClientSideGroupSessionDriver) GetTreeKEMState() *treekem.TreeKEMState {
	return csgsd.treekemState
}

/*
GetMultiTreeKEM returns the MultiTreeKEM.
*/
func (csgsd *ClientSideGroupSessionDriver) GetMultiTreeKEM() *treekem.MultiTreeKEM {
	return csgsd.multiTreeKEM
}

/*
InitiateMultiTreeKEMExternal initiates the MultiTreeKEMExternal.
*/
func (csgsd *ClientSideGroupSessionDriver) InitiateMultiTreeKEMExternal(treekemRootPub []byte, treekemRootSignPub []byte, initLeaf []byte) error {
	csgsd.multiTreeKEMExternal = treekem.NewMultiTreeKEMExternal(treekemRootPub, treekemRootSignPub, initLeaf)
	return nil
}

/*
GetMultiTreeKEMExternal returns the MultiTreeKEMExternal.
*/
func (csgsd *ClientSideGroupSessionDriver) GetMultiTreeKEMExternal() *treekem.MultiTreeKEMExternal {
	return csgsd.multiTreeKEMExternal
}

/*
GenerateMultiTreeKEMExternalKeyUpdate issues a key update from the MultiTreeKEMExternal.
*/
func (csgsd *ClientSideGroupSessionDriver) GenerateMultiTreeKEMExternalKeyUpdate() (treekem.ECKEMCipherText, []byte, []byte, error) {
	return csgsd.multiTreeKEMExternal.UpdateExternalNode()
}

/*
HandleMultiTreeKEMExternalKeyUpdate handles the key update from the MultiTreeKEMExternal.
*/
func (csgsd *ClientSideGroupSessionDriver) HandleMultiTreeKEMExternalKeyUpdate(chatbotId string, chatbotUpdate treekem.ECKEMCipherText, newCbPubKey []byte, newCbSignPubKey []byte) error {
	return csgsd.multiTreeKEM.HandleExternalNodeUpdate(chatbotId, chatbotUpdate, newCbPubKey, newCbSignPubKey)
}

/*
HandleTreeKEMUserKeyUpdate handles the TreeKEM key update request for MultiTreeKEM.
*/
func (csgsd *ClientSideGroupSessionDriver) HandleTreeKEMUserKeyUpdate(updateMessage treekem.ECKEMCipherText, newPubKey []byte, newSignPubKey []byte) error {
	return csgsd.multiTreeKEMExternal.HandleTreeKEMUpdate(updateMessage, newPubKey, newSignPubKey)
}
