package user

import (
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/treekem"
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/libsignal/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
)

/*
SetupChatServiceClient set up the chatServiceClient and chatServiceClientCtx into the ClientSideUser.
Note that this function is not called when the ClientSideUser is created. It needs to be called explicitly.
The close() will be returned. One should defer the close() after calling this function.
*/
func (csu *ClientSideUser) SetupChatServiceClient(addr string) func() error {
	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("did not connect: ", err)
	}

	csu.chatServiceClient = pb.NewChatServiceClient(conn)
	csu.chatServiceClientCtx = context.Background()

	return conn.Close
}

/*
SetupMessageStreamService set up the message stream service and return two channels, one for receiving messages, and one for receiving done signal.
*/
func (csu *ClientSideUser) SetupMessageStreamService() (chan *pb.MessageWrapper, chan bool) {
	// Set up a stream to the server.
	messageStream, err := csu.chatServiceClient.MessageStream(csu.chatServiceClientCtx, &pb.MessageStreamInit{UserID: csu.userID})

	if err != nil {
		logger.Error("MessageStream failed: ", err)
	}

	messageStreamDone := make(chan bool)
	messageStreamChan := make(chan *pb.MessageWrapper)
	go func() {
		for {
			resp, err := messageStream.Recv()
			if err == io.EOF {
				logger.Info("MessageStream EOF received")
				messageStreamDone <- true
				return
			}
			if err != nil {
				//logger.Error("MessageStream cannot receive ", err)
				messageStreamDone <- true
				return
			}
			logger.Info("MessageStream received: ", resp)
			messageStreamChan <- resp
		}
	}()

	return messageStreamChan, messageStreamDone
}

/*
SetupServerEventStreamService set up the server event stream service and return two channels, one for receiving messages, and one for receiving done signal.
*/
func (csu *ClientSideUser) SetupServerEventStreamService() (chan *pb.ServerEvent, chan bool) {
	// Set up a stream to the server.
	serverEventStream, err := csu.chatServiceClient.ServerEventStream(csu.chatServiceClientCtx, &pb.ServerEventStreamInit{UserID: csu.userID})

	if err != nil {
		logger.Error("ServerEventStream failed: ", err)
	}

	serverEventStreamDone := make(chan bool)
	serverEventStreamChan := make(chan *pb.ServerEvent)
	go func() {
		for {
			resp, err := serverEventStream.Recv()
			if err == io.EOF {
				logger.Info("ServerEventStream EOF received")
				serverEventStreamDone <- true
				return
			}
			if err != nil {
				//logger.Error("ServerEventStream cannot receive ", err)
				serverEventStreamDone <- true
				return
			}
			logger.Info("ServerEventStream received: ", resp)
			serverEventStreamChan <- resp
		}
	}()

	return serverEventStreamChan, serverEventStreamDone
}

/*
ListenToStreams for messages and server events. This should be executed in a goroutine.
*/
func (csu *ClientSideUser) ListenToStreams() {
	messageStreamChan, messageStreamDone := csu.SetupMessageStreamService()
	serverEventStreamChan, serverEventStreamDone := csu.SetupServerEventStreamService()

	for {
		select {
		case messageData := <-messageStreamChan:
			output, messageType := csu.ParseMessageWrapper(messageData)
			if output != nil {
				csu.messageChan <- OutputMessage{Message: output, MessageType: messageType}
			}
		case eventData := <-serverEventStreamChan:
			output, eventType := csu.ParseServerEvent(eventData)
			if output != nil {
				csu.messageChan <- OutputMessage{Message: output, EventType: eventType}
			}
		case <-messageStreamDone:
			logger.Info("messageStreamDone received")
		case <-serverEventStreamDone:
			logger.Info("eventStreamDone received")
		case <-csu.deactivateChan:
			logger.Info("Deactivate received")
			csu.messageChan <- OutputMessage{Message: []byte("Deactivate")}
			return
		}
	}
}

/*
ParseMessageWrapper parses the given messageWrapper and handles it.
*/
func (csu *ClientSideUser) ParseMessageWrapper(messageWrapper *pb.MessageWrapper) ([]byte, pb.MessageType) {
	// If recipient ID is not user ID, it should be the server-side group ID or MLS group ID
	if messageWrapper.RecipientID != csu.userID {
		// check if this is a server-side group
		if _, err := csu.Client.GetServerSideGroupSessionDriver(messageWrapper.RecipientID); err == nil {
			return csu.HandleServerSideGroupMessage(messageWrapper)
		}

		// check if this is a MLS group
		if _, err := csu.Client.GetMlsGroupSessionDriver(messageWrapper.RecipientID); err == nil {
			return csu.HandleMlsGroupMessage(messageWrapper)
		}

		logger.Error("Received message with unknown recipient ID: ", messageWrapper.RecipientID)
		return nil, -1
	}

	// If is individual message
	if messageWrapper.RecipientID == csu.userID {
		sessionDriver, err := csu.Client.GetSessionDriver(protocol.NewSignalAddress(messageWrapper.SenderID, 1))
		if err != nil {
			logger.Debug("Received message from user without session ", messageWrapper.SenderID)
			prekeyBundle, err := csu.Client.GetOthersPreKeyBundle(messageWrapper.SenderID)
			if err != nil {
				panic("")
			}
			sessionDriver, err = csu.Client.CreateSessionAndDriver(protocol.NewSignalAddress(messageWrapper.SenderID, 1), prekeyBundle)
			if err != nil {
				panic("")
			}
		}

		message, messageType, err := sessionDriver.DecryptMessage(messageWrapper.SenderID, messageWrapper.RecipientID, messageWrapper.EncryptedMessage, messageWrapper.HasPreKey)
		if err != nil {
			logger.Error("Error decrypting Message: ", err)
			return nil, -1
		}

		switch messageType {
		case pb.MessageType_TEXT_MESSAGE:
			logger.Info(fmt.Sprintf("Received text message from %v: %v", messageWrapper.SenderID, string(message)))
			return message, messageType

		case pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE:
			groupID, bounceBack, err := csu.Client.ParseSenderKeyDistributionMessage(message, messageWrapper.SenderID)
			if err != nil {
				logger.Error("Failed to parse sender key message: ", err)
			}

			if bounceBack {
				err = csu.DistributeSelfSenderKeyToUserID(messageWrapper.SenderID, groupID, false)
				if err != nil {
					logger.Error("Failed to distribute self sender key to user: ", err)
				}
			}
			return message, messageType

		case pb.MessageType_CLIENT_SIDE_GROUP_MESSAGE:
			return csu.HandleClientSideGroupMessage(
				message,
				messageWrapper.SenderID,
				messageWrapper.GetChatbotIds(),
				messageWrapper.GetTreeKEMKeyUpdatePack(),
				messageWrapper.GetChatbotKeyUpdatePack())
		case pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE:
			logger.Info("Received pseudonym registration message without IGA from ", messageWrapper.SenderID, "as a group member.")
			return message, messageType
		}
	}
	return nil, -1
}

/*
ParseServerEventRaw parses the given serverEventRaw and handles it.
*/
func (csu *ClientSideUser) ParseServerEventRaw(serverEventRaw []byte) {
	serverEvent := &pb.ServerEvent{}
	err := proto.Unmarshal(serverEventRaw, serverEvent)
	if err != nil {
		logger.Error("ParseMessage failed: ", err)
		return
	}
	csu.ParseServerEvent(serverEvent)
}

/*
ParseServerEvent parse the incoming server events.
*/
func (csu *ClientSideUser) ParseServerEvent(serverEvent *pb.ServerEvent) ([]byte, pb.ServerEventType) {
	logger.Info("Received server event: ", serverEvent.GetEventType())

	switch serverEvent.GetEventType() {
	case pb.ServerEventType_GROUP_INVITATION:
		csu.JoinGroup(
			serverEvent.GetGroupInvitation().GetGroupID(),
			serverEvent.GetGroupInvitation().GetGroupType(),
			serverEvent.GetGroupInvitation().GetParticipantIDs(),
			serverEvent.GetGroupInvitation().GetChatbotIDs(),
			serverEvent.GetGroupInvitation().GetChatbotIsIGA(),
			serverEvent.GetGroupInvitation().GetChatbotIsPseudo(),
			treekem.PbTreeKEMGroupInitKeyConvert(serverEvent.GetGroupInvitation().GetTreeKEMGroupInitKey()),
			serverEvent.GetGroupInvitation().GetTreeKEMInitLeaf(),
			serverEvent.GetGroupInvitation().GetChatbotPubKeys(),
			serverEvent.GetGroupInvitation().GetChatbotSignPubKeys(),
			treekem.PbECKEMCipherTextStringMapConvert(serverEvent.GetGroupInvitation().GetLastTreeKemRootCiphertexts()),
			serverEvent.GetGroupInvitation().GetMlsWelcomeMessage(),
			serverEvent.GetGroupInvitation().GetMlsKeyPackageID(),
		)
		return []byte(serverEvent.GetGroupInvitation().GetGroupID()), pb.ServerEventType_GROUP_INVITATION
	case pb.ServerEventType_GROUP_ADDITION:
		csu.AddUserToGroup(
			serverEvent.GetGroupAddition().GetGroupID(),
			serverEvent.GetGroupAddition().GetGroupType(),
			serverEvent.GetGroupAddition().GetSenderID(),
			serverEvent.GetGroupAddition().GetAddedID(),
			serverEvent.GetGroupAddition().GetParticipantIDs(),
			treekem.PbTreeKEMUserAddConvert(serverEvent.GetGroupAddition().GetTreeKEMUserAdd()),
			serverEvent.GetGroupAddition().GetMlsUserAdd(),
			serverEvent.GetGroupAddition().GetMlsAddCommit(),
		)
		return []byte(serverEvent.GetGroupAddition().GetGroupID()), pb.ServerEventType_GROUP_ADDITION
	case pb.ServerEventType_GROUP_REMOVAL:
		csu.RemoveUserFromGroup(
			serverEvent.GetGroupRemoval().GetGroupID(),
			serverEvent.GetGroupRemoval().GetGroupType(),
			serverEvent.GetGroupRemoval().GetRemovedID(),
			serverEvent.GetGroupRemoval().GetParticipantIDs(),
			serverEvent.GetGroupRemoval().GetMlsRemove(),
			serverEvent.GetGroupRemoval().GetMlsRemoveCommit(),
		)
		return []byte(serverEvent.GetGroupRemoval().GetGroupID()), pb.ServerEventType_GROUP_REMOVAL
	case pb.ServerEventType_GROUP_CHATBOT_ADDITION:
		csu.AddChatbotToGroup(
			serverEvent.GetGroupChatbotAddition().GetGroupID(),
			serverEvent.GetGroupChatbotAddition().GetGroupType(),
			serverEvent.GetGroupChatbotAddition().GetSenderID(),
			serverEvent.GetGroupChatbotAddition().GetAddedChatbotID(),
			serverEvent.GetGroupChatbotAddition().GetChatbotIDs(),
			serverEvent.GetGroupChatbotAddition().GetIsIGA(),
			serverEvent.GetGroupChatbotAddition().GetIsPseudo(),
			treekem.PbECKEMCipherTextConvert(serverEvent.GetGroupChatbotAddition().GetChatbotCipherText()),
			serverEvent.GetGroupChatbotAddition().GetMlsUserAdd(),
			serverEvent.GetGroupChatbotAddition().GetMlsAddCommit())
		return []byte(serverEvent.GetGroupChatbotAddition().GetGroupID()), pb.ServerEventType_GROUP_CHATBOT_ADDITION
	case pb.ServerEventType_GROUP_CHATBOT_REMOVAL:
		csu.RemoveChatbotFromGroup(
			serverEvent.GetGroupChatbotRemoval().GetGroupID(),
			serverEvent.GetGroupChatbotRemoval().GetGroupType(),
			serverEvent.GetGroupChatbotRemoval().GetRemovedChatbotID(),
			serverEvent.GetGroupChatbotRemoval().GetChatbotIDs())
		return []byte(serverEvent.GetGroupChatbotRemoval().GetGroupID()), pb.ServerEventType_GROUP_CHATBOT_REMOVAL
	}
	return nil, -1
}
