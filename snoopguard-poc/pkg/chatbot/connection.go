package chatbot

import (
	pb "chatbot-poc-go/pkg/protos/services"
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
SetupChatServiceClient set up the chatServiceClient and chatServiceClientCtx into the ClientSideChatbot.
Note that this function is not called when the ClientSideChatbot is created. It needs to be called explicitly.
The close() will be returned. One should defer the close() after calling this function.
*/
func (csc *ClientSideChatbot) SetupChatServiceClient(addr string) func() error {
	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("did not connect: ", err)
	}

	csc.chatServiceClient = pb.NewChatServiceClient(conn)
	csc.chatServiceClientCtx = context.Background()

	return conn.Close
}

/*
SetupMessageStreamService set up the message stream service and return two channels, one for receiving messages, and one for receiving done signal.
*/
func (csc *ClientSideChatbot) SetupMessageStreamService() (chan *pb.MessageWrapper, chan bool) {
	// Set up a stream to the server.
	messageStream, err := csc.chatServiceClient.MessageStream(csc.chatServiceClientCtx, &pb.MessageStreamInit{UserID: csc.chatbotID})

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
func (csc *ClientSideChatbot) SetupServerEventStreamService() (chan *pb.ServerEvent, chan bool) {
	// Set up a stream to the server.
	serverEventStream, err := csc.chatServiceClient.ServerEventStream(csc.chatServiceClientCtx, &pb.ServerEventStreamInit{UserID: csc.chatbotID})

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
func (csc *ClientSideChatbot) ListenToStreams() {
	messageStreamChan, messageStreamDone := csc.SetupMessageStreamService()
	serverEventStreamChan, serverEventStreamDone := csc.SetupServerEventStreamService()

	for {
		select {
		case messageData := <-messageStreamChan:
			output, messageType := csc.ParseMessageWrapper(messageData)
			if output != nil {
				csc.messageChan <- OutputMessage{Message: output, MessageType: messageType}
			}
		case eventData := <-serverEventStreamChan:
			output, eventType := csc.ParseServerEvent(eventData)
			if output != nil {
				csc.messageChan <- OutputMessage{Message: output, EventType: eventType}
			}
		case <-messageStreamDone:
			logger.Info("messageStreamDone received")
		case <-serverEventStreamDone:
			logger.Info("eventStreamDone received")
		case <-csc.deactivateChan:
			logger.Info("Deactivating chatbot")
			return
		}
	}
}

/*
ParseMessageWrapper parses the given messageWrapper and handles it.
*/
func (csc *ClientSideChatbot) ParseMessageWrapper(messageWrapper *pb.MessageWrapper) ([]byte, pb.MessageType) {
	// If recipient ID is not user ID, it should be the server-side group ID or MLS group ID
	if messageWrapper.RecipientID != csc.chatbotID {
		// check if this is a server-side group
		if _, err := csc.Client.GetServerSideGroupSessionDriver(messageWrapper.RecipientID); err == nil {
			return csc.HandleServerSideGroupMessage(messageWrapper)
		}

		// check if this is a MLS group
		if _, err := csc.Client.GetMlsGroupSessionDriver(messageWrapper.RecipientID); err == nil {
			return csc.HandleMlsGroupMessage(messageWrapper)
		}

		logger.Error("Received message with unknown recipient ID: ", messageWrapper.RecipientID)
		return nil, -1
	}

	// If is individual message
	if messageWrapper.RecipientID == csc.chatbotID {
		// Handle IGA message for client-side group
		//if messageWrapper.GetIsIGA() {
		//	return csc.HandleClientSideGroupIGAMessage(messageWrapper)
		//}

		sessionDriver, err := csc.Client.GetSessionDriver(protocol.NewSignalAddress(messageWrapper.SenderID, 1))
		if err != nil {
			logger.Debug("Received message from user without session ", messageWrapper.SenderID)
			prekeyBundle, err := csc.Client.GetOthersPreKeyBundle(messageWrapper.SenderID)
			if err != nil {
				panic("")
			}
			sessionDriver, err = csc.Client.CreateSessionAndDriver(protocol.NewSignalAddress(messageWrapper.SenderID, 1), prekeyBundle)
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
			groupID, bounceBack, err := csc.Client.ParseSenderKeyDistributionMessage(message, messageWrapper.SenderID)
			if err != nil {
				logger.Error("Failed to parse sender key message: ", err)
			}

			if bounceBack {
				err = csc.DistributeSelfSenderKeyToUserID(messageWrapper.SenderID, groupID, false)
				if err != nil {
					logger.Error("Failed to distribute self sender key to user: ", err)
				}
			}
			return message, messageType

		case pb.MessageType_CLIENT_SIDE_GROUP_MESSAGE:
			return csc.HandleClientSideGroupMessage(message, messageWrapper.SenderID, messageWrapper.GetTreeKEMKeyUpdatePack())
		case pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE:
			logger.Error("Received pseudonym registration message without IGA from ", messageWrapper.SenderID, "as a chatbot.")
			return message, messageType
		}
	}
	return nil, -1
}

/*
ParseServerEventRaw parses the given serverEventRaw and handles it.
*/
func (csc *ClientSideChatbot) ParseServerEventRaw(serverEventRaw []byte) {
	serverEvent := &pb.ServerEvent{}
	err := proto.Unmarshal(serverEventRaw, serverEvent)
	if err != nil {
		logger.Error("ParseMessage failed: ", err)
		return
	}
	csc.ParseServerEvent(serverEvent)
}

/*
ParseServerEvent parse the incoming server events.
*/
func (csc *ClientSideChatbot) ParseServerEvent(serverEvent *pb.ServerEvent) ([]byte, pb.ServerEventType) {
	logger.Info("Received server event: ", serverEvent.GetEventType(), serverEvent.String())

	switch serverEvent.GetEventType() {
	case pb.ServerEventType_GROUP_CHATBOT_INVITATION:
		csc.JoinGroup(
			serverEvent.GetGroupChatbotInvitation().GetGroupID(),
			serverEvent.GetGroupChatbotInvitation().GetGroupType(),
			serverEvent.GetGroupChatbotInvitation().GetParticipantIDs(),
			serverEvent.GetGroupChatbotInvitation().GetIsIGA(),
			serverEvent.GetGroupChatbotInvitation().GetIsPseudo(),
			serverEvent.GetGroupChatbotInvitation().GetTreekemRootPub(),
			serverEvent.GetGroupChatbotInvitation().GetTreekemRootSignPub(),
			serverEvent.GetGroupChatbotInvitation().GetChatbotInitLeaf(),
			serverEvent.GetGroupChatbotInvitation().GetMlsWelcomeMessage(),
			serverEvent.GetGroupChatbotInvitation().GetMlsKeyPackageID(),
		)
		return []byte(serverEvent.GetGroupChatbotInvitation().GetGroupID()), pb.ServerEventType_GROUP_CHATBOT_INVITATION
	case pb.ServerEventType_GROUP_ADDITION:
		csc.AddUserToGroup(
			serverEvent.GetGroupAddition().GetGroupID(),
			serverEvent.GetGroupAddition().GetGroupType(),
			serverEvent.GetGroupAddition().GetAddedID(),
			serverEvent.GetGroupAddition().GetParticipantIDs(),
			serverEvent.GetGroupAddition().GetMlsUserAdd(),
			serverEvent.GetGroupAddition().GetMlsAddCommit())
		return []byte(serverEvent.GetGroupAddition().GetGroupID()), pb.ServerEventType_GROUP_ADDITION
	case pb.ServerEventType_GROUP_CHATBOT_ADDITION:
		if !serverEvent.GetGroupChatbotAddition().GetIsIGA() {
			csc.AddChatbotToGroup(
				serverEvent.GetGroupChatbotAddition().GetGroupID(),
				serverEvent.GetGroupChatbotAddition().GetGroupType(),
				serverEvent.GetGroupChatbotAddition().GetAddedChatbotID(),
				serverEvent.GetGroupChatbotAddition().GetMlsUserAdd(),
				serverEvent.GetGroupChatbotAddition().GetMlsAddCommit())
		}

	case pb.ServerEventType_GROUP_REMOVAL:
		csc.RemoveUserFromGroup(
			serverEvent.GetGroupRemoval().GetGroupID(),
			serverEvent.GetGroupRemoval().GetGroupType(),
			serverEvent.GetGroupRemoval().GetRemovedID(),
			serverEvent.GetGroupRemoval().GetParticipantIDs(),
			serverEvent.GetGroupRemoval().GetMlsRemove(),
			serverEvent.GetGroupRemoval().GetMlsRemoveCommit())
		return []byte(serverEvent.GetGroupRemoval().GetGroupID()), pb.ServerEventType_GROUP_REMOVAL
	case pb.ServerEventType_GROUP_CHATBOT_REMOVAL:
		csc.LeaveGroup(serverEvent.GetGroupRemoval().GetGroupID(), serverEvent.GetGroupRemoval().GetGroupType())
		return []byte(serverEvent.GetGroupRemoval().GetGroupID()), pb.ServerEventType_GROUP_CHATBOT_REMOVAL
	}
	return nil, -1
}
