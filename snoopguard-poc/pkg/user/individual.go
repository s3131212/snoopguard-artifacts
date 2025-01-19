package user

import (
	"chatbot-poc-go/pkg/client"
	pb "chatbot-poc-go/pkg/protos/services"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/libsignal/protocol"
	"google.golang.org/protobuf/proto"
)

/*
CreateIndividualSession creates a session with a recipient.
*/
func (csu *ClientSideUser) CreateIndividualSession(recipientAddress *protocol.SignalAddress) (*client.ClientSessionDriver, error) {
	preKeyBundle, err := csu.Client.GetOthersPreKeyBundle(recipientAddress.Name())
	if err != nil {
		return nil, err
	}

	return csu.Client.CreateSessionAndDriver(recipientAddress, preKeyBundle)
}

/*
SendIndividualMessage send a message to a recipient given that the client session is already established.
*/
func (csu *ClientSideUser) SendIndividualMessage(recipientAddress *protocol.SignalAddress, message []byte, messageType pb.MessageType) error {
	logger.Debug("Sending individual message: ", string(message))
	packedMessage := &pb.Message{
		Message:     message,
		MessageType: messageType,
	}

	packedMessageMarshal, err := proto.Marshal(packedMessage)
	if err != nil {
		logger.Error(err)
		return err
	}
	sessionDriver, err := csu.Client.GetSessionDriver(recipientAddress)
	if err != nil {
		sessionDriver, err = csu.CreateIndividualSession(recipientAddress)
		if err != nil {
			logger.Error("Error creating session with ", recipientAddress.Name(), ": ", err)
			return err
		}
	}
	encryptedMessage := sessionDriver.EncryptMessage(packedMessageMarshal)
	packedMessageWrapper := &pb.MessageWrapper{
		SenderID:         csu.userID,
		RecipientID:      recipientAddress.Name(),
		EncryptedMessage: encryptedMessage.Serialize(),
		HasPreKey:        encryptedMessage.Type() == protocol.PREKEY_TYPE,
	}

	logger.Debug("Sending message to server: ", packedMessageWrapper.String())

	return csu.Client.SendIndividualMessage(recipientAddress, packedMessageWrapper)
}

/*
SendIndividualMessageBenchmark send a message to a recipient given that the client session is already established.
*/
func (csu *ClientSideUser) SendIndividualMessageBenchmark(recipientAddress *protocol.SignalAddress, message []byte, messageType pb.MessageType) (*pb.MessageWrapper, error) {
	logger.Debug("Sending individual message: ", string(message))
	packedMessage := &pb.Message{
		Message:     message,
		MessageType: messageType,
	}

	packedMessageMarshal, err := proto.Marshal(packedMessage)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	sessionDriver, err := csu.Client.GetSessionDriver(recipientAddress)
	if err != nil {
		sessionDriver, err = csu.CreateIndividualSession(recipientAddress)
		if err != nil {
			logger.Error("Error creating session with ", recipientAddress.Name(), ": ", err)
			return nil, err
		}
	}
	encryptedMessage := sessionDriver.EncryptMessage(packedMessageMarshal)
	packedMessageWrapper := &pb.MessageWrapper{
		SenderID:         csu.userID,
		RecipientID:      recipientAddress.Name(),
		EncryptedMessage: encryptedMessage.Serialize(),
		HasPreKey:        encryptedMessage.Type() == protocol.PREKEY_TYPE,
	}

	logger.Debug("Sending message to server: ", packedMessageWrapper.String())

	return packedMessageWrapper, nil
}
