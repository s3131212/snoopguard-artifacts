package user

import (
	"chatbot-poc-go/pkg/client"
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/treekem"
	"context"
	"go.mau.fi/libsignal/logger"
	"google.golang.org/grpc"
	"net"
)

type ClientSideUser struct {
	Client         *client.Client
	userID         string
	messageChan    chan OutputMessage
	deactivateChan chan bool

	pseudoUsers map[string]map[string]*PseudoUser

	chatServiceClient    pb.ChatServiceClient
	chatServiceClientCtx context.Context
	chatServiceAddress   string
}

func NewClientSideUser(userID string, chatServiceAddress string, setup bool) (*ClientSideUser, func() error) {
	clientObj := client.NewClient(userID)
	csu := &ClientSideUser{
		Client:             clientObj,
		userID:             userID,
		messageChan:        make(chan OutputMessage, 100),
		deactivateChan:     make(chan bool),
		pseudoUsers:        make(map[string]map[string]*PseudoUser),
		chatServiceAddress: chatServiceAddress,
	}

	closeChatServiceClient := csu.SetupChatServiceClient(chatServiceAddress)
	clientObj.SetChatServiceClient(&csu.chatServiceClient, &csu.chatServiceClientCtx)

	if !csu.RegisterUserToServer() {
		logger.Error("User registration failed")
		return nil, nil
	}

	go csu.ListenToStreams()

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

	return csu, closeChatServiceClient
}

func NewClientSideUserBufconn(userID string, dialer func(context.Context, string) (net.Conn, error), setup bool) *ClientSideUser {
	clientObj := client.NewClient(userID)
	csu := &ClientSideUser{
		Client:             clientObj,
		userID:             userID,
		messageChan:        make(chan OutputMessage, 100),
		deactivateChan:     make(chan bool),
		pseudoUsers:        make(map[string]map[string]*PseudoUser),
		chatServiceAddress: "",
	}

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "", grpc.WithInsecure(), grpc.WithContextDialer(dialer))
	if err != nil {
		panic(err)
	}

	serviceClient := pb.NewChatServiceClient(conn)
	clientObj.SetChatServiceClient(&serviceClient, &ctx)
	csu.chatServiceClient = serviceClient
	csu.chatServiceClientCtx = ctx

	if !csu.RegisterUserToServer() {
		logger.Error("User registration failed")
		return nil
	}

	go csu.ListenToStreams()

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

	return csu
}

func (csu *ClientSideUser) RegisterUserToServer() bool {
	logger.Info("Registering User: ", csu.userID)

	// Register user
	res, err := csu.chatServiceClient.SetUser(csu.chatServiceClientCtx, &pb.SetUserRequest{
		UserID:            csu.userID,
		IdentityKeyPublic: csu.Client.GetIdentityKey().PublicKey().Serialize(),
		RegistrationID:    csu.Client.GetRegistrationID(),
	})

	if err != nil {
		logger.Error("SetUser failed: ", err)
		return false
	}

	if !res.GetSuccess() || res.GetErrorMessage() != "" {
		logger.Error("SetUser failed: ", res.GetErrorMessage())
		return false
	}

	return true
}

/*
GetUserID returns the user ID.
*/
func (csu *ClientSideUser) GetUserID() string {
	return csu.userID
}

/*
GetMessageChan returns the message channel.
*/
func (csu *ClientSideUser) GetMessageChan() <-chan OutputMessage {
	return csu.messageChan
}

/*
Deactivate deactivates the user.
*/
func (csu *ClientSideUser) Deactivate() {
	csu.deactivateChan <- true
}

/*
OutputMessage is used in the message channel to denote the message receive by the user.
*/
type OutputMessage struct {
	Message     []byte
	MessageType pb.MessageType
	EventType   pb.ServerEventType
}

/*
PseudoUser is used to denote a pseudo user.
*/
type PseudoUser struct {
	PseudoUserID   string
	SigningKeyPair treekem.Keypair
	SignSecret     []byte
}
