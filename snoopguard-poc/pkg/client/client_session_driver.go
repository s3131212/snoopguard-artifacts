package client

import (
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/util"
	"context"
	"fmt"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/libsignal/protocol"
	"google.golang.org/protobuf/proto"
)

/*
ClientSessionDriver handles the client-side session.
*/
type ClientSessionDriver struct {
	userID      string
	recipientID string
	session     *util.SessionWrapper

	chatServiceClient    *pb.ChatServiceClient
	chatServiceClientCtx *context.Context
}

/*
NewClientSessionDriver creates a new ClientSessionDriver.
*/
func NewClientSessionDriver(userID string, recipientID string, session *util.SessionWrapper) *ClientSessionDriver {
	clientSessionDriver := &ClientSessionDriver{}
	clientSessionDriver.userID = userID
	clientSessionDriver.recipientID = recipientID
	clientSessionDriver.session = session

	return clientSessionDriver
}

/*
SetChatServiceClient inject the chatServiceClient and chatServiceClientCtx into the ClientSessionDriver.
*/
func (ClientSessionDriver *ClientSessionDriver) SetChatServiceClient(chatServiceClient *pb.ChatServiceClient, chatServiceClientCtx *context.Context) {
	ClientSessionDriver.chatServiceClient = chatServiceClient
	ClientSessionDriver.chatServiceClientCtx = chatServiceClientCtx
}

func (ClientSessionDriver *ClientSessionDriver) SendMessage(messageWrapper *pb.MessageWrapper) error {
	logger.Debug("Sending individual Message: ", messageWrapper.String())

	// Send Message to server
	res, err := (*ClientSessionDriver.chatServiceClient).SendMessage(*ClientSessionDriver.chatServiceClientCtx, messageWrapper)
	if err != nil {
		logger.Error("Error sending Message to server: ", err)
		return err
	}

	if res.ErrorMessage != "" {
		logger.Error("Error sending Message to server: ", res.ErrorMessage)
		return fmt.Errorf(res.ErrorMessage)
	}
	return nil
}

func (ClientSessionDriver *ClientSessionDriver) EncryptMessage(messageRaw []byte) protocol.CiphertextMessage {
	return ClientSessionDriver.session.EncryptMsg(messageRaw)

}

func (ClientSessionDriver *ClientSessionDriver) DecryptMessage(senderID string, recipientID string, encryptedMessage []byte, hasPreKey bool) ([]byte, pb.MessageType, error) {
	decryptedMessage, err := ClientSessionDriver.session.DecryptMsg(ClientSessionDriver.session.ParseRawMessage(encryptedMessage, hasPreKey))
	if err != nil {
		logger.Error("Error decrypting Message: ", err)
		return nil, -1, err
	}
	logger.Debug("Decrypted Message: ", decryptedMessage)
	message := &pb.Message{}
	if proto.Unmarshal(decryptedMessage, message) != nil {
		logger.Error("Error unmarshalling Message")
		return nil, -1, err
	}

	return message.Message, message.MessageType, nil
}
