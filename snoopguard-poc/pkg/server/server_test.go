package server

import (
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/util"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"go.mau.fi/libsignal/serialize"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"log"
	"net"
	"testing"
)

func server(ctx context.Context) (pb.ChatServiceClient, func()) {
	buffer := 101024 * 1024
	lis := bufconn.Listen(buffer)

	baseServer := grpc.NewServer()
	pb.RegisterChatServiceServer(baseServer, &ServiceServer{})
	go func() {
		if err := baseServer.Serve(lis); err != nil {
			log.Printf("error serving server: %v", err)
		}
	}()

	conn, err := grpc.DialContext(ctx, "",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("error connecting to server: %v", err)
	}

	closer := func() {
		err := lis.Close()
		if err != nil {
			log.Printf("error closing listener: %v", err)
		}
		baseServer.Stop()
	}

	client := pb.NewChatServiceClient(conn)

	return client, closer
}

func TestUser(t *testing.T) {
	ctx := context.Background()

	client, closer := server(ctx)
	defer closer()

	// New user
	serializer := serialize.NewProtoBufSerializer()
	user := util.NewUser("alice", 1, serializer)

	// Test set user
	setUserRes, setUserErr := client.SetUser(ctx, &pb.SetUserRequest{UserID: user.UserID, IdentityKeyPublic: user.GetIdentityKey().PublicKey().Serialize(), RegistrationID: user.GetRegistrationID()})
	assert.NotNil(t, setUserRes, "SetUser response should not be nil")
	assert.Empty(t, setUserRes.GetErrorMessage(), "SetUser response should not have an error message")
	assert.Nil(t, setUserErr, "SetUser error should be nil")

	// Test get user
	getUserRes, getUserErr := client.GetUser(ctx, &pb.GetUserRequest{UserID: user.UserID})
	assert.NotNil(t, getUserRes, "GetUser response should not be nil")
	assert.Empty(t, getUserRes.GetErrorMessage(), "GetUser response should not have an error message")
	assert.Nil(t, getUserErr, "GetUser error should be nil")
	assert.Equal(t, "alice", getUserRes.UserID, "GetUser response should have the same user ID as the request")
	assert.Equal(t, getUserRes.GetIdentityKeyPublic(), user.GetIdentityKey().PublicKey().Serialize(), "GetUser response should have the same identity key as the request")
	assert.Equal(t, getUserRes.GetRegistrationID(), user.GetRegistrationID(), "GetUser response should have the same registration ID as the request")
}

func TestInvalidUser(t *testing.T) {
	ctx := context.Background()

	client, closer := server(ctx)
	defer closer()

	// Test get user
	getUserRes, getUserErr := client.GetUser(ctx, &pb.GetUserRequest{UserID: "invalid"})
	assert.NotNil(t, getUserRes, "GetUser response should not be nil")
	assert.NotEmpty(t, getUserRes.GetErrorMessage(), "GetUser response should have an error message")
	assert.Nil(t, getUserErr, "GetUser error should be nil")
}

func TestPreKeyAndSignedPreKey(t *testing.T) {
	ctx := context.Background()

	client, closer := server(ctx)
	defer closer()

	// New user
	serializer := serialize.NewProtoBufSerializer()
	user := util.NewUser("alice", 1, serializer)
	setUserRes, setUserErr := client.SetUser(ctx, &pb.SetUserRequest{UserID: user.UserID})
	assert.NotNil(t, setUserRes, "SetUser response should not be nil")
	assert.Nil(t, setUserErr, "SetUser error should be nil")

	// Generate Pre Keys
	preKeyID := user.GeneratePreKey(0)

	uploadPreKeyRes, uploadPreKeyErr := client.UploadPreKey(ctx, &pb.UploadPreKeyRequest{UserID: "alice", PreKey: user.GetPreKey(preKeyID).Serialize()})
	assert.NotNil(t, uploadPreKeyRes, "UploadPreKey response should not be nil")
	assert.Nil(t, uploadPreKeyErr, "UploadPreKey error should be nil")

	fetchPreKeyRes, fetchPreKeyErr := client.FetchPreKey(ctx, &pb.FetchPreKeyRequest{UserID: user.UserID})
	assert.NotNil(t, fetchPreKeyRes, "FetchPreKey response should not be nil")
	assert.Nil(t, fetchPreKeyErr, "FetchPreKey error should be nil")
	assert.Equal(t, user.GetPreKey(preKeyID).Serialize(), fetchPreKeyRes.PreKey, "FetchPreKey response should have the same pre key.")
	assert.Equal(t, preKeyID, fetchPreKeyRes.PreKeyID, "UploadPreKey response should have the same pre key ID.")

	// Generate Signed Pre Key
	signedPreKeyID := user.GenerateSignedPreKey()

	sig := user.GetSignedPreKey(signedPreKeyID).Signature()
	uploadSignedPreKeyRes, uploadSignedPreKeyErr := client.UploadSignedPreKey(ctx, &pb.UploadSignedPreKeyRequest{UserID: user.UserID, SignedPreKey: user.GetSignedPreKey(signedPreKeyID).KeyPair().PublicKey().Serialize(), SignedPreKeyID: signedPreKeyID, SignedPreKeySig: sig[:]})
	assert.NotNil(t, uploadSignedPreKeyRes, "UploadSignedPreKey response should not be nil")
	assert.Nil(t, uploadSignedPreKeyErr, "UploadSignedPreKey error should be nil")

	fetchSignedPreKeyRes, fetchSignedPreKeyErr := client.FetchSignedPreKey(ctx, &pb.FetchSignedPreKeyRequest{UserID: user.UserID})
	assert.NotNil(t, fetchSignedPreKeyRes, "FetchSignedPreKey response should not be nil")
	assert.Nil(t, fetchSignedPreKeyErr, "FetchSignedPreKey error should be nil")
	assert.Equal(t, user.GetSignedPreKey(signedPreKeyID).KeyPair().PublicKey().Serialize(), fetchSignedPreKeyRes.SignedPreKey, "FetchSignedPreKey response should have the same signed pre key.")
	assert.Equal(t, signedPreKeyID, fetchSignedPreKeyRes.SignedPreKeyID, "UploadSignedPreKey response should have the same signed pre key ID.")
	assert.Equal(t, sig[:], fetchSignedPreKeyRes.SignedPreKeySig, "UploadSignedPreKey response should have the same signed pre key signature.")
}

func TestMLSKeyPackage(t *testing.T) {
	ctx := context.Background()

	client, closer := server(ctx)
	defer closer()

	// New user
	serializer := serialize.NewProtoBufSerializer()
	user := util.NewUser("alice", 1, serializer)
	setUserRes, setUserErr := client.SetUser(ctx, &pb.SetUserRequest{UserID: user.UserID})
	assert.NotNil(t, setUserRes, "SetUser response should not be nil")
	assert.Nil(t, setUserErr, "SetUser error should be nil")

	// Generate MLS Key Package
	for i := 1; i <= 10; i++ {
		user.GenerateMLSKeyPackage(uint32(i))
		serializedKp, err := util.SerializeMLSKeyPackage(user.GetMLSKeyPackage(uint32(i)))
		assert.Nil(t, err, "SerializeMLSKeyPackage error should be nil")
		uploadMlsKeyPackageRes, uploadMlsKeyPackageErr := client.UploadMLSKeyPackage(ctx, &pb.UploadMLSKeyPackageRequest{UserID: user.UserID, MlsKeyPackage: serializedKp, MlsKeyPackageId: uint32(i)})
		assert.NotNil(t, uploadMlsKeyPackageRes, "UploadMLSKeyPackage response should not be nil")
		assert.Nil(t, uploadMlsKeyPackageErr, "UploadMLSKeyPackage error should be nil")
	}

	// Fetch MLS Key Package 10 times and check if it matches what the user stored.
	for i := 1; i <= 10; i++ {
		fetchMlsKeyPackageRes, fetchMlsKeyPackageErr := client.FetchMLSKeyPackage(ctx, &pb.FetchMLSKeyPackageRequest{UserID: user.UserID})
		assert.NotNil(t, fetchMlsKeyPackageRes, "FetchMLSKeyPackage response should not be nil")
		assert.Nil(t, fetchMlsKeyPackageErr, "FetchMLSKeyPackage error should be nil")

		deserializedKp, err := util.DeserializeMLSKeyPackage(fetchMlsKeyPackageRes.GetMlsKeyPackage())
		assert.Nil(t, err, "DeserializeMLSKeyPackage error should be nil")
		assert.True(t, user.GetMLSKeyPackage(uint32(fetchMlsKeyPackageRes.GetMlsKeyPackageId())).Equals(deserializedKp), "FetchMLSKeyPackage response should have the same MLS key package.")
	}
}

func TestGroup(t *testing.T) {
	ctx := context.Background()

	client, closer := server(ctx)
	defer closer()

	// New user
	serializer := serialize.NewProtoBufSerializer()
	alice := util.NewUser("alice", 1, serializer)
	if _, err := client.SetUser(ctx, &pb.SetUserRequest{UserID: alice.UserID}); err != nil {
		t.Error(err)
	}

	bob := util.NewUser("bob", 1, serializer)
	if _, err := client.SetUser(ctx, &pb.SetUserRequest{UserID: bob.UserID}); err != nil {
		t.Error(err)
	}

	carol := util.NewUser("carol", 1, serializer)
	if _, err := client.SetUser(ctx, &pb.SetUserRequest{UserID: carol.UserID}); err != nil {
		t.Error(err)
	}

	// Create group
	createGroupRes, createGroupErr := client.CreateGroup(ctx, &pb.CreateGroupRequest{InitiatorID: alice.UserID, GroupType: pb.GroupType_SERVER_SIDE})
	assert.NotNil(t, createGroupRes, "CreateGroup response should not be nil")
	assert.Nil(t, createGroupErr, "CreateGroup error should be nil")
	assert.True(t, createGroupRes.Success, "CreateGroup response should be successful")

	groupID := createGroupRes.GroupID

	// Get group
	getGroupRes, getGroupErr := client.GetGroup(ctx, &pb.GetGroupRequest{GroupID: groupID})
	assert.NotNil(t, getGroupRes, "GetGroup response should not be nil")
	assert.Nil(t, getGroupErr, "GetGroup error should be nil")
	assert.Equal(t, createGroupRes.GroupID, getGroupRes.GroupID, "GetGroup response should have the same group ID")
	assert.Equal(t, []string{alice.UserID}, getGroupRes.ParticipantIDs, "GetGroup response should have the same participant IDs")

	// Invite Member
	for _, userId := range []string{bob.UserID, carol.UserID} {
		inviteMemberRes, inviteMemberErr := client.InviteMember(ctx, &pb.InviteMemberRequest{GroupID: groupID, InitiatorID: alice.UserID, InvitedID: userId})
		assert.NotNil(t, inviteMemberRes, "RequestInviteUser response should not be nil")
		assert.Nil(t, inviteMemberErr, "RequestInviteUser error should be nil")
	}

	getGroupRes2, getGroupErr2 := client.GetGroup(ctx, &pb.GetGroupRequest{GroupID: groupID})
	assert.NotNil(t, getGroupRes2, "GetGroup response should not be nil")
	assert.Nil(t, getGroupErr2, "GetGroup error should be nil")
	assert.Equal(t, getGroupRes2.ParticipantIDs, []string{alice.UserID, bob.UserID, carol.UserID}, "GetGroup response should have the same participant IDs")

	// Remove Member
	removeMemberRes, removeMemberErr := client.RemoveMember(ctx, &pb.RemoveMemberRequest{GroupID: groupID, InitiatorID: alice.UserID, RemovedID: bob.UserID})
	assert.NotNil(t, removeMemberRes, "RequestRemoveMember response should not be nil")
	assert.Nil(t, removeMemberErr, "RequestRemoveMember error should be nil")
	getGroupRes3, getGroupErr3 := client.GetGroup(ctx, &pb.GetGroupRequest{GroupID: groupID})
	assert.NotNil(t, getGroupRes3, "GetGroup response should not be nil")
	assert.Nil(t, getGroupErr3, "GetGroup error should be nil")
	assert.Equal(t, getGroupRes3.ParticipantIDs, []string{alice.UserID, carol.UserID}, "GetGroup response should have the same participant IDs")
}

func TestMessageStream(t *testing.T) {
	ctx := context.Background()

	aliceClient, aliceCloser := server(ctx)
	bobClient, bobCloser := server(ctx)
	defer aliceCloser()
	defer bobCloser()

	// New user
	serializer := serialize.NewProtoBufSerializer()
	alice := util.NewUser("alice", 1, serializer)
	if _, err := aliceClient.SetUser(ctx, &pb.SetUserRequest{UserID: alice.UserID}); err != nil {
		t.Error(err)
	}

	bob := util.NewUser("bob", 1, serializer)
	if _, err := bobClient.SetUser(ctx, &pb.SetUserRequest{UserID: bob.UserID}); err != nil {
		t.Error(err)
	}

	// Build connection
	aliceConn, err := aliceClient.MessageStream(ctx, &pb.MessageStreamInit{UserID: alice.UserID})
	if err != nil {
		t.Error(err)
	}

	bobConn, err := bobClient.MessageStream(ctx, &pb.MessageStreamInit{UserID: bob.UserID})
	if err != nil {
		t.Error(err)
	}

	// Test send messages
	// Alice to Bob
	aliceClient.SendMessage(ctx, &pb.MessageWrapper{
		SenderID:         alice.UserID,
		RecipientID:      bob.UserID,
		EncryptedMessage: []byte("Test Alice to Bob"), // not encrypted intentionally
		HasPreKey:        false,
	})

	bobRecv, err := bobConn.Recv()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, bobRecv.SenderID, alice.UserID, "Bob should receive message from Alice")
	assert.Equal(t, bobRecv.RecipientID, bob.UserID, "Bob should receive message from Alice")
	assert.Equal(t, bobRecv.EncryptedMessage, []byte("Test Alice to Bob"), "Bob should receive message from Alice")

	// Bob to Alice
	bobClient.SendMessage(ctx, &pb.MessageWrapper{
		SenderID:         bob.UserID,
		RecipientID:      alice.UserID,
		EncryptedMessage: []byte("Test Bob to Alice"), // not encrypted intentionally
		HasPreKey:        false,
	})

	aliceRecv, err := aliceConn.Recv()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, aliceRecv.SenderID, bob.UserID, "Alice should receive message from Bob")
	assert.Equal(t, aliceRecv.RecipientID, alice.UserID, "Alice should receive message from Bob")
	assert.Equal(t, aliceRecv.EncryptedMessage, []byte("Test Bob to Alice"), "Alice should receive message from Bob")

	// Multiple messages
	for i := 0; i < 10; i++ {
		aliceClient.SendMessage(ctx, &pb.MessageWrapper{
			SenderID:         alice.UserID,
			RecipientID:      bob.UserID,
			EncryptedMessage: []byte(fmt.Sprintf("Test Alice to Bob %v", i)), // not encrypted intentionally
			HasPreKey:        false,
		})
		bobClient.SendMessage(ctx, &pb.MessageWrapper{
			SenderID:         bob.UserID,
			RecipientID:      alice.UserID,
			EncryptedMessage: []byte(fmt.Sprintf("Test Bob to Alice %v", i)), // not encrypted intentionally
			HasPreKey:        false,
		})
	}

	for i := 0; i < 10; i++ {
		aliceRecv, err := aliceConn.Recv()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, aliceRecv.SenderID, bob.UserID, "Alice should receive message from Bob")
		assert.Equal(t, aliceRecv.RecipientID, alice.UserID, "Alice should receive message from Bob")
		assert.Equal(t, aliceRecv.EncryptedMessage, []byte(fmt.Sprintf("Test Bob to Alice %v", i)), "Alice should receive message from Bob")
	}

	for i := 0; i < 10; i++ {
		bobRecv, err := bobConn.Recv()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, bobRecv.SenderID, alice.UserID, "Bob should receive message from Alice")
		assert.Equal(t, bobRecv.RecipientID, bob.UserID, "Bob should receive message from Alice")
		assert.Equal(t, bobRecv.EncryptedMessage, []byte(fmt.Sprintf("Test Alice to Bob %v", i)), "Bob should receive message from Alice")
	}
}
