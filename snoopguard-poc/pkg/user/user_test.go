package user

import (
	"bytes"
	"chatbot-poc-go/pkg/client"
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/server"
	"chatbot-poc-go/pkg/treekem"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"go.mau.fi/libsignal/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"
)

var alice, bob, carol, david *ClientSideUser

const bufSize = 1024 * 1024

var listener *bufconn.Listener

func TestCreateUserAndSetupConnection(t *testing.T) {
	alice = createClientSideUserWithRandomUserID("alice")
	bob = createClientSideUserWithRandomUserID("bob")
	carol = createClientSideUserWithRandomUserID("carol")
	david = createClientSideUserWithRandomUserID("david")
}

func TestUserPreKeyAndSignedPreKey(t *testing.T) {
	// Generate prekey and signed prekey
	// Note that this is only for test purpose, when setup=true, the prekeys are generated and uploaded by the time the ClientSideUser is initialized.
	alicePreKeyID := alice.Client.GeneratePreKey(0)
	aliceSignedPreKeyID := alice.Client.GenerateSignedPreKey()
	assert.True(t, alice.Client.UploadPreKeyByID(alicePreKeyID), "Alice upload prekey should be successful")
	assert.True(t, alice.Client.UploadSignedPreKeyByID(aliceSignedPreKeyID), "Alice upload signed prekey should be successful")

	bobPreKeyID := bob.Client.GeneratePreKey(0)
	bobSignedPreKeyID := bob.Client.GenerateSignedPreKey()
	assert.True(t, bob.Client.UploadPreKeyByID(bobPreKeyID), "Bob upload prekey should be successful")
	assert.True(t, bob.Client.UploadSignedPreKeyByID(bobSignedPreKeyID), "Bob upload signed prekey should be successful")

	carolPreKeyID := carol.Client.GeneratePreKey(0)
	carolSignedPreKeyID := carol.Client.GenerateSignedPreKey()
	assert.True(t, carol.Client.UploadPreKeyByID(carolPreKeyID), "Carol upload prekey should be successful")
	assert.True(t, carol.Client.UploadSignedPreKeyByID(carolSignedPreKeyID), "Carol upload signed prekey should be successful")

	// Test PreKeyBundle
	aliceGetBobsPreKeyBundle, err := alice.Client.GetOthersPreKeyBundle(bob.GetUserID())
	assert.Nil(t, err, "Alice should be able to get Bob's prekey bundle")
	bobGetAlicesPreKeyBundle, err := bob.Client.GetOthersPreKeyBundle(alice.GetUserID())
	assert.Nil(t, err, "Bob should be able to get Alice's prekey bundle")

	// Get real PreKeyBundle
	alicePreKeyBundle := alice.Client.GetSelfPreKeyBundle(bobGetAlicesPreKeyBundle.PreKeyID().Value, bobGetAlicesPreKeyBundle.SignedPreKeyID())
	bobPreKeyBundle := bob.Client.GetSelfPreKeyBundle(aliceGetBobsPreKeyBundle.PreKeyID().Value, bobGetAlicesPreKeyBundle.SignedPreKeyID())

	assert.Equal(t, bobGetAlicesPreKeyBundle.IdentityKey(), alicePreKeyBundle.IdentityKey(), "Alice's identity key should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetBobsPreKeyBundle.IdentityKey(), bobPreKeyBundle.IdentityKey(), "Bob's identity key should be the same as the one in the prekey bundle")

	assert.Equal(t, bobGetAlicesPreKeyBundle.PreKeyID(), alicePreKeyBundle.PreKeyID(), "Alice's prekey ID should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetBobsPreKeyBundle.PreKeyID(), bobPreKeyBundle.PreKeyID(), "Bob's prekey ID should be the same as the one in the prekey bundle")

	assert.Equal(t, bobGetAlicesPreKeyBundle.PreKey().Serialize(), alicePreKeyBundle.PreKey().Serialize(), "Alice's prekey should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetBobsPreKeyBundle.PreKey().Serialize(), bobPreKeyBundle.PreKey().Serialize(), "Bob's prekey should be the same as the one in the prekey bundle")

	assert.Equal(t, bobGetAlicesPreKeyBundle.SignedPreKeyID(), alicePreKeyBundle.SignedPreKeyID(), "Alice's signed prekey ID should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetBobsPreKeyBundle.SignedPreKeyID(), bobPreKeyBundle.SignedPreKeyID(), "Bob's signed prekey ID should be the same as the one in the prekey bundle")

	assert.Equal(t, bobGetAlicesPreKeyBundle.SignedPreKey().Serialize(), alicePreKeyBundle.SignedPreKey().Serialize(), "Alice's signed prekey should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetBobsPreKeyBundle.SignedPreKey().Serialize(), bobPreKeyBundle.SignedPreKey().Serialize(), "Bob's signed prekey should be the same as the one in the prekey bundle")

	assert.Equal(t, bobGetAlicesPreKeyBundle.DeviceID(), alicePreKeyBundle.DeviceID(), "Alice's device ID should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetBobsPreKeyBundle.DeviceID(), bobPreKeyBundle.DeviceID(), "Bob's device ID should be the same as the one in the prekey bundle")

	assert.Equal(t, bobGetAlicesPreKeyBundle.RegistrationID(), alicePreKeyBundle.RegistrationID(), "Alice's registration ID should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetBobsPreKeyBundle.RegistrationID(), bobPreKeyBundle.RegistrationID(), "Bob's registration ID should be the same as the one in the prekey bundle")

	assert.Equal(t, bobGetAlicesPreKeyBundle.SignedPreKeySignature(), alicePreKeyBundle.SignedPreKeySignature(), "Alice's signed prekey signature should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetBobsPreKeyBundle.SignedPreKeySignature(), bobPreKeyBundle.SignedPreKeySignature(), "Bob's signed prekey signature should be the same as the one in the prekey bundle")
}

func TestSendMessage(t *testing.T) {
	// Alice to Bob
	_, err := alice.CreateIndividualSession(protocol.NewSignalAddress(bob.GetUserID(), 1))
	assert.Nil(t, err, "Alice should be able to create a session with Bob")
	for i := 0; i < 10; i++ {
		err := alice.SendIndividualMessage(protocol.NewSignalAddress(bob.GetUserID(), 1), []byte(fmt.Sprintf("Hello Bob! %v", i)), pb.MessageType_TEXT_MESSAGE)
		assert.Nil(t, err, "Alice should be able to send message to Bob")
		msg, success := timeOutReadFromMessageChannel(bob.messageChan)
		assert.True(t, success, "Bob should receive a message from Alice")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Bob should receive a text message from Alice")
		assert.Equal(t, fmt.Sprintf("Hello Bob! %v", i), string(msg.Message), "Bob should receive the same message from Alice")
	}

	// Bob to Alice
	_, err = bob.CreateIndividualSession(protocol.NewSignalAddress(alice.GetUserID(), 1))
	assert.Nil(t, err, "Bob should be able to create a session with Alice")
	for i := 0; i < 10; i++ {
		err := bob.SendIndividualMessage(protocol.NewSignalAddress(alice.GetUserID(), 1), []byte(fmt.Sprintf("Hello Alice! %v", i)), pb.MessageType_TEXT_MESSAGE)
		assert.Nil(t, err, "Bob should be able to send message to Alice")
		msg, success := timeOutReadFromMessageChannel(alice.messageChan)
		assert.True(t, success, "Alice should receive a message from Bob")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Alice should receive a text message from Bob")
		assert.Equal(t, fmt.Sprintf("Hello Alice! %v", i), string(msg.Message), "Alice should receive the same message from Bob")
	}
}

// TestClientSideGroupMessage test the client side group Message.
func TestClientSideGroupMessage(t *testing.T) {
	// Alice is the initiator of the group
	groupId, err := alice.CreateGroup(pb.GroupType_CLIENT_SIDE)
	assert.Nil(t, err, "Alice should be able to create a group")

	// Alice should have the group session
	aliceSessionDriver, err := alice.Client.GetClientSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Alice should have the group session")

	// Invite Bob to the group.
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_CLIENT_SIDE, bob.GetUserID())
	msg, success := timeOutReadFromMessageChannel(bob.GetMessageChan())
	assert.True(t, success, "Bob should receive a group invitation from Alice")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "Bob should receive a group invitation from Alice")

	// Bob should have the group session
	bobSessionDriver, err := bob.Client.GetClientSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Bob should have the group session")

	// Alice should receive a GROUP_ADDITION event.
	msg, success = timeOutReadFromMessageChannel(alice.messageChan)
	assert.True(t, success, "Alice should receive a group addition event")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Alice should receive a group addition event")

	// TreeKEM should be the same.
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), bobSessionDriver.GetTreeKEMState()), "Alice and Bob should have the same TreeKEM state")
	assert.True(t, multiTreeKemEqual(aliceSessionDriver.GetMultiTreeKEM(), bobSessionDriver.GetMultiTreeKEM()), "Alice and Bob should have the same MultiTreeKEM state")

	// Alice send a message to the group
	err = alice.SendClientSideGroupMessage(groupId, []byte("Hello everyone! I'm Alice."), pb.MessageType_TEXT_MESSAGE, nil)
	assert.Nil(t, err, "Alice should be able to send a message to the group")

	// Bob should receive the Message
	msg, success = timeOutReadFromMessageChannel(bob.messageChan)
	assert.True(t, success, "Bob should receive a Message from Alice")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Bob should receive a group text Message from Alice")
	assert.Equal(t, "Hello everyone! I'm Alice.", string(msg.Message), "Bob should receive a group text Message from Alice")

	// TreeKEM should be the same.
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), bobSessionDriver.GetTreeKEMState()), "Alice and Bob should have the same TreeKEM state")
	assert.True(t, multiTreeKemEqual(aliceSessionDriver.GetMultiTreeKEM(), bobSessionDriver.GetMultiTreeKEM()), "Alice and Bob should have the same MultiTreeKEM state")

	// Invite Carol to the group.
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_CLIENT_SIDE, carol.GetUserID())

	// Carol should receive a group invitation from Alice
	msg, success = timeOutReadFromMessageChannel(carol.GetMessageChan())
	assert.True(t, success, "Carol should receive a group invitation from Alice")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "Carol should receive a group invitation from Alice")

	// Carol should have the group session
	carolSessionDriver, err := carol.Client.GetClientSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Carol should have the group session")

	// Alice and Bob should receive a GROUP_ADDITION event.
	for _, c := range []<-chan OutputMessage{alice.messageChan, bob.messageChan} {
		msg, success = timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Alice and Bob should receive a group addition event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Alice and Bob should receive a group addition event")
	}

	// TreeKEM should be the same.
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), carolSessionDriver.GetTreeKEMState()), "Alice and Carol should have the same TreeKEM state")
	assert.True(t, treekemGroupEqual(bobSessionDriver.GetTreeKEMState(), carolSessionDriver.GetTreeKEMState()), "Bob and Carol should have the same TreeKEM state")
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), bobSessionDriver.GetTreeKEMState()), "Alice and Bob should have the same TreeKEM state")

	assert.True(t, multiTreeKemEqual(aliceSessionDriver.GetMultiTreeKEM(), carolSessionDriver.GetMultiTreeKEM()), "Alice and Carol should have the same MultiTreeKEM state")
	assert.True(t, multiTreeKemEqual(bobSessionDriver.GetMultiTreeKEM(), carolSessionDriver.GetMultiTreeKEM()), "Bob and Carol should have the same MultiTreeKEM state")
	assert.True(t, multiTreeKemEqual(aliceSessionDriver.GetMultiTreeKEM(), bobSessionDriver.GetMultiTreeKEM()), "Alice and Bob should have the same MultiTreeKEM state")

	// Bob send a message to the group
	err = bob.SendClientSideGroupMessage(groupId, []byte("Hello everyone! I'm Bob."), pb.MessageType_TEXT_MESSAGE, nil)
	assert.Nil(t, err, "Bob should be able to send a message to the group")

	// Alice should receive the message
	msg, success = timeOutReadFromMessageChannel(alice.messageChan)
	assert.True(t, success, "Alice should receive a message from Bob")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Alice should receive a group text message from Bob")

	// Carol should receive the message
	msg, success = timeOutReadFromMessageChannel(carol.messageChan)
	assert.True(t, success, "Carol should receive a message from Bob")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Carol should receive a group text message from Bob")

	// TreeKEM should be the same.
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), carolSessionDriver.GetTreeKEMState()), "Alice and Carol should have the same TreeKEM state")
	assert.True(t, treekemGroupEqual(bobSessionDriver.GetTreeKEMState(), carolSessionDriver.GetTreeKEMState()), "Bob and Carol should have the same TreeKEM state")
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), bobSessionDriver.GetTreeKEMState()), "Alice and Bob should have the same TreeKEM state")

	assert.True(t, multiTreeKemEqual(aliceSessionDriver.GetMultiTreeKEM(), carolSessionDriver.GetMultiTreeKEM()), "Alice and Carol should have the same MultiTreeKEM state")
	assert.True(t, multiTreeKemEqual(bobSessionDriver.GetMultiTreeKEM(), carolSessionDriver.GetMultiTreeKEM()), "Bob and Carol should have the same MultiTreeKEM state")
	assert.True(t, multiTreeKemEqual(aliceSessionDriver.GetMultiTreeKEM(), bobSessionDriver.GetMultiTreeKEM()), "Alice and Bob should have the same MultiTreeKEM state")

	// Carol invite David to the group.
	carol.RequestInviteUserToGroup(groupId, pb.GroupType_CLIENT_SIDE, david.GetUserID())
	msg, success = timeOutReadFromMessageChannel(david.GetMessageChan())
	assert.True(t, success, "David should receive a group invitation from Carol")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "David should receive a group invitation from Carol")

	// David should have the group session
	davidSessionDriver, err := david.Client.GetClientSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "David should have the group session")

	// Alice, Bob, and Carol should receive a GROUP_ADDITION event.
	for _, c := range []<-chan OutputMessage{alice.messageChan, bob.messageChan, carol.messageChan} {
		msg, success = timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Alice, Bob, and Carol should receive a group addition event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Alice, Bob, and Carol should receive a group addition event")
	}

	// TreeKEM should be the same
	for _, sessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
		for _, otherSessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
			assert.True(t, treekemGroupEqual(sessionDriver.GetTreeKEMState(), otherSessionDriver.GetTreeKEMState()), "All users should have the same TreeKEM state")
			assert.True(t, multiTreeKemEqual(sessionDriver.GetMultiTreeKEM(), otherSessionDriver.GetMultiTreeKEM()), "All users should have the same MultiTreeKEM state")
		}
	}

	// Carol send a message to the group
	err = carol.SendClientSideGroupMessage(groupId, []byte("Hello everyone! I'm Carol."), pb.MessageType_TEXT_MESSAGE, nil)
	assert.Nil(t, err, "Carol should be able to send a message to the group")

	// Alice, Bob, and David should receive the message
	for _, c := range []<-chan OutputMessage{alice.messageChan, bob.messageChan, david.messageChan} {
		msg, success = timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Should receive a message from Carol")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text message from Carol")
		assert.Equal(t, "Hello everyone! I'm Carol.", string(msg.Message), "Should receive a group text message from Carol")
	}

	// TreeKEM should be the same
	for _, sessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
		for _, otherSessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
			assert.True(t, treekemGroupEqual(sessionDriver.GetTreeKEMState(), otherSessionDriver.GetTreeKEMState()), "All users should have the same TreeKEM state")
			assert.True(t, multiTreeKemEqual(sessionDriver.GetMultiTreeKEM(), otherSessionDriver.GetMultiTreeKEM()), "All users should have the same MultiTreeKEM state")
		}
	}

}

// TestServerSideGroupMessage test the server side group Message.
func TestServerSideGroupMessage(t *testing.T) {
	// Alice is the initiator of the group
	groupId, err := alice.CreateGroup(pb.GroupType_SERVER_SIDE)
	assert.Nil(t, err, "Alice should be able to create a group")

	// Alice should have the group session
	aliceSessionDriver, err := alice.Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Alice should have the group session")

	// Invite Bob to the group.
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_SERVER_SIDE, bob.GetUserID())
	msg, success := timeOutReadFromMessageChannel(bob.GetMessageChan())
	assert.True(t, success, "Bob should receive a group invitation from Alice")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "Bob should receive a group invitation from Alice")

	// Bob should have the group session
	bobSessionDriver, err := bob.Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Bob should have the group session")

	// Alice should receive a GROUP_ADDITION event.
	msg, success = timeOutReadFromMessageChannel(alice.messageChan)
	assert.True(t, success, "Alice should receive a group addition event")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Alice should receive a group addition event")

	// Bob distribute his sender key to all
	err = bob.DistributeSelfSenderKeyToAll(groupId)
	assert.Nil(t, err, "Bob should be able to distribute his sender key to all")

	// Alice and Bob should receive sender key distribution messages from others
	for _, c := range []<-chan OutputMessage{alice.messageChan, bob.messageChan} {
		msg, success := timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Should receive a sender key distribution message from the group")
		assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "Should receive a sender key distribution message from the group")
	}

	// Alice and Bob should have each other's receiving sessions.
	assert.NotNil(t, aliceSessionDriver.HasUserReceivingSession(bob.GetUserID()), "Alice should have Bob's receiving session")
	assert.NotNil(t, bobSessionDriver.HasUserReceivingSession(alice.GetUserID()), "Bob should have Alice's receiving session")

	// TreeKEM should be the same.
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), bobSessionDriver.GetTreeKEMState()), "Alice and Bob should have the same TreeKEM state")

	// Alice send a message to the group
	err = alice.SendServerSideGroupMessage(groupId, []byte("Hello everyone! I'm Alice."), pb.MessageType_TEXT_MESSAGE, nil, false)
	assert.Nil(t, err, "Alice should be able to send a message to the group")

	// Bob should receive the message
	msg, success = timeOutReadFromMessageChannel(bob.messageChan)
	assert.True(t, success, "Bob should receive a Message from Alice")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Bob should receive a group text message from Alice")
	assert.Equal(t, "Hello everyone! I'm Alice.", string(msg.Message), "Bob should receive a group text message from Alice")

	// TreeKEM should be the same.
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), bobSessionDriver.GetTreeKEMState()), "Alice and Bob should have the same TreeKEM state")
	assert.True(t, multiTreeKemEqual(aliceSessionDriver.GetMultiTreeKEM(), bobSessionDriver.GetMultiTreeKEM()), "Alice and Bob should have the same MultiTreeKEM state")

	// Invite Carol to the group.
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_SERVER_SIDE, carol.GetUserID())

	// Carol should receive a group invitation from Alice
	msg, success = timeOutReadFromMessageChannel(carol.GetMessageChan())
	assert.True(t, success, "Carol should receive a group invitation from Alice")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "Carol should receive a group invitation from Alice")

	// Alice and Bob should receive a GROUP_ADDITION event.
	for _, c := range []<-chan OutputMessage{alice.messageChan, bob.messageChan} {
		msg, success = timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Alice and Bob should receive a group addition event")
		assert.Equal(t, pb.ServerEventType_GROUP_ADDITION, msg.EventType, "Alice and Bob should receive a group addition event")
	}

	// Carol should have the group session
	carolSessionDriver, err := carol.Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Carol should have the group session")

	err = carol.DistributeSelfSenderKeyToAll(groupId)
	assert.Nil(t, err, "Carol should be able to distribute her sender key to all")

	// Alice, Bob should receive sender key distribution messages from Carol
	for _, c := range []<-chan OutputMessage{alice.messageChan, bob.messageChan} {
		msg, success := timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Should receive a sender key distribution message from the group")
		assert.Equal(t, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, msg.MessageType, "Should receive a sender key distribution message from the group")
	}

	// Carol should receive sender key distribution messages from Alice and Bob
	for i := 0; i < 2; i++ {
		msg, success := timeOutReadFromMessageChannel(carol.messageChan)
		assert.True(t, success, "Should receive a sender key distribution message from the group")
		assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "Should receive a sender key distribution message from the group")
	}

	// Alice, Bob, and Carol should have each other's receiving sessions.
	assert.NotNil(t, aliceSessionDriver.HasUserReceivingSession(bob.GetUserID()), "Alice should have Bob's receiving session")
	assert.NotNil(t, aliceSessionDriver.HasUserReceivingSession(carol.GetUserID()), "Alice should have Carol's receiving session")
	assert.NotNil(t, bobSessionDriver.HasUserReceivingSession(alice.GetUserID()), "Bob should have Alice's receiving session")
	assert.NotNil(t, bobSessionDriver.HasUserReceivingSession(carol.GetUserID()), "Bob should have Carol's receiving session")
	assert.NotNil(t, carolSessionDriver.HasUserReceivingSession(alice.GetUserID()), "Carol should have Alice's receiving session")
	assert.NotNil(t, carolSessionDriver.HasUserReceivingSession(bob.GetUserID()), "Carol should have Bob's receiving session")

	// TreeKEM should be the same.
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), carolSessionDriver.GetTreeKEMState()), "Alice and Carol should have the same TreeKEM state")
	assert.True(t, treekemGroupEqual(bobSessionDriver.GetTreeKEMState(), carolSessionDriver.GetTreeKEMState()), "Bob and Carol should have the same TreeKEM state")
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), bobSessionDriver.GetTreeKEMState()), "Alice and Bob should have the same TreeKEM state")
	assert.True(t, multiTreeKemEqual(aliceSessionDriver.GetMultiTreeKEM(), carolSessionDriver.GetMultiTreeKEM()), "Alice and Carol should have the same MultiTreeKEM state")
	assert.True(t, multiTreeKemEqual(bobSessionDriver.GetMultiTreeKEM(), carolSessionDriver.GetMultiTreeKEM()), "Bob and Carol should have the same MultiTreeKEM state")
	assert.True(t, multiTreeKemEqual(aliceSessionDriver.GetMultiTreeKEM(), bobSessionDriver.GetMultiTreeKEM()), "Alice and Bob should have the same MultiTreeKEM state")

	// Bob send a message to the group
	err = bob.SendServerSideGroupMessage(groupId, []byte("Hello everyone! I'm Bob."), pb.MessageType_TEXT_MESSAGE, nil, false)
	assert.Nil(t, err, "Bob should be able to send a message to the group")

	// Alice should receive the message
	msg, success = timeOutReadFromMessageChannel(alice.messageChan)
	assert.True(t, success, "Alice should receive a message from Bob")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Alice should receive a group text message from Bob")

	// Carol should receive the message
	msg, success = timeOutReadFromMessageChannel(carol.messageChan)
	assert.True(t, success, "Carol should receive a message from Bob")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Carol should receive a group text message from Bob")

	// TreeKEM should be the same.
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), carolSessionDriver.GetTreeKEMState()), "Alice and Carol should have the same TreeKEM state")
	assert.True(t, treekemGroupEqual(bobSessionDriver.GetTreeKEMState(), carolSessionDriver.GetTreeKEMState()), "Bob and Carol should have the same TreeKEM state")
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), bobSessionDriver.GetTreeKEMState()), "Alice and Bob should have the same TreeKEM state")
	assert.True(t, multiTreeKemEqual(aliceSessionDriver.GetMultiTreeKEM(), carolSessionDriver.GetMultiTreeKEM()), "Alice and Carol should have the same MultiTreeKEM state")
	assert.True(t, multiTreeKemEqual(bobSessionDriver.GetMultiTreeKEM(), carolSessionDriver.GetMultiTreeKEM()), "Bob and Carol should have the same MultiTreeKEM state")
	assert.True(t, multiTreeKemEqual(aliceSessionDriver.GetMultiTreeKEM(), bobSessionDriver.GetMultiTreeKEM()), "Alice and Bob should have the same MultiTreeKEM state")

	// Carol invite David to the group.
	carol.RequestInviteUserToGroup(groupId, pb.GroupType_SERVER_SIDE, david.GetUserID())
	msg, success = timeOutReadFromMessageChannel(david.GetMessageChan())
	assert.True(t, success, "David should receive a group invitation from Carol")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "David should receive a group invitation from Carol")

	// David should have the group session
	davidSessionDriver, err := david.Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "David should have the group session")

	// Alice, Bob, and Carol should receive a GROUP_ADDITION event.
	for _, c := range []<-chan OutputMessage{alice.messageChan, bob.messageChan, carol.messageChan} {
		msg, success = timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Alice, Bob, and Carol should receive a group addition event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Alice, Bob, and Carol should receive a group addition event")
	}

	// David distribute his sender key to all
	err = david.DistributeSelfSenderKeyToAll(groupId)
	assert.Nil(t, err, "David should be able to distribute his sender key to all")

	// Alice, Bob, Carol should receive sender key distribution messages from David
	for _, c := range []<-chan OutputMessage{alice.messageChan, bob.messageChan, carol.messageChan} {
		msg, success = timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Should receive a sender key distribution message from the group")
		assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "Should receive a sender key distribution message from the group")
	}

	// David should receive sender key distribution messages from Alice, Bob, and Carol
	for i := 0; i < 3; i++ {
		msg, success := timeOutReadFromMessageChannel(david.messageChan)
		assert.True(t, success, "Should receive a sender key distribution message from the group")
		assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "Should receive a sender key distribution message from the group")
	}

	// TreeKEM should be the same
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
		for _, otherSessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
			assert.True(t, treekemGroupEqual(sessionDriver.GetTreeKEMState(), otherSessionDriver.GetTreeKEMState()), "All users should have the same TreeKEM state")
			assert.True(t, multiTreeKemEqual(sessionDriver.GetMultiTreeKEM(), otherSessionDriver.GetMultiTreeKEM()), "All users should have the same MultiTreeKEM state")
		}
	}

	// Carol send a message to the group
	err = carol.SendServerSideGroupMessage(groupId, []byte("Hello everyone! I'm Carol."), pb.MessageType_TEXT_MESSAGE, nil, false)
	assert.Nil(t, err, "Carol should be able to send a message to the group")

	// Alice, Bob, and David should receive the message
	for _, c := range []<-chan OutputMessage{alice.messageChan, bob.messageChan, david.messageChan} {
		msg, success = timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Should receive a message from Carol")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text message from Carol")
		assert.Equal(t, "Hello everyone! I'm Carol.", string(msg.Message), "Should receive a group text message from Carol")
	}

	// TreeKEM should be the same
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
		for _, otherSessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
			assert.True(t, treekemGroupEqual(sessionDriver.GetTreeKEMState(), otherSessionDriver.GetTreeKEMState()), "All users should have the same TreeKEM state")
			assert.True(t, multiTreeKemEqual(sessionDriver.GetMultiTreeKEM(), otherSessionDriver.GetMultiTreeKEM()), "All users should have the same MultiTreeKEM state")
		}
	}

}

// TestMlsGroupMessage test the MLS group Message.
func TestMlsGroupMessage(t *testing.T) {
	// Alice is the initiator of the group
	groupId, err := alice.CreateGroup(pb.GroupType_MLS)
	assert.Nil(t, err, "Alice should be able to create a group")

	// Alice should have the group session
	aliceSessionDriver, err := alice.Client.GetMlsGroupSessionDriver(groupId)
	assert.Nil(t, err, "Alice should have the group session")

	// Invite Bob to the group.
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_MLS, bob.GetUserID())
	msg, success := timeOutReadFromMessageChannel(bob.GetMessageChan())
	assert.True(t, success, "Bob should receive a group invitation from Alice")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "Bob should receive a group invitation from Alice")

	// Bob should have the group session
	bobSessionDriver, err := bob.Client.GetMlsGroupSessionDriver(groupId)
	assert.Nil(t, err, "Bob should have the group session")

	// Alice should receive a GROUP_ADDITION event.
	msg, success = timeOutReadFromMessageChannel(alice.messageChan)
	assert.True(t, success, "Alice should receive a group addition event")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Alice should receive a group addition event")

	// Alice and Bob should have the same state
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*bobSessionDriver.GetGroupState()), "Alice and Bob should have the same group state")

	// Alice send a message to the group
	err = alice.SendMlsGroupMessage(groupId, []byte("Hello everyone! I'm Alice."), pb.MessageType_TEXT_MESSAGE, nil, false)
	assert.Nil(t, err, "Alice should be able to send a message to the group")

	// Bob should receive the message
	msg, success = timeOutReadFromMessageChannel(bob.messageChan)
	assert.True(t, success, "Bob should receive a Message from Alice")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Bob should receive a group text message from Alice")
	assert.Equal(t, "Hello everyone! I'm Alice.", string(msg.Message), "Bob should receive a group text message from Alice")

	// Alice and Bob should have the same state
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*bobSessionDriver.GetGroupState()), "Alice and Bob should have the same group state")

	// Bob send a message to the group
	err = bob.SendMlsGroupMessage(groupId, []byte("Hello everyone! I'm Bob."), pb.MessageType_TEXT_MESSAGE, nil, false)
	assert.Nil(t, err, "Bob should be able to send a message to the group")

	// Alice should receive the message
	msg, success = timeOutReadFromMessageChannel(alice.messageChan)
	assert.True(t, success, "Alice should receive a message from Bob")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Alice should receive a group text message from Bob")
	assert.Equal(t, "Hello everyone! I'm Bob.", string(msg.Message), "Alice should receive a group text message from Bob")

	// Alice and Bob should have the same state
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*bobSessionDriver.GetGroupState()), "Alice and Bob should have the same group state")

	// Invite Carol to the group.
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_MLS, carol.GetUserID())
	msg, success = timeOutReadFromMessageChannel(carol.GetMessageChan())
	assert.True(t, success, "Carol should receive a group invitation from Alice")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "Carol should receive a group invitation from Alice")

	// Carol should have the group session
	carolSessionDriver, err := carol.Client.GetMlsGroupSessionDriver(groupId)
	assert.Nil(t, err, "Carol should have the group session")

	// Alice and Bob should receive a GROUP_ADDITION event.
	for _, c := range []<-chan OutputMessage{alice.messageChan, bob.messageChan} {
		msg, success = timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Alice and Bob should receive a group addition event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Alice and Bob should receive a group addition event")
	}

	// Alice, Bob, and Carol should have the same state
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*bobSessionDriver.GetGroupState()), "Alice and Bob should have the same group state")
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*carolSessionDriver.GetGroupState()), "Alice and Carol should have the same group state")
	assert.True(t, bobSessionDriver.GetGroupState().Equals(*carolSessionDriver.GetGroupState()), "Bob and Carol should have the same group state")

	// Carol send a message to the group
	err = carol.SendMlsGroupMessage(groupId, []byte("Hello everyone! I'm Carol."), pb.MessageType_TEXT_MESSAGE, nil, false)
	assert.Nil(t, err, "Carol should be able to send a message to the group")

	// Alice and Bob should receive the message
	for _, c := range []<-chan OutputMessage{alice.messageChan, bob.messageChan} {
		msg, success = timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Should receive a message from Carol")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text message from Carol")
		assert.Equal(t, "Hello everyone! I'm Carol.", string(msg.Message), "Should receive a group text message from Carol")
	}

	// Alice, Bob, and Carol should have the same state
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*bobSessionDriver.GetGroupState()), "Alice and Bob should have the same group state")
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*carolSessionDriver.GetGroupState()), "Alice and Carol should have the same group state")
	assert.True(t, bobSessionDriver.GetGroupState().Equals(*carolSessionDriver.GetGroupState()), "Bob and Carol should have the same group state")

	// Bob send a message to the group
	err = bob.SendMlsGroupMessage(groupId, []byte("Hello everyone! I'm Bob."), pb.MessageType_TEXT_MESSAGE, nil, false)
	assert.Nil(t, err, "Bob should be able to send a message to the group")

	// Alice and Carol should receive the message
	for _, c := range []<-chan OutputMessage{alice.messageChan, carol.messageChan} {
		msg, success = timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Should receive a message from Bob")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text message from Bob")
		assert.Equal(t, "Hello everyone! I'm Bob.", string(msg.Message), "Should receive a group text message from Bob")
	}

	// Alice, Bob, and Carol should have the same state
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*bobSessionDriver.GetGroupState()), "Alice and Bob should have the same group state")
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*carolSessionDriver.GetGroupState()), "Alice and Carol should have the same group state")
	assert.True(t, bobSessionDriver.GetGroupState().Equals(*carolSessionDriver.GetGroupState()), "Bob and Carol should have the same group state")

	// Carol invite David to the group.
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_MLS, david.GetUserID())
	msg, success = timeOutReadFromMessageChannel(david.GetMessageChan())
	assert.True(t, success, "David should receive a group invitation from Carol")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "David should receive a group invitation from Carol")

	// David should have the group session
	davidSessionDriver, err := david.Client.GetMlsGroupSessionDriver(groupId)
	assert.Nil(t, err, "David should have the group session")

	// Alice, Bob, and Carol should receive a GROUP_ADDITION event.
	for _, c := range []<-chan OutputMessage{alice.messageChan, bob.messageChan, carol.messageChan} {
		msg, success = timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Alice, Bob, and Carol should receive a group addition event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Alice, Bob, and Carol should receive a group addition event")
	}

	// Alice, Bob, Carol, and David should have the same state
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
		for _, otherSessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
			assert.True(t, sessionDriver.GetGroupState().Equals(*otherSessionDriver.GetGroupState()), "All users should have the same group state")
		}
	}

	// David send a message to the group
	err = david.SendMlsGroupMessage(groupId, []byte("Hello everyone! I'm David."), pb.MessageType_TEXT_MESSAGE, nil, false)
	assert.Nil(t, err, "David should be able to send a message to the group")

	// Alice, Bob, Carol should receive the message
	for _, c := range []<-chan OutputMessage{alice.messageChan, bob.messageChan, carol.messageChan} {
		msg, success = timeOutReadFromMessageChannel(c)
		assert.True(t, success, "Should receive a message from David")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text message from David")
		assert.Equal(t, "Hello everyone! I'm David.", string(msg.Message), "Should receive a group text message from David")
	}

	// Alice, Bob, Carol, and David should have the same state
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
		for _, otherSessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
			assert.True(t, sessionDriver.GetGroupState().Equals(*otherSessionDriver.GetGroupState()), "All users should have the same group state")
		}
	}
}

func TestDeactivate(t *testing.T) {
	// Alice send a message to Bob.
	_, err := alice.CreateIndividualSession(protocol.NewSignalAddress(bob.GetUserID(), 1))
	assert.Nil(t, err, "Alice should be able to create a session with Bob")

	err = alice.SendIndividualMessage(protocol.NewSignalAddress(bob.GetUserID(), 1), []byte("Hello Bob!"), pb.MessageType_TEXT_MESSAGE)
	assert.Nil(t, err, "Alice should be able to send message to Bob")
	msg, success := timeOutReadFromMessageChannel(bob.messageChan)
	assert.True(t, success, "Bob should receive a message from Alice")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Bob should receive a text message from Alice")
	assert.Equal(t, "Hello Bob!", string(msg.Message), "Bob should receive the same message from Alice")

	// Deactivate Bob
	bob.Deactivate()
	msg, success = timeOutReadFromMessageChannel(bob.messageChan)
	assert.True(t, success, "Bob should receive the deactivate message")
	assert.Equal(t, msg.Message, []byte("Deactivate"), "Bob should receive the deactivate message")

	// Alice send a message to Bob
	err = alice.SendIndividualMessage(protocol.NewSignalAddress(bob.GetUserID(), 1), []byte("Hello Bob again!"), pb.MessageType_TEXT_MESSAGE)
	assert.Nil(t, err, "Alice should be able to send message to Bob")

	// Bob should not receive the message
	msg, success = timeOutReadFromMessageChannel(bob.messageChan)
	assert.False(t, success, "Bob should not receive the message from Alice")
}

func dialer() func(context.Context, string) (net.Conn, error) {
	listener = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterChatServiceServer(s, &server.ServiceServer{})
	go func() {
		if err := s.Serve(listener); err != nil {
			log.Fatal(err)
		}
	}()

	return func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

// createClientSideUserWithRandomUserID create a client side user with a random user ID.
func createClientSideUserWithRandomUserID(prefix string) *ClientSideUser {
	userID := prefix + randomString(8)
	//user, _ := NewClientSideUser(userID, "localhost:50051", true)
	user := NewClientSideUserBufconn(userID, dialer(), true)
	return user
}

// randomString create a random string with the given length.
func randomString(length int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

// timeOutReadFromMessageChannel read from the given channel and return false if it times out.
func timeOutReadFromMessageChannel(ch <-chan OutputMessage) (OutputMessage, bool) {
	for {
		select {
		case msg := <-ch:
			return msg, true
		case <-time.After(5 * time.Second):
			return OutputMessage{}, false
		}
	}
}

func treekemNodeEqual(n1, n2 *treekem.Node) bool {
	return string(n1.Public) == string(n2.Public) && string(n1.SignPublic) == string(n2.SignPublic)
}

func treekemGroupEqual(g1, g2 *treekem.TreeKEMState) bool {
	if g1.Size() != g2.Size() {
		return false
	}

	for i := 0; i < g1.Size(); i++ {
		lhn := g1.Nodes()[i]
		rhn := g2.Nodes()[i]
		if lhn == nil || rhn == nil {
			continue
		}

		if !treekemNodeEqual(lhn, rhn) {
			return false
		}
	}

	return true
}

func multiTreeKemEqual(mt1, mt2 *treekem.MultiTreeKEM) bool {
	if !treekemGroupEqual(mt1.GetTreeKEM(), mt2.GetTreeKEM()) {
		return false
	}

	for key, root := range mt1.GetRoots() {
		if !bytes.Equal(root.Secret, mt2.GetRootSecret(key)) {
			return false
		}
	}

	return true
}
