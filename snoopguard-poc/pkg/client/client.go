package client

import (
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/util"
	"context"
	"fmt"
	"github.com/s3131212/go-mls"
	"go.mau.fi/libsignal/ecc"
	"go.mau.fi/libsignal/keys/identity"
	"go.mau.fi/libsignal/keys/prekey"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/libsignal/protocol"
	"go.mau.fi/libsignal/serialize"
	"go.mau.fi/libsignal/util/optional"
	"google.golang.org/protobuf/proto"
	"sync"
)

/*
Client is an abstraction of the "user". It can be an actual user (see ../user), a chatbot (see ../chatbot), or anything.
It contains a userID(string), an address (SignalAddress), several drivers, and an underlying user obj (util.User).
*/
type Client struct {
	userID  string
	address *protocol.SignalAddress

	clientSessionDrivers          sync.Map
	serverSideGroupSessionDrivers map[string]*ServerSideGroupSessionDriver
	clientSideGroupSessionDrivers map[string]*ClientSideGroupSessionDriver
	mlsGroupSessionDrivers        map[string]*MlsGroupSessionDriver

	user *util.User

	chatServiceClient    pb.ChatServiceClient
	chatServiceClientCtx context.Context
}

/*
NewClient creates a new Client.
It takes in a userID(string) and an initUser(bool), which determines whether the user should do the initialization, e.g. registering the user, uploading the prekey bundle.
*/
func NewClient(userID string) *Client {
	return &Client{
		userID:                        userID,
		address:                       protocol.NewSignalAddress(userID, 1),
		serverSideGroupSessionDrivers: make(map[string]*ServerSideGroupSessionDriver),
		clientSideGroupSessionDrivers: make(map[string]*ClientSideGroupSessionDriver),
		mlsGroupSessionDrivers:        make(map[string]*MlsGroupSessionDriver),
		user:                          util.NewUser(userID, 1, serialize.NewProtoBufSerializer()),
	}
}

/*
SetChatServiceClient injects the chatServiceClient and chatServiceClientCtx into the Client.
*/
func (client *Client) SetChatServiceClient(chatServiceClient *pb.ChatServiceClient, chatServiceClientCtx *context.Context) {
	client.chatServiceClient = *chatServiceClient
	client.chatServiceClientCtx = *chatServiceClientCtx
}

/*
GetIdentityKey returns the identity key of the user.
*/
func (client *Client) GetIdentityKey() *identity.KeyPair {
	return client.user.GetIdentityKey()
}

/*
GetRegistrationID returns the registration ID of the user.
*/
func (client *Client) GetRegistrationID() uint32 {
	return client.user.GetRegistrationID()
}

/*
UploadPreKeyByID uploads the serialized preKey determined by the preKeyID(uint32) to the server.
*/
func (client *Client) UploadPreKeyByID(preKeyID uint32) bool {
	//logger.Info("Uploading preKey for User: ", client.userID)

	// Upload preKey
	res, err := client.chatServiceClient.UploadPreKey(client.chatServiceClientCtx, &pb.UploadPreKeyRequest{
		UserID:   client.userID,
		PreKey:   client.user.GetPreKey(preKeyID).KeyPair().PublicKey().Serialize(),
		PreKeyID: preKeyID,
	})

	if err != nil {
		logger.Error("UploadPreKeyByID failed: ", err)
	}

	if res.GetErrorMessage() != "" {
		logger.Error("UploadPreKeyByID failed: ", res.GetErrorMessage())
	}

	return res.GetSuccess()
}

/*
UploadSignedPreKeyByID uploads the serialized signedPreKey determined by the signedPreKeyID(uint32) to the server.
*/
func (client *Client) UploadSignedPreKeyByID(signedPreKeyID uint32) bool {
	//logger.Info("Uploading signedPreKey for User: ", client.userID)

	// Upload SignedPreKey
	sig := client.user.GetSignedPreKey(signedPreKeyID).Signature()
	res, err := client.chatServiceClient.UploadSignedPreKey(client.chatServiceClientCtx, &pb.UploadSignedPreKeyRequest{
		UserID:          client.userID,
		SignedPreKey:    client.user.GetSignedPreKey(signedPreKeyID).KeyPair().PublicKey().Serialize(),
		SignedPreKeySig: sig[:],
		SignedPreKeyID:  signedPreKeyID,
	})

	if err != nil {
		logger.Error("UploadSignedPreKeyByID failed: ", err)
	}

	if res.GetErrorMessage() != "" {
		logger.Error("UploadSignedPreKeyByID failed: ", res.GetErrorMessage())
	}

	return res.GetSuccess()
}

/*
GeneratePreKey generates a preKey.
*/
func (client *Client) GeneratePreKey(start int) uint32 {
	return client.user.GeneratePreKey(start)
}

/*
GenerateSignedPreKey generates a signedPreKey.
*/
func (client *Client) GenerateSignedPreKey() uint32 {
	return client.user.GenerateSignedPreKey()
}

/*
GetSelfPreKeyBundle gets the preKeyBundle of the user.
*/
func (client *Client) GetSelfPreKeyBundle(preKeyID uint32, signedPreKeyID uint32) *prekey.Bundle {
	return client.user.GetPreKeyBundle(preKeyID, signedPreKeyID)
}

/*
GetOthersPreKeyBundle gets the preKeyBundle of the given recipientAddress from the server.
*/
func (client *Client) GetOthersPreKeyBundle(recipientID string) (*prekey.Bundle, error) {
	logger.Info("Getting preKeyBundle for User: ", recipientID)

	// Get others preKey
	resPreKey, err := client.chatServiceClient.FetchPreKey(client.chatServiceClientCtx, &pb.FetchPreKeyRequest{
		UserID: recipientID,
	})
	if err != nil {
		logger.Error("FetchPreKey failed: ", err)
		return nil, err
	}

	// Get others signedPreKey
	resSignedPreKey, err := client.chatServiceClient.FetchSignedPreKey(client.chatServiceClientCtx, &pb.FetchSignedPreKeyRequest{
		UserID: recipientID,
	})
	if err != nil {
		logger.Error("FetchSignedPreKey failed: ", err)
		return nil, err
	}

	// Get others user info
	resUserInfo, err := client.chatServiceClient.GetUser(client.chatServiceClientCtx, &pb.GetUserRequest{
		UserID: recipientID,
	})
	if err != nil {
		logger.Error("GetUser failed: ", err)
		return nil, err
	}

	return prekey.NewBundle(
		resUserInfo.GetRegistrationID(),
		1,
		optional.NewOptionalUint32(resPreKey.GetPreKeyID()),
		resSignedPreKey.GetSignedPreKeyID(),
		ecc.NewDjbECPublicKey([32]byte(resPreKey.GetPreKey()[1:])),
		ecc.NewDjbECPublicKey([32]byte(resSignedPreKey.GetSignedPreKey()[1:])),
		[64]byte(resSignedPreKey.GetSignedPreKeySig()),
		identity.NewKey(ecc.NewDjbECPublicKey([32]byte(resUserInfo.GetIdentityKeyPublic()[1:]))),
	), nil
}

/*
GenerateMLSKeyPackage generates the MLS key package.
*/
func (client *Client) GenerateMLSKeyPackage(id uint32) {
	client.user.GenerateMLSKeyPackage(id)
}

/*
GetSelfMLSKeyPackage gets the MLS key package of the user.
*/
func (client *Client) GetSelfMLSKeyPackage(id uint32) mls.KeyPackage {
	return client.user.GetMLSKeyPackage(id)
}

/*
UploadMLSKeyPackage uploads the MLS key package to the server.
*/
func (client *Client) UploadMLSKeyPackage(id uint32) bool {
	//logger.Info("Uploading MLS key package for User: ", client.userID)
	serializedKp, err := util.SerializeMLSKeyPackage(client.user.GetMLSKeyPackage(id))
	if err != nil {
		logger.Error("SerializeMLSKeyPackage failed: ", err)
		return false
	}

	// Upload MLS key package
	res, err := client.chatServiceClient.UploadMLSKeyPackage(client.chatServiceClientCtx, &pb.UploadMLSKeyPackageRequest{
		UserID:          client.userID,
		MlsKeyPackage:   serializedKp,
		MlsKeyPackageId: id,
	})

	if err != nil {
		logger.Error("UploadMLSKeyPackage failed: ", err)
	}

	if res.GetErrorMessage() != "" {
		logger.Error("UploadMLSKeyPackage failed: ", res.GetErrorMessage())
	}

	return res.GetSuccess()
}

/*
GetOthersMLSKeyPackage gets the MLS key package of the given recipientID from the server.
*/
func (client *Client) GetOthersMLSKeyPackage(recipientID string) (mls.KeyPackage, uint32, error) {
	logger.Info("Getting MLS key package for User: ", recipientID)

	// Get others MLS key package
	res, err := client.chatServiceClient.FetchMLSKeyPackage(client.chatServiceClientCtx, &pb.FetchMLSKeyPackageRequest{
		UserID: recipientID,
	})
	if err != nil {
		logger.Error("FetchMLSKeyPackage failed: ", err)
		return mls.KeyPackage{}, 0, err
	}

	deserializedKp, err := util.DeserializeMLSKeyPackage(res.GetMlsKeyPackage())
	if err != nil {
		logger.Error("DeserializeMLSKeyPackage failed: ", err)
		return mls.KeyPackage{}, 0, err

	}

	return deserializedKp, res.GetMlsKeyPackageId(), nil
}

/*
CreateSessionAndDriver creates a session and its driver.
*/
func (client *Client) CreateSessionAndDriver(recipientAddress *protocol.SignalAddress, preKeyBundle *prekey.Bundle) (*ClientSessionDriver, error) {
	if session, exists := client.clientSessionDrivers.Load(recipientAddress.Name()); exists {
		logger.Debug("Session exists, reusing the same session for ", recipientAddress.Name())
		return session.(*ClientSessionDriver), nil
	}
	logger.Info("Creating session for ", recipientAddress.Name())
	sessionWrapper := client.user.CreateSessionWrapper(recipientAddress, preKeyBundle)
	session := NewClientSessionDriver(client.userID, recipientAddress.Name(), sessionWrapper)
	client.clientSessionDrivers.Store(recipientAddress.Name(), session)

	session.SetChatServiceClient(&client.chatServiceClient, &client.chatServiceClientCtx)
	return session, nil
}

/*
GetSessionDriver gets the session driver of the given recipientAddress.
*/
func (client *Client) GetSessionDriver(recipientAddress *protocol.SignalAddress) (*ClientSessionDriver, error) {
	if session, exists := client.clientSessionDrivers.Load(recipientAddress.Name()); exists {
		return session.(*ClientSessionDriver), nil
	}
	return nil, fmt.Errorf("session not found")
}

/*
CreateServerSideGroupSessionAndDriver creates a server-side group session and its driver.
*/
func (client *Client) CreateServerSideGroupSessionAndDriver(groupID string, groupParticipantIDs []string, groupChatbotIDs []string) *ServerSideGroupSessionDriver {
	if session, exists := client.serverSideGroupSessionDrivers[groupID]; exists {
		logger.Debug("Server-side group session exists, reusing the same session for ", groupID)
		return session
	}
	logger.Info("Creating server-side group session for ", groupID)
	groupSession := util.NewGroupChatServerSideFanout(client.user, protocol.NewSenderKeyName(groupID, client.address))
	client.serverSideGroupSessionDrivers[groupID] = NewServerSideGroupSessionDriver(client.userID, groupID, groupSession, groupParticipantIDs, groupChatbotIDs)
	client.serverSideGroupSessionDrivers[groupID].SetChatServiceClient(&client.chatServiceClient, &client.chatServiceClientCtx)
	client.serverSideGroupSessionDrivers[groupID].SetSendIndividualMessage(client.SendIndividualMessage)

	return client.serverSideGroupSessionDrivers[groupID]
}

/*
GetServerSideGroupSessionDriver gets the server-side group session driver of the given groupID.
*/
func (client *Client) GetServerSideGroupSessionDriver(groupID string) (*ServerSideGroupSessionDriver, error) {
	if session, exists := client.serverSideGroupSessionDrivers[groupID]; exists {
		return session, nil
	}
	return nil, fmt.Errorf("server-side group session not found")
}

/*
CreateClientSideGroupSessionAndDriver creates a client-side group session and its driver.
*/
func (client *Client) CreateClientSideGroupSessionAndDriver(groupID string, groupParticipantIDs []string, groupChatbotIDs []string) *ClientSideGroupSessionDriver {
	if session, exists := client.clientSideGroupSessionDrivers[groupID]; exists {
		logger.Debug("Client-side group session exists, reusing the same session for ", groupID)
		return session
	}
	logger.Info("Creating client-side group session for ", groupID)
	groupSession := util.NewGroupChatClientSideFanout(client.user, groupID)
	client.clientSideGroupSessionDrivers[groupID] = NewClientSideGroupSessionDriver(client.userID, groupID, groupSession, groupParticipantIDs, groupChatbotIDs)
	client.clientSideGroupSessionDrivers[groupID].SetChatServiceClient(&client.chatServiceClient, &client.chatServiceClientCtx)
	client.clientSideGroupSessionDrivers[groupID].SetSendIndividualMessage(client.SendIndividualMessage)
	return client.clientSideGroupSessionDrivers[groupID]
}

/*
GetClientSideGroupSessionDriver gets the client-side group session driver of the given groupID.
*/
func (client *Client) GetClientSideGroupSessionDriver(groupID string) (*ClientSideGroupSessionDriver, error) {
	if session, exists := client.clientSideGroupSessionDrivers[groupID]; exists {
		return session, nil
	}
	return nil, fmt.Errorf("client-side group session not found")
}

/*
CreateMlsGroupSessionAndDriver creates a MLS group session and its driver without setting the state.
*/
func (client *Client) CreateMlsGroupSessionAndDriver(groupID string, groupParticipantIDs []string, groupChatbotIDs []string) *MlsGroupSessionDriver {
	if session, exists := client.mlsGroupSessionDrivers[groupID]; exists {
		logger.Debug("MLS group session exists, reusing the same session for ", groupID)
		return session
	}
	logger.Info("Creating MLS group session for ", groupID)

	client.mlsGroupSessionDrivers[groupID] = NewMlsGroupSessionDriver(client.userID, groupID, groupParticipantIDs, groupChatbotIDs)
	client.mlsGroupSessionDrivers[groupID].SetChatServiceClient(&client.chatServiceClient, &client.chatServiceClientCtx)
	client.mlsGroupSessionDrivers[groupID].SetSendIndividualMessage(client.SendIndividualMessage)
	return client.mlsGroupSessionDrivers[groupID]
}

/*
GetMlsGroupSessionDriver gets the MLS group session driver of the given groupID.
*/
func (client *Client) GetMlsGroupSessionDriver(groupID string) (*MlsGroupSessionDriver, error) {
	if session, exists := client.mlsGroupSessionDrivers[groupID]; exists {
		return session, nil
	}
	return nil, fmt.Errorf("MLS group session not found")
}

/*
SetMlsGroupStateFromWelcome sets the MLS group state from the welcome message.
*/
func (client *Client) SetMlsGroupStateFromWelcome(groupID string, welcome mls.Welcome, keyPackageId uint32) error {
	groupState, err := client.user.GenerateMLSStateFromWelcome(&welcome, client.user.GetMLSKeyPackage(keyPackageId))
	if err != nil {
		logger.Error("Failed to generate MLS state from welcome: ", err)
		return err
	}
	client.mlsGroupSessionDrivers[groupID].SetGroupState(groupState)
	return nil
}

/*
SetMlsGroupStateFromEmpty sets the MLS group state from the empty state.
*/
func (client *Client) SetMlsGroupStateFromEmpty(groupID string, keyPackageId uint32) error {
	groupState, err := client.user.GenerateMLSStateFromEmpty([]byte(groupID), client.user.GetMLSKeyPackage(keyPackageId))
	if err != nil {
		logger.Error("Failed to generate MLS state from empty: ", err)
		return err
	}
	client.mlsGroupSessionDrivers[groupID].SetGroupState(groupState)
	return nil

}

/*
SendIndividualMessage sends an individual Message and its MessageType to the given recipientAddress.
*/
func (client *Client) SendIndividualMessage(recipientAddress *protocol.SignalAddress, messageWrapper *pb.MessageWrapper) error {

	sessionDriver, exists := client.clientSessionDrivers.Load(recipientAddress.Name())
	if !exists {
		logger.Debug("Send individual Message but session not found, creating one: ", recipientAddress.Name())
		prekeyBundle, err := client.GetOthersPreKeyBundle(recipientAddress.Name())
		if err != nil {
			return err
		}
		sessionDriver, err = client.CreateSessionAndDriver(recipientAddress, prekeyBundle)
		if err != nil {
			return err
		}
	}

	logger.Info(fmt.Sprintf("Send individual Message to %s: %s", recipientAddress.Name(), messageWrapper.String()))
	return sessionDriver.(*ClientSessionDriver).SendMessage(messageWrapper)
}

/*
SendServerSideGroupMessage sends a server-side group Message and its MessageType to the given groupID.
*/
func (client *Client) SendServerSideGroupMessage(groupID string, message *pb.MessageWrapper) error {
	sessionDriver, exist := client.serverSideGroupSessionDrivers[groupID]
	if !exist {
		logger.Debug("Send server-side group Message but session not found, creating one: ", groupID)
		sessionDriver = client.CreateServerSideGroupSessionAndDriver(groupID, []string{}, []string{})
	}

	logger.Info(fmt.Sprintf("Send server-side group Message to group %s: %s", groupID, message.String()))
	return sessionDriver.SendMessage(message)
}

/*
SendClientSideGroupMessage sends a client-side group Message and its MessageType to the given groupID, that is, to all members in pairwise.
*/
func (client *Client) SendClientSideGroupMessage(groupID string, messages map[string]*pb.MessageWrapper) error {
	sessionDriver, exist := client.clientSideGroupSessionDrivers[groupID]
	if !exist {
		logger.Info("Send client-side group Message but session not found, creating one: ", groupID)
		sessionDriver = client.CreateClientSideGroupSessionAndDriver(groupID, []string{}, []string{})
	}

	return sessionDriver.SendMessage(messages)
}

/*
SendMlsGroupMessage sends a MLS group Message and its MessageType to the given groupID.
*/
func (client *Client) SendMlsGroupMessage(groupID string, message *pb.MessageWrapper) error {
	sessionDriver, exist := client.mlsGroupSessionDrivers[groupID]
	if !exist {
		logger.Error("Send MLS group Message but session not found: ", groupID)
		return fmt.Errorf("session not found")
	}

	logger.Info(fmt.Sprintf("Send MLS group Message to group %s: %s", groupID, message.String()))
	return sessionDriver.SendMessage(message)
}

/*
ParseSenderKeyDistributionMessage takes the raw senderKeyMessage, parses it into a SenderKeyMessage object, and then add to the corresponding server-side group session.
*/
func (client *Client) ParseSenderKeyDistributionMessage(senderKeyMessageRaw []byte, senderID string) (string, bool, error) {
	senderKeyDistributionMessageParsed := &pb.SenderKeyDistributionMessage{}
	if err := proto.Unmarshal(senderKeyMessageRaw, senderKeyDistributionMessageParsed); err != nil {
		logger.Error("Failed to parse sender key distribution Message: ", err)
		return "", false, err
	}

	logger.Info(fmt.Sprintf("Received sender key distribution message from %v to %v for server-side group %v", senderID, client.userID, senderKeyDistributionMessageParsed.GetGroupID()))

	senderKeyDistributionMessage, err := protocol.NewSenderKeyDistributionMessageFromBytes(senderKeyDistributionMessageParsed.GetSenderKeyDistributionMessage(), client.user.Serializer.SenderKeyDistributionMessage)
	if err != nil {
		logger.Error("Failed to decode sender key distribution Message: ", err)
		return "", false, err
	}

	session, exist := client.serverSideGroupSessionDrivers[senderKeyDistributionMessageParsed.GetGroupID()]
	if !exist {
		logger.Error("Received Message from user without server-side session", senderID)
		session = client.CreateServerSideGroupSessionAndDriver(senderKeyDistributionMessageParsed.GetGroupID(), []string{}, []string{})
	}

	session.AddSenderKey(senderID, senderKeyDistributionMessage)

	return senderKeyDistributionMessageParsed.GetGroupID(), senderKeyDistributionMessageParsed.GetBounceBack(), nil
}

/*
ParseSenderKeyMessage parses the incoming server-side group Message.
*/
func (client *Client) ParseSenderKeyMessage(messageRaw []byte) (*protocol.SenderKeyMessage, error) {
	message, err := protocol.NewSenderKeyMessageFromBytes(messageRaw, client.user.Serializer.SenderKeyMessage)
	return message, err
}

/*
ParseClientSideGroupMessage parses the incoming client-side group Message.
*/
func (client *Client) ParseClientSideGroupMessage(clientSideGroupMessageRaw []byte, senderID string) (string, []byte, pb.MessageType) {
	clientSideGroupMessage := &pb.ClientSideGroupMessage{}
	err := proto.Unmarshal(clientSideGroupMessageRaw, clientSideGroupMessage)
	if err != nil {
		logger.Error("Failed to decode client side group Message: ", err)
		panic("")
	}

	session, exist := client.clientSideGroupSessionDrivers[clientSideGroupMessage.GroupID]
	if !exist {
		logger.Debug("Received Message from user without session: ", senderID)
		client.CreateClientSideGroupSessionAndDriver(clientSideGroupMessage.GroupID, []string{}, []string{})
	}

	groupMessage, groupMessageType := session.ParseDecryptedMessage(clientSideGroupMessage)

	logger.Info(fmt.Sprintf("Received Message from %v in client-side group %v: %v", senderID, clientSideGroupMessage.GroupID, string(groupMessage)))

	return clientSideGroupMessage.GroupID, groupMessage, groupMessageType
}

/*
JoinGroup adds the user to the group.
*/
func (client *Client) JoinGroup(groupID string, groupType pb.GroupType, participantIDs []string, chatbotIDs []string) {
	switch groupType {
	case pb.GroupType_SERVER_SIDE:
		// Check if already in the group
		if _, exist := client.serverSideGroupSessionDrivers[groupID]; exist {
			logger.Info("Already in the group: ", groupID)
			return
		}
		client.CreateServerSideGroupSessionAndDriver(groupID, participantIDs, chatbotIDs)
	case pb.GroupType_CLIENT_SIDE:
		// Check if already in the group
		if _, exist := client.clientSideGroupSessionDrivers[groupID]; exist {
			logger.Info("Already in the group: ", groupID)
			return
		}
		client.CreateClientSideGroupSessionAndDriver(groupID, participantIDs, chatbotIDs)
	case pb.GroupType_MLS:
		// Check if already in the group
		if _, exist := client.mlsGroupSessionDrivers[groupID]; exist {
			logger.Info("Already in the group: ", groupID)
			return
		}

		client.CreateMlsGroupSessionAndDriver(groupID, participantIDs, chatbotIDs)
	}
	logger.Info("Joining group: ", groupID, " with type: ", groupType, " and participant IDs: ", participantIDs, " and chatbot IDs: ", chatbotIDs)
}

/*
LeaveGroup removes the group from the session driver
*/
func (client *Client) LeaveGroup(groupID string, groupType pb.GroupType) {
	if groupType == pb.GroupType_SERVER_SIDE {
		delete(client.serverSideGroupSessionDrivers, groupID)
	} else if groupType == pb.GroupType_CLIENT_SIDE {
		delete(client.clientSideGroupSessionDrivers, groupID)
	} else if groupType == pb.GroupType_MLS {
		delete(client.mlsGroupSessionDrivers, groupID)
	}
}

/*
GetUserID returns the userID.
*/
func (client *Client) GetUserID() string {
	return client.userID
}
