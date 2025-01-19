package chatbot

import (
	"chatbot-poc-go/pkg/client"
	pb "chatbot-poc-go/pkg/protos/services"
	"context"
	"go.mau.fi/libsignal/logger"
	"google.golang.org/grpc"
	"net"
)

type ClientSideChatbot struct {
	Client         *client.Client
	chatbotID      string
	messageChan    chan OutputMessage
	deactivateChan chan bool

	groupPseudonyms map[string]map[string]*PseudoUser

	chatServiceClient    pb.ChatServiceClient
	chatServiceClientCtx context.Context
}

func NewClientSideChatbot(userID string, chatServiceAddress string, setup bool) (*ClientSideChatbot, func() error) {
	clientObj := client.NewClient(userID)
	csc := &ClientSideChatbot{
		Client:          clientObj,
		chatbotID:       userID,
		messageChan:     make(chan OutputMessage, 100),
		deactivateChan:  make(chan bool),
		groupPseudonyms: make(map[string]map[string]*PseudoUser),
	}

	closeChatServiceClient := csc.SetupChatServiceClient(chatServiceAddress)
	clientObj.SetChatServiceClient(&csc.chatServiceClient, &csc.chatServiceClientCtx)
	if !csc.RegisterChatbotToServer() {
		logger.Error("Chatbot registration failed")
		return nil, nil
	}

	go csc.ListenToStreams()

	if setup {
		for i := 1; i <= 20; i = i + 1 {
			preKeyID := clientObj.GeneratePreKey(i)
			clientObj.UploadPreKeyByID(preKeyID)
		}
		signedPreKeyID := clientObj.GenerateSignedPreKey()
		clientObj.UploadSignedPreKeyByID(signedPreKeyID)
	}

	return csc, closeChatServiceClient
}

func NewClientSideChatbotBufconn(userID string, dialer func(context.Context, string) (net.Conn, error), setup bool) *ClientSideChatbot {
	clientObj := client.NewClient(userID)
	csc := &ClientSideChatbot{
		Client:          clientObj,
		chatbotID:       userID,
		messageChan:     make(chan OutputMessage, 100),
		deactivateChan:  make(chan bool),
		groupPseudonyms: make(map[string]map[string]*PseudoUser),
	}

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer))
	if err != nil {
		panic(err)
	}

	serviceClient := pb.NewChatServiceClient(conn)
	clientObj.SetChatServiceClient(&serviceClient, &ctx)
	csc.chatServiceClient = serviceClient
	csc.chatServiceClientCtx = ctx

	if !csc.RegisterChatbotToServer() {
		logger.Error("Chatbot registration failed")
		return nil
	}

	go csc.ListenToStreams()

	if setup {
		// Generate and upload prekeys
		for i := 1; i <= 300; i = i + 1 {
			preKeyID := clientObj.GeneratePreKey(i)
			clientObj.UploadPreKeyByID(preKeyID)
		}

		// Generate and upload signed prekey
		signedPreKeyID := clientObj.GenerateSignedPreKey()
		clientObj.UploadSignedPreKeyByID(signedPreKeyID)

		// Generate and upload MLS key packages
		for i := 1; i <= 200; i = i + 1 {
			clientObj.GenerateMLSKeyPackage(uint32(i))
			clientObj.UploadMLSKeyPackage(uint32(i))
		}
		logger.Info("Finish setup.")
	}

	return csc
}

func (csc *ClientSideChatbot) RegisterChatbotToServer() bool {
	logger.Info("Registering User: ", csc.chatbotID)

	// Register user
	res, err := csc.chatServiceClient.SetChatbot(csc.chatServiceClientCtx, &pb.SetChatbotRequest{
		ChatbotID:         csc.chatbotID,
		IdentityKeyPublic: csc.Client.GetIdentityKey().PublicKey().Serialize(),
		RegistrationID:    csc.Client.GetRegistrationID(),
	})

	if err != nil {
		logger.Error("SetChatbot failed: ", err)
		return false
	}

	if !res.GetSuccess() || res.GetErrorMessage() != "" {
		logger.Error("SetChatbot failed: ", res.GetErrorMessage())
		return false
	}

	return true
}

/*
GetChatbotID returns the chatbot ID.
*/
func (csc *ClientSideChatbot) GetChatbotID() string {
	return csc.chatbotID
}

/*
GetMessageChan returns the message channel.
*/
func (csc *ClientSideChatbot) GetMessageChan() <-chan OutputMessage {
	return csc.messageChan
}

/*
Deactivate deactivates the chatbot.
*/
func (csc *ClientSideChatbot) Deactivate() {
	csc.deactivateChan <- true
}

/*
OutputMessage is used in the message channel to denote the message receive by the user.
*/
type OutputMessage struct {
	Message     []byte
	MessageType pb.MessageType
	EventType   pb.ServerEventType
}

type PseudoUser struct {
	PseudoUserID  string
	SigningPubKey []byte
}
