package server

import (
	pb "chatbot-poc-go/pkg/protos/services"
	"context"
	"log"
	"math/rand"
)

var storage = NewStorage()

// ServiceServer is used to implement ChatServiceServer.
type ServiceServer struct {
	pb.UnimplementedChatServiceServer
}

// GetUser handles the get user requests.
func (s *ServiceServer) GetUser(ctx context.Context, in *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	log.Printf("Received GetUser: %v", in.GetUserID())
	if !storage.ContainUser(in.GetUserID()) {
		return &pb.GetUserResponse{UserID: "", IdentityKeyPublic: nil, Success: false, ErrorMessage: "userID does not exist"}, nil
	}
	user := storage.GetUser(in.GetUserID())
	return &pb.GetUserResponse{UserID: in.GetUserID(), IdentityKeyPublic: user.identityKey, RegistrationID: user.registrationID, Success: true, ErrorMessage: ""}, nil
}

// SetUser handles the set user requests.
func (s *ServiceServer) SetUser(ctx context.Context, in *pb.SetUserRequest) (*pb.SetUserResponse, error) {
	log.Printf("Received SetUser: %v, %v", in.GetUserID(), in.GetIdentityKeyPublic())
	if !storage.ContainUser(in.GetUserID()) {
		storage.AddUser(in.GetUserID())
	}

	user := storage.GetUser(in.GetUserID())
	user.identityKey = in.GetIdentityKeyPublic()
	user.registrationID = in.GetRegistrationID()

	return &pb.SetUserResponse{Success: true, ErrorMessage: ""}, nil
}

// GetChatbot handles the get chatbot requests.
func (s *ServiceServer) GetChatbot(ctx context.Context, in *pb.GetChatbotRequest) (*pb.GetChatbotResponse, error) {
	log.Printf("Received GetChatbot: %v", in.GetChatbotID())
	if !storage.ContainChatbot(in.GetChatbotID()) {
		return &pb.GetChatbotResponse{ChatbotID: "", IdentityKeyPublic: nil, Success: false, ErrorMessage: "chatbotID does not exist"}, nil
	}
	user := storage.GetChatbot(in.GetChatbotID())
	return &pb.GetChatbotResponse{ChatbotID: in.GetChatbotID(), IdentityKeyPublic: user.identityKey, RegistrationID: user.registrationID, Success: true, ErrorMessage: ""}, nil
}

// SetChatbot handles the set user requests.
func (s *ServiceServer) SetChatbot(ctx context.Context, in *pb.SetChatbotRequest) (*pb.SetChatbotResponse, error) {
	log.Printf("Received SetChatbot: %v, %v", in.GetChatbotID(), in.GetIdentityKeyPublic())
	if !storage.ContainChatbot(in.GetChatbotID()) {
		storage.AddChatbot(in.GetChatbotID())
	}

	chatbot := storage.GetChatbot(in.GetChatbotID())
	chatbot.identityKey = in.GetIdentityKeyPublic()
	chatbot.registrationID = in.GetRegistrationID()

	return &pb.SetChatbotResponse{Success: true, ErrorMessage: ""}, nil
}

// UploadPreKey handles the upload preKey requests.
func (s *ServiceServer) UploadPreKey(ctx context.Context, in *pb.UploadPreKeyRequest) (*pb.UploadPreKeyResponse, error) {
	//log.Printf("Received UploadPreKey: %v %v", in.GetUserID(), in.GetPreKey())

	// Check if the user ID exists
	if !storage.ContainUser(in.GetUserID()) {
		return &pb.UploadPreKeyResponse{Success: false, ErrorMessage: "userID does not exist"}, nil
	}

	storage.GetUser(in.GetUserID()).AddSerializedPreKey(in.GetPreKey(), in.GetPreKeyID())
	return &pb.UploadPreKeyResponse{Success: true, ErrorMessage: ""}, nil
}

// FetchPreKey handles the fetch preKey requests.
func (s *ServiceServer) FetchPreKey(ctx context.Context, in *pb.FetchPreKeyRequest) (*pb.FetchPreKeyResponse, error) {
	log.Printf("Received FetchPreKey: %v", in.GetUserID())

	// Check if the user ID exists
	if !storage.ContainUser(in.GetUserID()) {
		return &pb.FetchPreKeyResponse{Success: false, ErrorMessage: "userID does not exist"}, nil
	}

	preKey, preKeyID := storage.GetUser(in.GetUserID()).GetSerializedPreKey()

	return &pb.FetchPreKeyResponse{PreKey: preKey, PreKeyID: preKeyID, Success: true, ErrorMessage: ""}, nil
}

// UploadSignedPreKey handles the upload signed preKey requests.
func (s *ServiceServer) UploadSignedPreKey(ctx context.Context, in *pb.UploadSignedPreKeyRequest) (*pb.UploadSignedPreKeyResponse, error) {
	//log.Printf("Received UploadSignedPreKey: %v %v", in.GetUserID(), in.GetSignedPreKey())

	// Check if the user ID exists
	if !storage.ContainUser(in.GetUserID()) {
		return &pb.UploadSignedPreKeyResponse{Success: false, ErrorMessage: "userID does not exist"}, nil
	}

	storage.GetUser(in.GetUserID()).SetSerializedSignedPreKey(in.GetSignedPreKey(), in.GetSignedPreKeySig(), in.GetSignedPreKeyID())
	return &pb.UploadSignedPreKeyResponse{Success: true, ErrorMessage: ""}, nil
}

// FetchSignedPreKey handles the fetch signed preKey requests.
func (s *ServiceServer) FetchSignedPreKey(ctx context.Context, in *pb.FetchSignedPreKeyRequest) (*pb.FetchSignedPreKeyResponse, error) {
	log.Printf("Received FetchSignedPreKey: %v", in.GetUserID())

	// Check if the user ID exists
	if !storage.ContainUser(in.GetUserID()) {
		return &pb.FetchSignedPreKeyResponse{Success: false, ErrorMessage: "userID does not exist"}, nil
	}

	signedPreKey, signedPreKeySig, signedPreKeyID := storage.GetUser(in.GetUserID()).GetSerializedSignedPreKey()

	return &pb.FetchSignedPreKeyResponse{SignedPreKey: signedPreKey, SignedPreKeySig: signedPreKeySig, SignedPreKeyID: signedPreKeyID, Success: true, ErrorMessage: ""}, nil
}

// UploadMLSKeyPackage handles the upload MLS key package requests.
func (s *ServiceServer) UploadMLSKeyPackage(ctx context.Context, in *pb.UploadMLSKeyPackageRequest) (*pb.UploadMLSKeyPackageResponse, error) {
	//log.Printf("Received UploadMLSKeyPackage: %v %v %v", in.GetUserID(), in.GetMlsKeyPackageId(), in.GetMlsKeyPackage())

	// Check if the user ID exists
	if !storage.ContainUser(in.GetUserID()) {
		return &pb.UploadMLSKeyPackageResponse{Success: false, ErrorMessage: "userID does not exist"}, nil
	}

	storage.GetUser(in.GetUserID()).SetSerializedMlsKeyPackage(in.GetMlsKeyPackage(), in.GetMlsKeyPackageId())
	return &pb.UploadMLSKeyPackageResponse{Success: true, ErrorMessage: ""}, nil

}

// FetchMLSKeyPackage handles the fetch MLS key package requests.
func (s *ServiceServer) FetchMLSKeyPackage(ctx context.Context, in *pb.FetchMLSKeyPackageRequest) (*pb.FetchMLSKeyPackageResponse, error) {
	log.Printf("Received FetchMLSKeyPackage: %v", in.GetUserID())

	// Check if the user ID exists
	if !storage.ContainUser(in.GetUserID()) {
		return &pb.FetchMLSKeyPackageResponse{Success: false, ErrorMessage: "userID does not exist"}, nil
	}

	mlsKeyPackage, mlsKeyPackageId := storage.GetUser(in.GetUserID()).GetSerializedMlsKeyPackage()

	log.Printf("mlsKeyPackageId: %v", mlsKeyPackageId)

	return &pb.FetchMLSKeyPackageResponse{MlsKeyPackage: mlsKeyPackage, MlsKeyPackageId: mlsKeyPackageId, Success: true, ErrorMessage: ""}, nil

}

// FetchIdentityKey handles the fetch identity key requests.
func (s *ServiceServer) FetchIdentityKey(ctx context.Context, in *pb.FetchIdentityKeyRequest) (*pb.FetchIdentityKeyResponse, error) {
	log.Printf("Received FetchIdentityKey: %v", in.GetUserID())

	// Check if the user ID exists
	if !storage.ContainUser(in.GetUserID()) {
		return &pb.FetchIdentityKeyResponse{Success: false, ErrorMessage: "userID does not exist"}, nil
	}

	return &pb.FetchIdentityKeyResponse{IdentityKey: storage.GetUser(in.GetUserID()).identityKey, Success: true, ErrorMessage: ""}, nil
}

// CreateGroup handles the create group requests.
func (s *ServiceServer) CreateGroup(ctx context.Context, in *pb.CreateGroupRequest) (*pb.CreateGroupResponse, error) {
	log.Printf("Received CreateGroup: group type=%v", in.GetGroupType())

	// Randomly choose an ID that does not exist.
	var groupID string
	for {
		// randomly choose a groupID
		groupID = "group" + RandomString(8)

		if !storage.ContainGroup(groupID) {
			break
		}
	}

	// Check if the initiator exist.
	if !storage.ContainUser(in.GetInitiatorID()) {
		return &pb.CreateGroupResponse{GroupID: "", Success: false, ErrorMessage: "initiatorID does not exist"}, nil
	}

	// Create a new group.
	storage.AddGroup(groupID, int(in.GetGroupType().Number()))
	storage.GetGroup(groupID).AddParticipantByID(in.GetInitiatorID())

	return &pb.CreateGroupResponse{GroupID: groupID, Success: true, ErrorMessage: ""}, nil
}

// GetGroup handles the get group requests.
func (s *ServiceServer) GetGroup(ctx context.Context, in *pb.GetGroupRequest) (*pb.GetGroupResponse, error) {
	log.Printf("Received GetGroup: %v", in.GetGroupID())
	// TODO: Check if the user or chatbot is a participant of the group and is allowed to read the participant list.
	return &pb.GetGroupResponse{GroupID: in.GetGroupID(), ParticipantIDs: storage.GetGroup(in.GetGroupID()).GetParticipantIDs(), Success: true, ErrorMessage: ""}, nil
}

// InviteMember handles the invite member requests.
func (s *ServiceServer) InviteMember(ctx context.Context, in *pb.InviteMemberRequest) (*pb.InviteMemberResponse, error) {
	log.Printf("Received RequestInviteUser: %v %v", in.GetGroupID(), in.GetInvitedID())

	// Check if the groupID exists.
	if !storage.ContainGroup(in.GetGroupID()) {
		return &pb.InviteMemberResponse{Success: false, ErrorMessage: "groupID does not exist"}, nil
	}

	// Check if the participantID exists.
	if !storage.ContainUser(in.GetInvitedID()) {
		return &pb.InviteMemberResponse{Success: false, ErrorMessage: "participantID does not exist"}, nil
	}

	// Add the participant to the group.
	storage.GetGroup(in.GetGroupID()).AddParticipantByID(in.GetInvitedID())

	// Create a ServerEvent GROUP_INVITATION to the participant to notify it that it is added to a group.
	eventMsg := &pb.ServerEvent{
		EventType: pb.ServerEventType_GROUP_INVITATION,
		EventData: &pb.ServerEvent_GroupInvitation{
			GroupInvitation: &pb.GroupInvitation{
				SenderID:                   in.GetInitiatorID(),
				GroupID:                    in.GetGroupID(),
				ParticipantIDs:             storage.GetGroup(in.GetGroupID()).GetParticipantIDs(),
				ChatbotIDs:                 storage.GetGroup(in.GetGroupID()).GetChatbotIDs(),
				ChatbotIsIGA:               storage.GetGroup(in.GetGroupID()).GetChatbotIsIGA(),
				ChatbotIsPseudo:            storage.GetGroup(in.GetGroupID()).GetChatbotIsPseudo(),
				TreeKEMGroupInitKey:        in.GetTreeKEMGroupInitKey(),
				TreeKEMInitLeaf:            in.GetTreeKEMInitLeaf(),
				ChatbotPubKeys:             in.GetChatbotPubKeys(),
				LastTreeKemRootCiphertexts: in.GetLastTreeKemRootCiphertexts(),
				MlsWelcomeMessage:          in.GetMlsWelcomeMessage(),
				MlsKeyPackageID:            in.GetMlsKeyPackageID(),
				GroupType:                  pb.GroupType(storage.GetGroup(in.GetGroupID()).GroupType),
			},
		},
	}

	storage.GetUser(in.GetInvitedID()).PushServerEventToQueue(eventMsg)

	// Create a ServerEvent GROUP_ADDITION to other group members to notify them that a new user is added to a group.
	eventMsg = &pb.ServerEvent{
		EventType: pb.ServerEventType_GROUP_ADDITION,
		EventData: &pb.ServerEvent_GroupAddition{
			GroupAddition: &pb.GroupAddition{
				SenderID:       in.GetInitiatorID(),
				AddedID:        in.GetInvitedID(),
				GroupID:        in.GetGroupID(),
				ParticipantIDs: storage.GetGroup(in.GetGroupID()).GetParticipantIDs(),
				TreeKEMUserAdd: in.GetTreeKEMUserAdd(),
				MlsUserAdd:     in.GetMlsUserAdd(),
				MlsAddCommit:   in.GetMlsAddCommit(),
				GroupType:      pb.GroupType(storage.GetGroup(in.GetGroupID()).GroupType),
			},
		},
	}

	for _, pid := range storage.GetGroup(in.GetGroupID()).GetParticipantIDs() {
		if pid != in.GetInvitedID() {
			storage.GetUser(pid).PushServerEventToQueue(eventMsg)
		}
	}

	for _, cid := range storage.GetGroup(in.GetGroupID()).GetChatbotIDs() {
		if !storage.GetGroup(in.GetGroupID()).GetChatbotIsIGA()[cid] {
			storage.GetChatbot(cid).PushServerEventToQueue(eventMsg)
		}
	}

	return &pb.InviteMemberResponse{Success: true, ErrorMessage: ""}, nil
}

// RemoveMember handles the remove member requests.
func (s *ServiceServer) RemoveMember(ctx context.Context, in *pb.RemoveMemberRequest) (*pb.RemoveMemberResponse, error) {
	log.Printf("Received RequestRemoveMember: %v %v", in.GetGroupID(), in.GetRemovedID())

	// Check if the groupID exists.
	if !storage.ContainGroup(in.GetGroupID()) {
		return &pb.RemoveMemberResponse{Success: false, ErrorMessage: "groupID does not exist"}, nil
	}

	// Check if the removedID exists.
	if !storage.ContainUser(in.GetRemovedID()) {
		return &pb.RemoveMemberResponse{Success: false, ErrorMessage: "participantID does not exist"}, nil
	}

	// Remove the participant from the group.
	storage.GetGroup(in.GetGroupID()).RemoveParticipantByID(in.GetRemovedID())

	// Create a ServerEvent GROUP_REMOVAL to the participant to notify it that it is removed from a group.
	eventMsg := &pb.ServerEvent{
		EventType: pb.ServerEventType_GROUP_REMOVAL,
		EventData: &pb.ServerEvent_GroupRemoval{
			GroupRemoval: &pb.GroupRemoval{
				SenderID:        in.GetInitiatorID(),
				RemovedID:       in.GetRemovedID(),
				GroupID:         in.GetGroupID(),
				ParticipantIDs:  storage.GetGroup(in.GetGroupID()).GetParticipantIDs(),
				MlsRemove:       in.GetMlsRemove(),
				MlsRemoveCommit: in.GetMlsRemoveCommit(),
				GroupType:       pb.GroupType(storage.GetGroup(in.GetGroupID()).GroupType),
			},
		},
	}

	for _, pid := range storage.GetGroup(in.GetGroupID()).GetParticipantIDs() {
		storage.GetUser(pid).PushServerEventToQueue(eventMsg)
	}
	for _, cid := range storage.GetGroup(in.GetGroupID()).GetChatbotIDs() {
		if !storage.GetGroup(in.GetGroupID()).ChatbotIsIGA[cid] {
			storage.GetChatbot(cid).PushServerEventToQueue(eventMsg)
		}
	}
	storage.GetUser(in.GetRemovedID()).PushServerEventToQueue(eventMsg)

	return &pb.RemoveMemberResponse{Success: true, ErrorMessage: ""}, nil
}

/*
InviteChatbot handles the invite chatbot requests.
*/
func (s *ServiceServer) InviteChatbot(ctx context.Context, in *pb.InviteChatbotRequest) (*pb.InviteChatbotResponse, error) {
	log.Printf("Received RequestInviteChatbot: %v %v", in.GetGroupID(), in.GetInvitedID())

	// Check if the groupID exists.
	if !storage.ContainGroup(in.GetGroupID()) {
		return &pb.InviteChatbotResponse{Success: false, ErrorMessage: "groupID does not exist"}, nil
	}

	// Check if the participantID exists.
	if !storage.ContainChatbot(in.GetInvitedID()) {
		return &pb.InviteChatbotResponse{Success: false, ErrorMessage: "chatbotID does not exist"}, nil
	}

	// Reject if pseudonymity does not come with IGA.
	if in.GetIsPseudo() && !in.GetIsIGA() {
		return &pb.InviteChatbotResponse{Success: false, ErrorMessage: "pseudonimity must come with IGA"}, nil
	}

	// Add the chatbot to the group.
	storage.GetGroup(in.GetGroupID()).AddChatbotByID(in.GetInvitedID(), in.GetIsIGA(), in.GetIsPseudo())

	// Create a ServerEvent GROUP_CHATBOT_INVITATION to the participant to notify it that it is added to a group.
	var participantIDs []string
	if in.GetIsIGA() || in.GetIsPseudo() {
		participantIDs = nil
	} else {
		participantIDs = storage.GetGroup(in.GetGroupID()).GetParticipantIDs()
	}

	eventMsg := &pb.ServerEvent{
		EventType: pb.ServerEventType_GROUP_CHATBOT_INVITATION,
		EventData: &pb.ServerEvent_GroupChatbotInvitation{
			GroupChatbotInvitation: &pb.GroupChatbotInvitation{
				SenderID:           in.GetInitiatorID(),
				GroupID:            in.GetGroupID(),
				ParticipantIDs:     participantIDs,
				GroupType:          pb.GroupType(storage.GetGroup(in.GetGroupID()).GroupType),
				IsIGA:              in.GetIsIGA(),
				IsPseudo:           in.GetIsPseudo(),
				TreekemRootPub:     in.GetTreekemRootPub(),
				TreekemRootSignPub: in.GetTreekemRootSignPub(),
				ChatbotInitLeaf:    in.GetChatbotInitLeaf(),
				MlsKeyPackageID:    in.GetMlsKeyPackageID(),
				MlsWelcomeMessage:  in.GetMlsWelcomeMessage(),
			},
		},
	}

	storage.GetChatbot(in.GetInvitedID()).PushServerEventToQueue(eventMsg)

	// Create a ServerEvent GROUP_CHATBOT_ADDITION to other group members to notify them that a new user is added to a group.
	eventMsg = &pb.ServerEvent{
		EventType: pb.ServerEventType_GROUP_CHATBOT_ADDITION,
		EventData: &pb.ServerEvent_GroupChatbotAddition{
			GroupChatbotAddition: &pb.GroupChatbotAddition{
				SenderID:          in.GetInitiatorID(),
				AddedChatbotID:    in.GetInvitedID(),
				GroupID:           in.GetGroupID(),
				ChatbotIDs:        storage.GetGroup(in.GetGroupID()).GetChatbotIDs(),
				IsIGA:             in.GetIsIGA(),
				IsPseudo:          in.GetIsPseudo(),
				GroupType:         pb.GroupType(storage.GetGroup(in.GetGroupID()).GroupType),
				ChatbotCipherText: in.GetChatbotCipherText(),
				MlsUserAdd:        in.GetMlsUserAdd(),
				MlsAddCommit:      in.GetMlsAddCommit(),
			},
		},
	}

	for _, pid := range storage.GetGroup(in.GetGroupID()).GetParticipantIDs() {
		storage.GetUser(pid).PushServerEventToQueue(eventMsg)
	}

	// If is MLS group, send invitation to chatbot as well
	if storage.GetGroup(in.GetGroupID()).GroupType == int(pb.GroupType_MLS) {
		for _, cid := range storage.GetGroup(in.GetGroupID()).GetChatbotIDs() {
			if cid != in.GetInvitedID() {
				storage.GetChatbot(cid).PushServerEventToQueue(eventMsg)
			}
		}
	}

	return &pb.InviteChatbotResponse{Success: true, ErrorMessage: ""}, nil
}

/*
RemoveChatbot handles the remove chatbot requests.
*/
func (s *ServiceServer) RemoveChatbot(ctx context.Context, in *pb.RemoveChatbotRequest) (*pb.RemoveChatbotResponse, error) {
	log.Printf("Received RequestRemoveChatbot: %v %v", in.GetGroupID(), in.GetRemovedID())

	// Check if the groupID exists.
	if !storage.ContainGroup(in.GetGroupID()) {
		return &pb.RemoveChatbotResponse{Success: false, ErrorMessage: "groupID does not exist"}, nil
	}

	// Check if the removedID exists.
	if !storage.ContainChatbot(in.GetRemovedID()) {
		return &pb.RemoveChatbotResponse{Success: false, ErrorMessage: "chatbotID does not exist"}, nil
	}

	// Remove the participant from the group.
	storage.GetGroup(in.GetGroupID()).RemoveChatbotByID(in.GetRemovedID())

	// Create a ServerEvent GROUP_CHATBOT_REMOVAL to the participant to notify it that it is removed from a group.
	eventMsg := &pb.ServerEvent{
		EventType: pb.ServerEventType_GROUP_CHATBOT_REMOVAL,
		EventData: &pb.ServerEvent_GroupChatbotRemoval{
			GroupChatbotRemoval: &pb.GroupChatbotRemoval{
				SenderID:         in.GetInitiatorID(),
				RemovedChatbotID: in.GetRemovedID(),
				GroupID:          in.GetGroupID(),
				ChatbotIDs:       storage.GetGroup(in.GetGroupID()).GetParticipantIDs(),
				GroupType:        pb.GroupType(storage.GetGroup(in.GetGroupID()).GroupType),
			},
		},
	}

	for _, pid := range storage.GetGroup(in.GetGroupID()).GetParticipantIDs() {
		storage.GetUser(pid).PushServerEventToQueue(eventMsg)
	}

	storage.GetChatbot(in.GetRemovedID()).PushServerEventToQueue(eventMsg)

	return &pb.RemoveChatbotResponse{Success: true, ErrorMessage: ""}, nil
}

/*
MessageStream handles the message stream requests, i.e., the messages from the server to the client.
*/
func (s *ServiceServer) MessageStream(srv *pb.MessageStreamInit, stream pb.ChatService_MessageStreamServer) error {
	var messageWrapper *pb.MessageWrapper
	for {
		if messageWrapper == nil {
			messageWrapper = storage.GetUser(srv.GetUserID()).PopMessageFromQueue()
		}

		if err := stream.Send(messageWrapper); err != nil {
			log.Printf("send error %v", err)
		} else {
			messageWrapper = nil // message is sent successfully, so set it to nil
		}
	}
}

/*
SendMessage handles the send message requests, i.e., the messages from the client to the server.
Despite its name, it actually receives the message sent from the client.
*/
func (s *ServiceServer) SendMessage(ctx context.Context, in *pb.MessageWrapper) (*pb.SendMessageResponse, error) {
	log.Printf("Received SendMessage from %v to %v", in.GetSenderID(), in.GetRecipientID())

	messageWrapper := &pb.MessageWrapper{}

	// Serialize the message
	messageWrapper = &pb.MessageWrapper{
		SenderID:             in.GetSenderID(),
		RecipientID:          in.GetRecipientID(),
		EncryptedMessage:     in.GetEncryptedMessage(),
		ChatbotMessages:      nil,
		ChatbotIds:           in.GetChatbotIds(),
		HasPreKey:            in.GetHasPreKey(),
		IsIGA:                in.GetIsIGA(),
		TreeKEMKeyUpdatePack: in.GetTreeKEMKeyUpdatePack(),
		ChatbotKeyUpdatePack: in.GetChatbotKeyUpdatePack(),
		MlsCommit:            in.GetMlsCommit(),
	}

	if storage.ContainUser(in.GetRecipientID()) {
		// Push message to the queue of the recipient
		storage.GetUser(in.GetRecipientID()).PushMessageToQueue(messageWrapper)
	} else if storage.ContainGroup(in.GetRecipientID()) {
		// Push message to the queues of all participants
		for _, pid := range storage.GetGroup(in.GetRecipientID()).GetParticipantIDs() {
			if pid != in.GetSenderID() {
				storage.GetUser(pid).PushMessageToQueue(messageWrapper)
			}
		}

		for _, chatbotMessage := range in.GetChatbotMessages() {
			if storage.ContainChatbot(chatbotMessage.GetChatbotID()) && ContainString(chatbotMessage.GetChatbotID(), storage.GetGroup(in.GetRecipientID()).GetChatbotIDs()) {
				storage.GetChatbot(chatbotMessage.GetChatbotID()).PushMessageToQueue(chatbotMessage.GetMessageWrapper())
			} else {
				return &pb.SendMessageResponse{Success: false, ErrorMessage: "chatbotID does not exist"}, nil
			}
		}
	} else {
		return &pb.SendMessageResponse{Success: false, ErrorMessage: "recipientID does not exist"}, nil
	}

	return &pb.SendMessageResponse{Success: true, ErrorMessage: ""}, nil
}

/*
ServerEventStream handles the server event stream requests.
*/
func (s *ServiceServer) ServerEventStream(srv *pb.ServerEventStreamInit, stream pb.ChatService_ServerEventStreamServer) error {
	var serverEvent *pb.ServerEvent
	for {
		if serverEvent == nil {
			serverEvent = storage.GetUser(srv.GetUserID()).PopServerEventFromQueue()
		}

		if err := stream.Send(serverEvent); err != nil {
			log.Printf("send error %v", err)
		} else {
			serverEvent = nil // event is sent successfully, so set it to nil
		}
	}
}

// RandomString create a random string with the given length.
func RandomString(length int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

// ContainString checks if the given string is in the given string slice.
func ContainString(value string, slice []string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
