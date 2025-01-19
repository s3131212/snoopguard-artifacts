package chatbot

import (
	"bytes"
	"chatbot-poc-go/pkg/client"
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/server"
	"chatbot-poc-go/pkg/treekem"
	"chatbot-poc-go/pkg/user"
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

var alice, bob, carol, david *user.ClientSideUser
var chatbot1, chatbot2, chatbot3, chatbot4 *ClientSideChatbot

const bufSize = 1024 * 1024

var listener *bufconn.Listener

func setup() {
	alice = createClientSideUserWithRandomUserID("alice")
	bob = createClientSideUserWithRandomUserID("bob")
	carol = createClientSideUserWithRandomUserID("carol")
	david = createClientSideUserWithRandomUserID("david")

	chatbot1 = createClientSideChatbotWithRandomUserID("chatbot1")
	chatbot2 = createClientSideChatbotWithRandomUserID("chatbot2")
	chatbot3 = createClientSideChatbotWithRandomUserID("chatbot3")
	chatbot4 = createClientSideChatbotWithRandomUserID("chatbot4")
}

func TestUserPreKeyAndSignedPreKey(t *testing.T) {
	setup()

	// Generate prekey and signed prekey
	// Note that this is only for test purpose, when setup=true, the prekeys are generated and uploaded by the time the ClientSideUser is initialized.
	chatbot1PreKeyID := chatbot1.Client.GeneratePreKey(0)
	chatbot1SignedPreKeyID := chatbot1.Client.GenerateSignedPreKey()
	assert.True(t, chatbot1.Client.UploadPreKeyByID(chatbot1PreKeyID), "Chatbot1 upload prekey should be successful")
	assert.True(t, chatbot1.Client.UploadSignedPreKeyByID(chatbot1SignedPreKeyID), "Chatbot1 upload signed prekey should be successful")

	chatbot2PreKeyID := chatbot2.Client.GeneratePreKey(0)
	chatbot2SignedPreKeyID := chatbot2.Client.GenerateSignedPreKey()
	assert.True(t, chatbot2.Client.UploadPreKeyByID(chatbot2PreKeyID), "Chatbot2 upload prekey should be successful")
	assert.True(t, chatbot2.Client.UploadSignedPreKeyByID(chatbot2SignedPreKeyID), "Chatbot2 upload signed prekey should be successful")

	chatbot3PreKeyID := chatbot3.Client.GeneratePreKey(0)
	chatbot3SignedPreKeyID := chatbot3.Client.GenerateSignedPreKey()
	assert.True(t, chatbot3.Client.UploadPreKeyByID(chatbot3PreKeyID), "Chatbot3 upload prekey should be successful")
	assert.True(t, chatbot3.Client.UploadSignedPreKeyByID(chatbot3SignedPreKeyID), "Chatbot3 upload signed prekey should be successful")

	// Test PreKeyBundle
	aliceGetChatbot1PreKeyBundle, err := alice.Client.GetOthersPreKeyBundle(chatbot1.GetChatbotID())
	assert.Nil(t, err, "Alice should be able to get chatbot1's prekey bundle")
	chatbot1GetAlicesPreKeyBundle, err := chatbot1.Client.GetOthersPreKeyBundle(alice.GetUserID())
	assert.Nil(t, err, "Chatbot1 should be able to get Alice's prekey bundle")

	// Get real PreKeyBundle
	alicePreKeyBundle := alice.Client.GetSelfPreKeyBundle(chatbot1GetAlicesPreKeyBundle.PreKeyID().Value, chatbot1GetAlicesPreKeyBundle.SignedPreKeyID())
	chatbot1PreKeyBundle := chatbot1.Client.GetSelfPreKeyBundle(aliceGetChatbot1PreKeyBundle.PreKeyID().Value, aliceGetChatbot1PreKeyBundle.SignedPreKeyID())

	assert.Equal(t, chatbot1GetAlicesPreKeyBundle.IdentityKey(), alicePreKeyBundle.IdentityKey(), "Alice's identity key should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetChatbot1PreKeyBundle.IdentityKey(), chatbot1PreKeyBundle.IdentityKey(), "Chatbot1's identity key should be the same as the one in the prekey bundle")

	assert.Equal(t, chatbot1GetAlicesPreKeyBundle.PreKeyID(), alicePreKeyBundle.PreKeyID(), "Alice's prekey ID should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetChatbot1PreKeyBundle.PreKeyID(), chatbot1PreKeyBundle.PreKeyID(), "Chatbot1's prekey ID should be the same as the one in the prekey bundle")

	assert.Equal(t, chatbot1GetAlicesPreKeyBundle.PreKey().Serialize(), alicePreKeyBundle.PreKey().Serialize(), "Alice's prekey should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetChatbot1PreKeyBundle.PreKey().Serialize(), chatbot1PreKeyBundle.PreKey().Serialize(), "Chatbot1's prekey should be the same as the one in the prekey bundle")

	assert.Equal(t, chatbot1GetAlicesPreKeyBundle.SignedPreKeyID(), alicePreKeyBundle.SignedPreKeyID(), "Alice's signed prekey ID should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetChatbot1PreKeyBundle.SignedPreKeyID(), chatbot1PreKeyBundle.SignedPreKeyID(), "Chatbot1's signed prekey ID should be the same as the one in the prekey bundle")

	assert.Equal(t, chatbot1GetAlicesPreKeyBundle.SignedPreKey().Serialize(), alicePreKeyBundle.SignedPreKey().Serialize(), "Alice's signed prekey should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetChatbot1PreKeyBundle.SignedPreKey().Serialize(), chatbot1PreKeyBundle.SignedPreKey().Serialize(), "Chatbot1's signed prekey should be the same as the one in the prekey bundle")

	assert.Equal(t, chatbot1GetAlicesPreKeyBundle.DeviceID(), alicePreKeyBundle.DeviceID(), "Alice's device ID should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetChatbot1PreKeyBundle.DeviceID(), chatbot1PreKeyBundle.DeviceID(), "Chatbot1's device ID should be the same as the one in the prekey bundle")

	assert.Equal(t, chatbot1GetAlicesPreKeyBundle.RegistrationID(), alicePreKeyBundle.RegistrationID(), "Alice's registration ID should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetChatbot1PreKeyBundle.RegistrationID(), chatbot1PreKeyBundle.RegistrationID(), "Chatbot1's registration ID should be the same as the one in the prekey bundle")

	assert.Equal(t, chatbot1GetAlicesPreKeyBundle.SignedPreKeySignature(), alicePreKeyBundle.SignedPreKeySignature(), "Alice's signed prekey signature should be the same as the one in the prekey bundle")
	assert.Equal(t, aliceGetChatbot1PreKeyBundle.SignedPreKeySignature(), chatbot1PreKeyBundle.SignedPreKeySignature(), "Chatbot1's signed prekey signature should be the same as the one in the prekey bundle")
}

func TestSendMessage(t *testing.T) {
	setup()

	// Alice to Chatbot1
	_, err := alice.CreateIndividualSession(protocol.NewSignalAddress(chatbot1.GetChatbotID(), 1))
	assert.Nil(t, err, "Alice should be able to create a session with chatbot1")
	for i := 0; i < 10; i++ {
		err := alice.SendIndividualMessage(protocol.NewSignalAddress(chatbot1.GetChatbotID(), 1), []byte(fmt.Sprintf("Hello Chatbot1! %v", i)), pb.MessageType_TEXT_MESSAGE)
		assert.Nil(t, err, "Alice should be able to send message to chatbot1")
		msg, success := timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
		assert.True(t, success, "Chatbot1 should receive a message from Alice")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot1 should receive a text message from Alice")
		assert.Equal(t, fmt.Sprintf("Hello Chatbot1! %v", i), string(msg.Message), "Chatbot1 should receive the same message from Alice")
	}

	// Chatbot1 to Alice
	_, err = chatbot1.CreateIndividualSession(protocol.NewSignalAddress(alice.GetUserID(), 1))
	assert.Nil(t, err, "Chatbot1 should be able to create a session with Alice")
	for i := 0; i < 10; i++ {
		err := chatbot1.SendIndividualMessage(protocol.NewSignalAddress(alice.GetUserID(), 1), []byte(fmt.Sprintf("Hello Alice! %v", i)), pb.MessageType_TEXT_MESSAGE)
		assert.Nil(t, err, "Chatbot1 should be able to send message to Alice")
		msg, success := timeOutReadFromUserMessageChannel(alice.GetMessageChan())
		assert.True(t, success, "Alice should receive a message from chatbot1")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Alice should receive a text message from chatbot1")
		assert.Equal(t, fmt.Sprintf("Hello Alice! %v", i), string(msg.Message), "Alice should receive the same message from chatbot1")
	}
}

// TestClientSideGroupMessage test the client side group Message.
func _TestClientSideGroupMessage(t *testing.T) {
	setup()

	// Alice is the initiator of the group
	groupId, err := alice.CreateGroup(pb.GroupType_CLIENT_SIDE)
	assert.Nil(t, err, "Alice should be able to create a group")

	// Alice should have the group session
	aliceSessionDriver, err := alice.Client.GetClientSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Alice should have the group session")

	// Invite Bob to the group.
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_CLIENT_SIDE, bob.GetUserID())
	msg, success := timeOutReadFromUserMessageChannel(bob.GetMessageChan())
	assert.True(t, success, "Bob should receive a group invitation from Alice")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "Bob should receive a group invitation from Alice")

	bobSessionDriver, err := bob.Client.GetClientSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Bob should have the group session")

	// Alice should receive a GROUP_ADDITION event.
	msg, success = timeOutReadFromUserMessageChannel(alice.GetMessageChan())
	assert.True(t, success, "Alice should receive a group addition event")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Alice should receive a group addition event")

	// Invite Carol to the group
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_CLIENT_SIDE, carol.GetUserID())
	msg, success = timeOutReadFromUserMessageChannel(carol.GetMessageChan())
	assert.True(t, success, "Carol should receive a group invitation from Alice")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "Carol should receive a group invitation from Alice")

	// Alice and Bob should receive a GROUP_ADDITION event.
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a group addition event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Should receive a group addition event")
	}

	carolSessionDriver, err := carol.Client.GetClientSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Carol should have the group session")

	// Alice send a message to the group
	err = alice.SendClientSideGroupMessage(groupId, []byte("Hello everyone! I'm Alice."), pb.MessageType_TEXT_MESSAGE, nil)
	assert.Nil(t, err, "Alice should be able to send a message to the group")

	// Bob should receive the Message
	msg, success = timeOutReadFromUserMessageChannel(bob.GetMessageChan())
	assert.True(t, success, "Bob should receive a Message from Alice")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Bob should receive a group text Message from Alice")
	assert.Equal(t, "Hello everyone! I'm Alice.", string(msg.Message), "Bob should receive a group text Message from Alice")

	// Carol should receive the Message
	msg, success = timeOutReadFromUserMessageChannel(carol.GetMessageChan())
	assert.True(t, success, "Carol should receive a Message from Alice")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Carol should receive a group text Message from Alice")
	assert.Equal(t, "Hello everyone! I'm Alice.", string(msg.Message), "Carol should receive a group text Message from Alice")

	// Alice, Bob, and Carol should share the same TreeKEM.
	for _, sessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		for _, otherSessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
			assert.True(t, treekemGroupEqual(sessionDriver.GetTreeKEMState(), otherSessionDriver.GetTreeKEMState()), "All users should have the same TreeKEM state")
			assert.True(t, multiTreeKemEqual(sessionDriver.GetMultiTreeKEM(), otherSessionDriver.GetMultiTreeKEM()), "All users should have the same MultiTreeKEM state")
		}
	}

	// Alice invite chatbot1 to the group
	alice.RequestInviteChatbotToGroup(groupId, pb.GroupType_CLIENT_SIDE, chatbot1.GetChatbotID(), false, false)

	// Alice, Bob, and Carol should receive GROUP_CHATBOT_ADDITION event
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a GROUP_CHATBOT_ADDITION event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "Should receive a GROUP_CHATBOT_ADDITION event")
	}

	// Chatbot1 should receive GROUP_CHATBOT_INVIATION event
	msgc, success := timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a GROUP_CHATBOT_INVIATION event")
	assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_CHATBOT_INVITATION, "Chatbot1 should receive a GROUP_CHATBOT_INVIATION event")

	// Chatbot1 should be in the group
	chatbot1SessionDriver, err := chatbot1.Client.GetClientSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Chatbot1 should have the group session")
	assert.Contains(t, aliceSessionDriver.GetGroupChatbots(), chatbot1.GetChatbotID(), "Chatbot1 should be in the chatbot list")

	// Chatbot1's MultiTreeKEMExternal root should match
	for _, sessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot1.GetChatbotID(), chatbot1SessionDriver.GetMultiTreeKEMExternal()))
	}

	// Carol send a message
	err = carol.SendClientSideGroupMessage(groupId, []byte("Hello everyone! I'm Carol."), pb.MessageType_TEXT_MESSAGE, []string{chatbot1.GetChatbotID()})
	assert.Nil(t, err, "Carol should be able to send a Message to the group")

	// Alice and Bob should receive the Message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Carol")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Carol")
		assert.Equal(t, "Hello everyone! I'm Carol.", string(msg.Message), "Should receive a group text Message from Carol")
	}

	// Chatbot1 should receive the Message
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a Message from Alice")
	assert.True(t, msgc.MessageType == pb.MessageType_TEXT_MESSAGE, "Chatbot1 should receive a group text Message from Alice")
	assert.Equal(t, "Hello everyone! I'm Carol.", string(msgc.Message), "Chatbot1 should receive a group text Message from Alice")

	// Chatbot1's treekem root should match the group's treekem root
	for _, sessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot1.GetChatbotID(), chatbot1SessionDriver.GetMultiTreeKEMExternal()))
	}

	// Bob send a message
	err = bob.SendClientSideGroupMessage(groupId, []byte("Hello everyone! I'm Bob. Chatbot should not receive this."), pb.MessageType_TEXT_MESSAGE, nil)
	assert.Nil(t, err, "Bob should be able to send a Message to the group")

	// Alice and Bob should receive the Message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Bob")
		assert.True(t, msg.MessageType == pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Bob")
		assert.Equal(t, "Hello everyone! I'm Bob. Chatbot should not receive this.", string(msg.Message), "Should receive a group text Message from Bob")
	}

	// Chatbot1 should not receive the Message
	_, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.False(t, success, "Chatbot1 should not receive a Message from Alice")

	// Chatbot1 should still be able to send a message
	err = chatbot1.SendClientSideGroupMessage(groupId, []byte("Hello everyone! I'm Chatbot1. I can speak."), pb.MessageType_TEXT_MESSAGE)
	assert.Nil(t, err, "Chatbot1 should be able to send a Message to the group")

	// Alice, Bob, and Carol should receive the message.
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Chatbot1")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Chatbot1")
		assert.Equal(t, "Hello everyone! I'm Chatbot1. I can speak.", string(msg.Message), "Should receive a group text Message from Chatbot1")
	}

	// Chatbot1's treekem root should still match the group's treekem root
	for _, sessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot1.GetChatbotID(), chatbot1SessionDriver.GetMultiTreeKEMExternal()))
	}

	// All users should have the same TreeKEM state
	for _, sessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		for _, otherSessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
			assert.True(t, treekemGroupEqual(sessionDriver.GetTreeKEMState(), otherSessionDriver.GetTreeKEMState()), "All users should have the same TreeKEM state")
			assert.True(t, multiTreeKemEqual(sessionDriver.GetMultiTreeKEM(), otherSessionDriver.GetMultiTreeKEM()), "All users should have the same MultiTreeKEM state")
		}
	}

	// Alice invite David to the group
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_CLIENT_SIDE, david.GetUserID())

	// Alice, Bob, and Carol should receive GROUP_ADDITION event
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a GROUP_ADDITION event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Should receive a GROUP_ADDITION event")
	}

	// Chatbot1 should receive GROUP_ADDITION event
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a GROUP_ADDITION event")
	assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_ADDITION, "Chatbot1 should receive a GROUP_ADDITION event")

	// David should receive GROUP_INVITATION event
	msg, success = timeOutReadFromUserMessageChannel(david.GetMessageChan())
	assert.True(t, success, "David should receive a GROUP_INVITATION event")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "David should receive a GROUP_INVITATION event")

	davidSessionDriver, err := david.Client.GetClientSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "David should have the group session")

	// Alice send a message
	err = alice.SendClientSideGroupMessage(groupId, []byte("Here's David."), pb.MessageType_TEXT_MESSAGE, []string{chatbot1.GetChatbotID()})
	assert.Nil(t, err, "Alice should be able to send a Message to the group")

	// Bob and Carol should receive the Message even after David joins.
	for _, c := range []<-chan user.OutputMessage{bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Alice")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Bob and Carol should receive a group text Message from Alice")
		assert.Equal(t, "Here's David.", string(msg.Message), "Bob and Carol should receive a group text Message from Alice")
	}

	// David should receive the Message
	msg, success = timeOutReadFromUserMessageChannel(david.GetMessageChan())
	assert.True(t, success, "David should receive a Message from Alice")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "David should receive a group text Message from Alice")
	assert.Equal(t, "Here's David.", string(msg.Message), "David should receive a group text Message from Alice")

	// Chatbot1 should receive the Message
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a Message from Alice")
	assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot1 should receive a group text Message from Alice")
	assert.Equal(t, "Here's David.", string(msgc.Message), "Chatbot1 should receive a group text Message from Alice")

	// Chatbot1's treekem root should match the group's treekem root
	for _, sessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot1.GetChatbotID(), chatbot1SessionDriver.GetMultiTreeKEMExternal()))
	}

	// David should be able to send a Message to the group
	err = david.SendClientSideGroupMessage(groupId, []byte("Hello everyone! I'm David."), pb.MessageType_TEXT_MESSAGE, nil)
	assert.Nil(t, err, "David should be able to send a Message to the group")

	// Alice, Bob, and Carol should receive the Message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from David")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from David")
		assert.Equal(t, "Hello everyone! I'm David.", string(msg.Message), "Should receive a group text Message from David")
	}

	// Chatbot1 should not receive the Message
	_, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.False(t, success, "Chatbot1 should not receive a Message from Alice")

	// Chatbot1's treekem root should still match the group's treekem root
	for _, sessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot1.GetChatbotID(), chatbot1SessionDriver.GetMultiTreeKEMExternal()))
	}

	// Chatbot1 send a Message
	err = chatbot1.SendClientSideGroupMessage(groupId, []byte("Hello everyone! I'm Chatbot1."), pb.MessageType_TEXT_MESSAGE)
	assert.Nil(t, err, "Chatbot1 should be able to send a Message to the group")

	// Alice, Bob, Carol, and David should receive the message.
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan(), david.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Chatbot1")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Chatbot1")
		assert.Equal(t, "Hello everyone! I'm Chatbot1.", string(msg.Message), "Should receive a group text Message from Chatbot1")
	}

	// Chatbot1's treekem root should match the group's treekem root
	for i, sessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot1.GetChatbotID(), chatbot1SessionDriver.GetMultiTreeKEMExternal()), fmt.Sprintf("Chatbot1's treekem root should match the group's treekem root for user %v", i))
	}

	// Remove Carol from the group
	bob.RequestRemoveUserFromGroup(groupId, carol.GetUserID())

	// All members, including Carol, should receive the GROUP_REMOVAL event
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan(), david.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a GROUP_REMOVAL event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_REMOVAL, "Should receive a GROUP_REMOVAL event")
	}

	// Chatbot1 should receive the GROUP_REMOVAL event
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a GROUP_REMOVAL event")
	assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_REMOVAL, "Chatbot1 should receive a GROUP_REMOVAL event")

	// Carol should not have the group session
	carolSessionDriver, err = carol.Client.GetClientSideGroupSessionDriver(groupId)
	assert.Nil(t, carolSessionDriver, "Carol should not have the group session")
	assert.NotNil(t, err, "Carol should not have the group session")

	// Carol should not be in the participant list
	for _, sessionDriver := range []*client.ClientSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, davidSessionDriver} {
		assert.NotContains(t, sessionDriver.GetGroupParticipants(), carol.GetUserID(), "Carol should not be in the participant list")
	}

	// Alice send a message.
	err = alice.SendClientSideGroupMessage(groupId, []byte("Bye Carol."), pb.MessageType_TEXT_MESSAGE, nil)
	assert.Nil(t, err, "Alice should be able to send a Message to the group")

	// Carol should not receive the message
	msg, success = timeOutReadFromUserMessageChannel(carol.GetMessageChan())
	assert.False(t, success, "Carol should not receive a Message from Alice")

	// Bob and David should receive the message
	for _, c := range []<-chan user.OutputMessage{bob.GetMessageChan(), david.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Alice")
		assert.True(t, msg.MessageType == pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Alice")
		assert.Equal(t, "Bye Carol.", string(msg.Message), "Should receive a group text Message from Alice")
	}
}

func TestServerSideGroupMessage(t *testing.T) {
	setup()

	// Alice is the initiator of the group
	groupId, err := alice.CreateGroup(pb.GroupType_SERVER_SIDE)
	assert.Nil(t, err, "Alice should be able to create a group")

	// Alice should have the group session
	aliceSessionDriver, err := alice.Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Alice should have the group session")

	// Invite Bob to the group.
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_SERVER_SIDE, bob.GetUserID())
	msg, success := timeOutReadFromUserMessageChannel(bob.GetMessageChan())
	assert.True(t, success, "Bob should receive a group invitation from Alice")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "Bob should receive a group invitation from Alice")

	// Bob should have the group session
	bobSessionDriver, err := bob.Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Bob should have the group session")

	// Alice should receive a GROUP_ADDITION event.
	msg, success = timeOutReadFromUserMessageChannel(alice.GetMessageChan())
	assert.True(t, success, "Alice should receive a group addition event")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Alice should receive a group addition event")

	// Bob distribute his sender key to all
	err = bob.DistributeSelfSenderKeyToAll(groupId)
	assert.Nil(t, err, "Bob should be able to distribute his sender key to all")

	// Alice and Bob should receive sender key distribution messages from others
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan()} {
		msg, success := timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a sender key distribution message from the group")
		assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "Should receive a sender key distribution message from the group")
	}

	// Alice and Bob should have each other's receiving sessions.
	assert.True(t, aliceSessionDriver.HasUserReceivingSession(bob.GetUserID()), "Alice should have Bob's receiving session")
	assert.True(t, bobSessionDriver.HasUserReceivingSession(alice.GetUserID()), "Bob should have Alice's receiving session")

	// TreeKEM should be the same.
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), bobSessionDriver.GetTreeKEMState()), "Alice and Bob should have the same TreeKEM state")

	// Alice send a message to the group
	err = alice.SendServerSideGroupMessage(groupId, []byte("Hello everyone! I'm Alice."), pb.MessageType_TEXT_MESSAGE, nil, false)
	assert.Nil(t, err, "Alice should be able to send a message to the group")

	// Bob should receive the message
	msg, success = timeOutReadFromUserMessageChannel(bob.GetMessageChan())
	assert.True(t, success, "Bob should receive a Message from Alice")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Bob should receive a group text message from Alice")
	assert.Equal(t, "Hello everyone! I'm Alice.", string(msg.Message), "Bob should receive a group text message from Alice")

	// TreeKEM should be the same.
	assert.True(t, treekemGroupEqual(aliceSessionDriver.GetTreeKEMState(), bobSessionDriver.GetTreeKEMState()), "Alice and Bob should have the same TreeKEM state")
	assert.True(t, multiTreeKemEqual(aliceSessionDriver.GetMultiTreeKEM(), bobSessionDriver.GetMultiTreeKEM()), "Alice and Bob should have the same MultiTreeKEM state")

	// Invite Carol to the group.
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_SERVER_SIDE, carol.GetUserID())

	// Carol should receive a group invitation from Alice
	msg, success = timeOutReadFromUserMessageChannel(carol.GetMessageChan())
	assert.True(t, success, "Carol should receive a group invitation from Alice")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "Carol should receive a group invitation from Alice")

	// Carol should have the group session
	carolSessionDriver, err := carol.Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Carol should have the group session")

	// Alice and Bob should receive a GROUP_ADDITION event.
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Alice and Bob should receive a group addition event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Alice and Bob should receive a group addition event")
	}

	// Carol distribute her sender key to all
	err = carol.DistributeSelfSenderKeyToAll(groupId)
	assert.Nil(t, err, "Carol should be able to distribute her sender key to all")

	// Alice, Bob should receive sender key distribution messages from Carol
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan()} {
		msg, success := timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a sender key distribution message from the group")
		assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "Should receive a sender key distribution message from the group")
	}

	// Carol should receive sender key distribution messages from Alice and Bob
	for i := 0; i < 2; i++ {
		msg, success := timeOutReadFromUserMessageChannel(carol.GetMessageChan())
		assert.True(t, success, "Should receive a sender key distribution message from the group")
		assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "Should receive a sender key distribution message from the group")
	}

	// Alice, Bob, and Carol should have each other's receiving sessions.
	assert.True(t, aliceSessionDriver.HasUserReceivingSession(bob.GetUserID()), "Alice should have Bob's receiving session")
	assert.True(t, aliceSessionDriver.HasUserReceivingSession(carol.GetUserID()), "Alice should have Carol's receiving session")
	assert.True(t, bobSessionDriver.HasUserReceivingSession(alice.GetUserID()), "Bob should have Alice's receiving session")
	assert.True(t, bobSessionDriver.HasUserReceivingSession(carol.GetUserID()), "Bob should have Carol's receiving session")
	assert.True(t, carolSessionDriver.HasUserReceivingSession(alice.GetUserID()), "Carol should have Alice's receiving session")
	assert.True(t, carolSessionDriver.HasUserReceivingSession(bob.GetUserID()), "Carol should have Bob's receiving session")

	// Alice invite chatbot1 to the group
	alice.RequestInviteChatbotToGroup(groupId, pb.GroupType_SERVER_SIDE, chatbot1.GetChatbotID(), false, false)

	// Alice, Bob, and Carol should receive GROUP_CHATBOT_ADDITION event
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a GROUP_CHATBOT_ADDITION event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "Should receive a GROUP_CHATBOT_ADDITION event")
	}

	// Chatbot1 should receive GROUP_CHATBOT_INVIATION event
	msgc, success := timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a GROUP_CHATBOT_INVIATION event")
	assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_CHATBOT_INVITATION, "Chatbot1 should receive a GROUP_CHATBOT_INVIATION event")

	// Chatbot1 should be in the group
	chatbot1SessionDriver, err := chatbot1.Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Chatbot1 should have the group session")
	assert.Contains(t, aliceSessionDriver.GetGroupChatbots(), chatbot1.GetChatbotID(), "Chatbot1 should be in the chatbot list")

	// Chatbot1 distribute its sender key to all
	err = chatbot1.DistributeSelfSenderKeyToAll(groupId)
	assert.Nil(t, err, "Chatbot1 should be able to distribute its sender key to all")

	// Alice, Bob, and Carol should receive sender key distribution messages from chatbot1
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success := timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a sender key distribution Message from Chatbot1")
		assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "Should receive a sender key distribution Message from Chatbot1")
	}

	// Chatbot1 should receive sender key distribution messages from Alice, Bob, and Carol
	for i := 0; i < 3; i++ {
		msg, success := timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
		assert.True(t, success, "Should receive a sender key distribution Message from Alice, Bob, and Carol")
		assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "Should receive a sender key distribution Message from Alice, Bob, and Carol")
	}

	// Alice, Bob, and Carol should have Chatbot1's sender keys
	assert.True(t, aliceSessionDriver.HasUserReceivingSession(chatbot1.GetChatbotID()), "Alice should have Chatbot1's sender key")
	assert.True(t, bobSessionDriver.HasUserReceivingSession(chatbot1.GetChatbotID()), "Bob should have Chatbot1's sender key")
	assert.True(t, carolSessionDriver.HasUserReceivingSession(chatbot1.GetChatbotID()), "Carol should have Chatbot1's sender key")

	// Chatbot1 should have Alice's, Bob's, and Carol's sender keys
	assert.True(t, chatbot1SessionDriver.HasUserReceivingSession(alice.GetUserID()), "Chatbot1 should have Alice's sender key")
	assert.True(t, chatbot1SessionDriver.HasUserReceivingSession(bob.GetUserID()), "Chatbot1 should have Bob's sender key")
	assert.True(t, chatbot1SessionDriver.HasUserReceivingSession(carol.GetUserID()), "Chatbot1 should have Carol's sender key")

	// Chatbot1's treekem root should match the group's treekem root
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot1.GetChatbotID(), chatbot1SessionDriver.GetMultiTreeKEMExternal()))
	}

	// Carol send a message
	err = carol.SendServerSideGroupMessage(groupId, []byte("Hello everyone! I'm Carol."), pb.MessageType_TEXT_MESSAGE, []string{chatbot1.GetChatbotID()}, false)
	assert.Nil(t, err, "Carol should be able to send a Message to the group")

	// Alice and Bob should receive the Message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Carol")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Carol")
		assert.Equal(t, "Hello everyone! I'm Carol.", string(msg.Message), "Should receive a group text Message from Carol")
	}

	// Chatbot1 should receive the Message
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a Message from Alice")
	assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot1 should receive a group text Message from Alice")
	assert.Equal(t, "Hello everyone! I'm Carol.", string(msgc.Message), "Chatbot1 should receive a group text Message from Alice")

	// Chatbot1's treekem root should match the group's treekem root
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot1.GetChatbotID(), chatbot1SessionDriver.GetMultiTreeKEMExternal()))
	}

	// Bob send a message
	err = bob.SendServerSideGroupMessage(groupId, []byte("Hello everyone! I'm Bob. Chatbot should not receive this."), pb.MessageType_TEXT_MESSAGE, nil, false)
	assert.Nil(t, err, "Bob should be able to send a Message to the group")

	// Alice and Bob should receive the Message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Bob")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Bob")
		assert.Equal(t, "Hello everyone! I'm Bob. Chatbot should not receive this.", string(msg.Message), "Should receive a group text Message from Bob")
	}

	// Chatbot1 should not receive the Message
	_, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.False(t, success, "Chatbot1 should not receive a Message from Alice")

	// Chatbot1's treekem root should still match the group's treekem root
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot1.GetChatbotID(), chatbot1SessionDriver.GetMultiTreeKEMExternal()))
	}

	// Chatbot1 should still be able to send a message
	err = chatbot1.SendServerSideGroupMessage(groupId, []byte("Hello everyone! I'm Chatbot1. I can speak."), pb.MessageType_TEXT_MESSAGE)
	assert.Nil(t, err, "Chatbot1 should be able to send a Message to the group")

	// Alice, Bob, and Carol should receive the message.
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Chatbot1")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Chatbot1")
		assert.Equal(t, "Hello everyone! I'm Chatbot1. I can speak.", string(msg.Message), "Should receive a group text Message from Chatbot1")
	}

	// Chatbot1's treekem root should match the group's treekem root
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot1.GetChatbotID(), chatbot1SessionDriver.GetMultiTreeKEMExternal()))
	}

	// All users should have the same TreeKEM state
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		for _, otherSessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
			assert.True(t, treekemGroupEqual(sessionDriver.GetTreeKEMState(), otherSessionDriver.GetTreeKEMState()), "All users should have the same TreeKEM state")
			assert.True(t, multiTreeKemEqual(sessionDriver.GetMultiTreeKEM(), otherSessionDriver.GetMultiTreeKEM()), "All users should have the same MultiTreeKEM state")
		}
	}

	// Invite chatbot2 to the group with IGA
	alice.RequestInviteChatbotToGroup(groupId, pb.GroupType_SERVER_SIDE, chatbot2.GetChatbotID(), true, false)

	// Alice, Bob, and Carol should receive GROUP_CHATBOT_ADDITION event
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a GROUP_CHATBOT_ADDITION event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "Should receive a GROUP_CHATBOT_ADDITION event")
	}

	// Chatbot2 should receive GROUP_CHATBOT_INVIATION event
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot2.GetMessageChan())
	assert.True(t, success, "Chatbot2 should receive a GROUP_CHATBOT_INVIATION event")
	assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_CHATBOT_INVITATION, "Chatbot2 should receive a GROUP_CHATBOT_INVIATION event")

	// Chatbot2 should be in the group
	chatbot2SessionDriver, err := chatbot2.Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Chatbot2 should have the group session")

	// Bob tries to send a message to both chatbot
	err = bob.SendServerSideGroupMessage(groupId, []byte("This is an IGA message. Chatbot 2 should receive it without learning my identity"), pb.MessageType_TEXT_MESSAGE, []string{chatbot1.GetChatbotID(), chatbot2.GetChatbotID()}, false)
	assert.Nil(t, err, "Bob should be able to send an IGA message to chatbots")

	// Alice and Carol should receive the message.
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Bob")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Bob")
		assert.Equal(t, "This is an IGA message. Chatbot 2 should receive it without learning my identity", string(msg.Message), "Should receive a group text Message from Bob")
	}

	// Chatbot1 should receive the message
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a Message from Bob")
	assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot2 should receive a group text Message from Bob")
	assert.Equal(t, "This is an IGA message. Chatbot 2 should receive it without learning my identity", string(msgc.Message), "Chatbot1 should receive a group text Message from Bob")

	// Chatbot2 should receive the message
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot2.GetMessageChan())
	assert.True(t, success, "Chatbot2 should receive a Message from Bob")
	assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot2 should receive a group text Message from Bob")
	assert.Equal(t, "This is an IGA message. Chatbot 2 should receive it without learning my identity", string(msgc.Message), "Chatbot2 should receive a group text Message from Bob")

	// Alice, Bob, and Carol should receive the validation message.
	//for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
	//	msg, success = timeOutReadFromUserMessageChannel(c)
	//	assert.True(t, success, "Should receive a validation message from the group")
	//	assert.Equal(t, msg.MessageType, pb.MessageType_VALIDATION_MESSAGE, "Should receive a validation message from the group")
	//}

	// Chatbot1's treekem root should match the group's treekem root
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot1.GetChatbotID(), chatbot1SessionDriver.GetMultiTreeKEMExternal()))
	}

	// Chatbot2's treekem root should match the group's treekem root
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot2.GetChatbotID(), chatbot2SessionDriver.GetMultiTreeKEMExternal()))
	}

	// All users should have the same TreeKEM state
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		for _, otherSessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
			assert.True(t, treekemGroupEqual(sessionDriver.GetTreeKEMState(), otherSessionDriver.GetTreeKEMState()), "All users should have the same TreeKEM state")
			assert.True(t, multiTreeKemEqual(sessionDriver.GetMultiTreeKEM(), otherSessionDriver.GetMultiTreeKEM()), "All users should have the same MultiTreeKEM state")
		}
	}

	// Chatbot2 tries to send a message to the group
	err = chatbot2.SendServerSideGroupMessage(groupId, []byte("Hello everyone! I'm Chatbot2."), pb.MessageType_TEXT_MESSAGE)

	// Alice, Bob, and Carol should receive the message.
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Chatbot2")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Chatbot2")
		assert.Equal(t, "Hello everyone! I'm Chatbot2.", string(msg.Message), "Should receive a group text Message from Chatbot2")
	}

	// Bob invites Chatbot3 with pseudonymity
	bob.RequestInviteChatbotToGroup(groupId, pb.GroupType_SERVER_SIDE, chatbot3.GetChatbotID(), true, true)

	// Alice, Bob, and Carol should receive GROUP_CHATBOT_ADDITION event
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a GROUP_CHATBOT_ADDITION event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "Should receive a GROUP_CHATBOT_ADDITION event")
	}

	// Chatbot3 should receive GROUP_CHATBOT_INVIATION event
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot3.GetMessageChan())
	assert.True(t, success, "Chatbot3 should receive a GROUP_CHATBOT_INVIATION event")
	assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_CHATBOT_INVITATION, "Chatbot3 should receive a GROUP_CHATBOT_INVIATION event")

	// Chatbot3 should be in the group
	chatbot3SessionDriver, err := chatbot3.Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "Chatbot3 should have the group session")

	// Alice issue a pseudonym
	err = alice.CreateAndRegisterServerSidePseudonym(groupId, chatbot3.GetChatbotID())
	assert.Nil(t, err, "Alice should be able to issue a pseudonym")

	// Chatbot3 should receive a pseudonym registration message from Alice
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot3.GetMessageChan())
	assert.True(t, success, "Chatbot3 should receive a pseudonym registration message from Alice")
	assert.Equal(t, msgc.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "Chatbot3 should receive a pseudonym registration message from Alice")

	// Bob and Carol should also receive a pseudonym registration message from Alice
	for _, c := range []<-chan user.OutputMessage{bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a pseudonym registration message from Alice")
		assert.Equal(t, msg.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "Should receive a pseudonym registration message from Alice")
	}

	time.Sleep(500 * time.Millisecond) // TODO: fix race condition

	// Alice send a message to chatbot 3
	err = alice.SendServerSideGroupMessage(groupId, []byte("Hello Chatbot 3! I'm Alice."), pb.MessageType_TEXT_MESSAGE, []string{chatbot3.GetChatbotID()}, false)
	assert.Nil(t, err, "Alice should be able to send a Message to the group")

	// Chatbot3 should receive a message from Alice
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot3.GetMessageChan())
	assert.True(t, success, "Chatbot3 should receive a Message from Alice")
	assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot3 should receive a group text Message from Alice")
	assert.Equal(t, "Hello Chatbot 3! I'm Alice.", string(msgc.Message), "Chatbot3 should receive a group text Message from Alice")

	// Bob and Carol should receive the message
	for _, c := range []<-chan user.OutputMessage{bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Alice")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Bob and Carol should receive a group text Message from Alice")
		assert.Equal(t, "Hello Chatbot 3! I'm Alice.", string(msg.Message), "Bob and Carol should receive a group text Message from Alice")
	}

	// Alice, Bob, and Carol should receive the validation message.
	//for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
	//	msg, success = timeOutReadFromUserMessageChannel(c)
	//	assert.True(t, success, "Should receive a validation message from the group")
	//	assert.Equal(t, msg.MessageType, pb.MessageType_VALIDATION_MESSAGE, "Should receive a validation message from the group")
	//}

	assert.Equal(t, 1, len(chatbot3SessionDriver.GetGroupParticipants()), "Chatbot3 should have Alice as the only participant.")

	// Bob issues a pseudonym
	err = bob.CreateAndRegisterServerSidePseudonym(groupId, chatbot3.GetChatbotID())
	assert.Nil(t, err, "Bob should be able to issue a pseudonym")

	// Chatbot3 should receive a pseudonym registration message from Bob
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot3.GetMessageChan())
	assert.True(t, success, "Chatbot3 should receive a pseudonym registration message from Bob")
	assert.Equal(t, msgc.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "Chatbot3 should receive a pseudonym registration message from Bob")

	// Alice and Carol should also receive a pseudonym registration message from Bob
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a pseudonym registration message from Bob")
		assert.Equal(t, msg.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "Should receive a pseudonym registration message from Bob")
	}

	time.Sleep(500 * time.Millisecond) // TODO: fix race condition

	// Bob send a message to all chatbots.
	err = bob.SendServerSideGroupMessage(groupId, []byte("Hello Chatbots! I'm Bob."), pb.MessageType_TEXT_MESSAGE, []string{chatbot1.GetChatbotID(), chatbot2.GetChatbotID(), chatbot3.GetChatbotID()}, false)

	// Alice and Carol should receive the message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Bob")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Bob")
		assert.Equal(t, "Hello Chatbots! I'm Bob.", string(msg.Message), "Should receive a group text Message from Bob")
	}

	// All chatbots should receive the message.
	for _, c := range []<-chan OutputMessage{chatbot1.GetMessageChan(), chatbot2.GetMessageChan(), chatbot3.GetMessageChan()} {
		msgc, success = timeOutReadFromChatbotMessageChannel(c)
		assert.True(t, success, "Chatbots should receive a Message from Bob")
		assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbots should receive a group text Message from Bob")
		assert.Equal(t, "Hello Chatbots! I'm Bob.", string(msgc.Message), "Chatbots should receive a group text Message from Bob")
	}

	// Alice, Bob, and Carol should receive the validation message from both chatbot2 and chatbot3.
	//for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
	//	for i := 0; i < 2; i++ {
	//		msg, success = timeOutReadFromUserMessageChannel(c)
	//		assert.True(t, success, "Should receive a validation message from the group")
	//		assert.Equal(t, msg.MessageType, pb.MessageType_VALIDATION_MESSAGE, "Should receive a validation message from the group")
	//	}
	//
	//}

	// All members should have the same TreeKEM state
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		for _, otherSessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
			assert.True(t, treekemGroupEqual(sessionDriver.GetTreeKEMState(), otherSessionDriver.GetTreeKEMState()), "All users should have the same TreeKEM state")
			assert.True(t, multiTreeKemEqual(sessionDriver.GetMultiTreeKEM(), otherSessionDriver.GetMultiTreeKEM()), "All users should have the same MultiTreeKEM state")
		}
	}

	// All chatbots should match all members Multi-TreeKEM state.
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot1.GetChatbotID(), chatbot1SessionDriver.GetMultiTreeKEMExternal()))
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot2.GetChatbotID(), chatbot2SessionDriver.GetMultiTreeKEMExternal()))
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot3.GetChatbotID(), chatbot3SessionDriver.GetMultiTreeKEMExternal()))
	}

	// Chatbot 3 sends a message to the group
	err = chatbot3.SendServerSideGroupMessage(groupId, []byte("Hello everyone! I'm Chatbot3."), pb.MessageType_TEXT_MESSAGE)
	assert.Nil(t, err, "Chatbot3 should be able to send a message to the group")

	// Alice, Bob, and Carol should receive the message.
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Chatbot3")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Chatbot3")
		assert.Equal(t, "Hello everyone! I'm Chatbot3.", string(msg.Message), "Should receive a group text Message from Chatbot3")
	}

	// Chatbot 3 should match all members Multi-TreeKEM state.
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot3.GetChatbotID(), chatbot3SessionDriver.GetMultiTreeKEMExternal()))
	}

	// Alice invite David to the group
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_SERVER_SIDE, david.GetUserID())

	// Alice, Bob, and Carol should receive GROUP_ADDITION event
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a GROUP_ADDITION event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Should receive a GROUP_ADDITION event")
	}

	// David should receive GROUP_INVITATION event
	msg, success = timeOutReadFromUserMessageChannel(david.GetMessageChan())
	assert.True(t, success, "David should receive a GROUP_INVITATION event")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "David should receive a GROUP_INVITATION event")

	// David should be in the group
	davidSessionDriver, err := david.Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "David should have the group session")

	// Chatbot1 should receive GROUP_ADDITION event
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a GROUP_ADDITION event")
	assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_ADDITION, "Chatbot1 should receive a GROUP_ADDITION event")

	// David distribute his sender key to all
	err = david.DistributeSelfSenderKeyToAll(groupId)
	assert.Nil(t, err, "David should be able to distribute his sender key to all")

	// Alice, Bob, and Carol should receive sender key distribution messages from David
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success := timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a sender key distribution Message from David")
		assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "Should receive a sender key distribution Message from David")
	}

	// Chatbot1 should receive sender key distribution messages from David
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a sender key distribution Message from David")
	assert.Equal(t, msgc.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "Chatbot1 should receive a sender key distribution Message from David")

	// David should receive sender key distribution messages from Alice, Bob, Carol, and Chatbot1
	for i := 0; i < 4; i++ {
		msg, success := timeOutReadFromUserMessageChannel(david.GetMessageChan())
		assert.True(t, success, "Should receive a sender key distribution Message from Alice, Bob, Carol, and Chatbot1")
		assert.Equal(t, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, msg.MessageType, "Should receive a sender key distribution Message from Alice, Bob, Carol, and Chatbot1")
	}

	// Alice, Bob, and Carol should have David's sender keys
	assert.True(t, aliceSessionDriver.HasUserReceivingSession(david.GetUserID()), "Alice should have David's sender key")
	assert.True(t, bobSessionDriver.HasUserReceivingSession(david.GetUserID()), "Bob should have David's sender key")
	assert.True(t, carolSessionDriver.HasUserReceivingSession(david.GetUserID()), "Carol should have David's sender key")

	// David should have Alice, Bob, and Carol's sender keys
	assert.True(t, davidSessionDriver.HasUserReceivingSession(alice.GetUserID()), "David should have Alice's sender key")
	assert.True(t, davidSessionDriver.HasUserReceivingSession(bob.GetUserID()), "David should have Bob's sender key")
	assert.True(t, davidSessionDriver.HasUserReceivingSession(carol.GetUserID()), "David should have Carol's sender key")

	// Chatbot1 should have David's sender keys
	assert.True(t, chatbot1SessionDriver.HasUserReceivingSession(david.GetUserID()), "Chatbot1 should have David's sender key")

	// Chatbot 2 should not have David's sender keys.
	assert.False(t, chatbot2SessionDriver.HasUserReceivingSession(david.GetUserID()), "Chatbot2 should not have David's sender key")

	// Alice send a message
	err = alice.SendServerSideGroupMessage(groupId, []byte("Here's David."), pb.MessageType_TEXT_MESSAGE, []string{chatbot1.GetChatbotID()}, false)
	assert.Nil(t, err, "Alice should be able to send a Message to the group")

	// Bob and Carol should receive the message even after David joins.
	for _, c := range []<-chan user.OutputMessage{bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Alice")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Bob and Carol should receive a group text Message from Alice")
		assert.Equal(t, "Here's David.", string(msg.Message), "Bob and Carol should receive a group text Message from Alice")
	}

	// David should receive the message
	msg, success = timeOutReadFromUserMessageChannel(david.GetMessageChan())
	assert.True(t, success, "David should receive a Message from Alice")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "David should receive a group text Message from Alice")
	assert.Equal(t, "Here's David.", string(msg.Message), "David should receive a group text Message from Alice")

	// Chatbot1 should receive the message
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a Message from Alice")
	assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot1 should receive a group text Message from Alice")
	assert.Equal(t, "Here's David.", string(msgc.Message), "Chatbot1 should receive a group text Message from Alice")

	// Note: at this moment, David's multi-treekem would not match chatbots' multi-treekem because no one has issued the update.

	// David should be able to send a message to the group
	err = david.SendServerSideGroupMessage(groupId, []byte("Hello everyone! I'm David."), pb.MessageType_TEXT_MESSAGE, nil, false)
	assert.Nil(t, err, "David should be able to send a Message to the group")

	// Alice, Bob, and Carol should receive the Message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from David")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from David")
		assert.Equal(t, "Hello everyone! I'm David.", string(msg.Message), "Should receive a group text Message from David")
	}

	// Chatbot1 should not receive the Message
	_, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.False(t, success, "Chatbot1 should not receive a Message from David")

	// David send a message which the chatbot should receive
	err = david.SendServerSideGroupMessage(groupId, []byte("This message is also for the chatbot."), pb.MessageType_TEXT_MESSAGE, []string{chatbot1.GetChatbotID(), chatbot2.GetChatbotID()}, false)
	assert.Nil(t, err, "David should be able to send a Message to the group")

	// Alice, Bob, and Carol should receive the Message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from David")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from David")
		assert.Equal(t, "This message is also for the chatbot.", string(msg.Message), "Should receive a group text Message from David")
	}

	// Chatbot1 should receive the Message
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a Message from David")
	assert.True(t, msgc.MessageType == pb.MessageType_TEXT_MESSAGE, "Chatbot1 should receive a group text Message from David")
	assert.Equal(t, "This message is also for the chatbot.", string(msgc.Message), "Chatbot1 should receive a group text Message from David")

	// Chatbot1's treekem root should match the group's treekem root
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot1.GetChatbotID(), chatbot1SessionDriver.GetMultiTreeKEMExternal()))
	}

	// Chatbot2 should receive the Message
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot2.GetMessageChan())
	assert.True(t, success, "Chatbot2 should receive a Message from David")
	assert.Equal(t, pb.MessageType_TEXT_MESSAGE, msgc.MessageType, "Chatbot2 should receive a group text Message from David")
	assert.Equal(t, "This message is also for the chatbot.", string(msgc.Message), "Chatbot2 should receive a group text Message from David")

	// Alice, Bob, and Carol should receive the validation message from chatbot2.
	//for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan(), david.GetMessageChan()} {
	//	msg, success = timeOutReadFromUserMessageChannel(c)
	//	assert.True(t, success, "Should receive a validation message from the group")
	//	assert.Equal(t, msg.MessageType, pb.MessageType_VALIDATION_MESSAGE, "Should receive a validation message from the group")
	//}

	// Chatbot1's treekem root should match the group's treekem root
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot2.GetChatbotID(), chatbot2SessionDriver.GetMultiTreeKEMExternal()))
	}

	// Chatbot2's treekem root should match the group's treekem root
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
		assert.True(t, multiTreeKemExternalEqual(sessionDriver.GetMultiTreeKEM(), chatbot2.GetChatbotID(), chatbot2SessionDriver.GetMultiTreeKEMExternal()))
	}

	// David issues a pseudonym for chatbot 3
	err = david.CreateAndRegisterServerSidePseudonym(groupId, chatbot3.GetChatbotID())
	assert.Nil(t, err, "David should be able to issue a pseudonym")

	// Chatbot3 should receive a pseudonym registration message from David
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot3.GetMessageChan())
	assert.True(t, success, "Chatbot3 should receive a pseudonym registration message from David")
	assert.Equal(t, msgc.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "Chatbot3 should receive a pseudonym registration message from David")

	// Alice, Bob, and Carol should also receive a pseudonym registration message from David
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a pseudonym registration message from David")
		assert.Equal(t, msg.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "Should receive a pseudonym registration message from David")
	}

	time.Sleep(500 * time.Millisecond) // TODO: fix race condition

	// David send a message to all chatbots.
	err = david.SendServerSideGroupMessage(groupId, []byte("Hello Chatbots! I'm David."), pb.MessageType_TEXT_MESSAGE, []string{chatbot1.GetChatbotID(), chatbot2.GetChatbotID(), chatbot3.GetChatbotID()}, false)

	// Alice, Bob, and Carol should receive the message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from David")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from David")
		assert.Equal(t, "Hello Chatbots! I'm David.", string(msg.Message), "Should receive a group text Message from David")
	}

	// All chatbots should receive the message.
	for _, c := range []<-chan OutputMessage{chatbot1.GetMessageChan(), chatbot2.GetMessageChan(), chatbot3.GetMessageChan()} {
		msgc, success = timeOutReadFromChatbotMessageChannel(c)
		assert.True(t, success, "Chatbots should receive a Message from David")
		assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbots should receive a group text Message from David")
		assert.Equal(t, "Hello Chatbots! I'm David.", string(msgc.Message), "Chatbots should receive a group text Message from David")
	}

	// Alice, Bob, and Carol should receive the validation message from chatbot2 and chatbot3.
	//for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan(), david.GetMessageChan()} {
	//	for i := 0; i < 2; i++ {
	//		msg, success = timeOutReadFromUserMessageChannel(c)
	//		assert.True(t, success, "Should receive a validation message from the group")
	//		assert.Equal(t, msg.MessageType, pb.MessageType_VALIDATION_MESSAGE, "Should receive a validation message from the group")
	//	}
	//}

	// Alice, Bob, Carol, and David's treekem roots should match
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
		for _, otherSessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver, davidSessionDriver} {
			assert.True(t, multiTreeKemEqual(sessionDriver.GetMultiTreeKEM(), otherSessionDriver.GetMultiTreeKEM()))
		}
	}

	// Chatbot1 send a message
	err = chatbot1.SendServerSideGroupMessage(groupId, []byte("Hello everyone! I'm Chatbot1."), pb.MessageType_TEXT_MESSAGE)
	assert.Nil(t, err, "Chatbot1 should be able to send a Message to the group")

	// Alice, Bob, Carol, and David should receive the message.
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan(), david.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Chatbot1")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Chatbot1")
		assert.Equal(t, "Hello everyone! I'm Chatbot1.", string(msg.Message), "Should receive a group text Message from Chatbot1")
	}

	// Chatbot2 send a message.
	err = chatbot2.SendServerSideGroupMessage(groupId, []byte("Hello everyone! I'm Chatbot2."), pb.MessageType_TEXT_MESSAGE)
	assert.Nil(t, err, "Chatbot2 should be able to send a Message to the group")

	// Alice, Bob, Carol, and David should receive the message.
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan(), david.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Chatbot2")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Chatbot2")
		assert.Equal(t, "Hello everyone! I'm Chatbot2.", string(msg.Message), "Should receive a group text Message from Chatbot2")
	}

	// Remove Carol from the group
	bob.RequestRemoveUserFromGroup(groupId, carol.GetUserID())

	// Alice, Bob, Carol, and David should receive GROUP_REMOVAL event
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan(), david.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a GROUP_REMOVAL event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_REMOVAL, "Should receive a GROUP_REMOVAL event")
	}

	// Chatbot 1 should receive GROUP_REMOVAL event
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a GROUP_REMOVAL event")
	assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_REMOVAL, "Chatbot1 should receive a GROUP_REMOVAL event")

	// Carol should not have the group session
	carolSessionDriver, err = carol.Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, carolSessionDriver, "Carol should not have the group session")
	assert.NotNil(t, err, "Carol should not have the group session")

	// Carol should not be in the participant list
	assert.NotContains(t, aliceSessionDriver.GetGroupParticipants(), carol.GetUserID(), "Carol should not be in the participant list")
	assert.NotContains(t, bobSessionDriver.GetGroupParticipants(), carol.GetUserID(), "Carol should not be in the participant list")
	assert.NotContains(t, davidSessionDriver.GetGroupParticipants(), carol.GetUserID(), "Carol should not be in the participant list")

	// Alice send a hide trigger message
	err = alice.SendServerSideGroupMessage(groupId, []byte("Bye Carol."), pb.MessageType_TEXT_MESSAGE, nil, true)
	assert.Nil(t, err, "Alice should be able to send a Message to the group")

	// Carol should not receive the message
	msg, success = timeOutReadFromUserMessageChannel(carol.GetMessageChan())
	assert.False(t, success, "Carol should not receive a Message from Alice")

	// Bob and David should receive the Message
	for _, c := range []<-chan user.OutputMessage{bob.GetMessageChan(), david.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Alice")
		assert.Equal(t, pb.MessageType_TEXT_MESSAGE, msg.MessageType, "Should receive a group text Message from Alice")
		assert.Equal(t, "Bye Carol.", string(msg.Message), "Should receive a group text Message from Alice")
	}

	// All chatbots should receive the message with type SKIP and the message content should not be the original one.
	for _, c := range []<-chan OutputMessage{chatbot1.GetMessageChan(), chatbot2.GetMessageChan(), chatbot3.GetMessageChan()} {
		msgc, success = timeOutReadFromChatbotMessageChannel(c)
		assert.True(t, success, "Chatbots should receive a SKIP Message from Alice")
		assert.Equal(t, pb.MessageType_SKIP, msgc.MessageType, "Chatbots should receive a SKIP Message from Alice")
		assert.NotEqual(t, "Bye Carol.", string(msgc.Message), "Chatbots should receive a SKIP Message from Alice")
	}

	// All member should receive the validation message from chatbot 2 and chatbot 3.
	//for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), david.GetMessageChan()} {
	//	for i := 0; i < 2; i++ {
	//		msg, success = timeOutReadFromUserMessageChannel(c)
	//		assert.True(t, success, "Should receive a validation message from the group")
	//		assert.Equal(t, msg.MessageType, pb.MessageType_VALIDATION_MESSAGE, "Should receive a validation message from the group")
	//	}
	//}

	// Bob send a hide trigger message to chatbot 2
	err = bob.SendServerSideGroupMessage(groupId, []byte("Only for Chatbot2."), pb.MessageType_TEXT_MESSAGE, []string{chatbot2.GetChatbotID()}, true)
	assert.Nil(t, err, "Bob should be able to send a Message to the group")

	// Alice and David should receive the Message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), david.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Bob")
		assert.Equal(t, pb.MessageType_TEXT_MESSAGE, msg.MessageType, "Should receive a group text Message from Bob")
		assert.Equal(t, "Only for Chatbot2.", string(msg.Message), "Should receive a group text Message from Bob")
	}

	// Chatbot 2 should receive the original message.
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot2.GetMessageChan())
	assert.True(t, success, "Chatbot2 should receive a Message from Bob")
	assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot2 should receive a group text Message from Bob")
	assert.Equal(t, "Only for Chatbot2.", string(msgc.Message), "Chatbot2 should receive a group text Message from Bob")

	// Chatbot 1 and Chatbot 3 should receive the message with type SKIP and the message content should not be the original one.
	for _, c := range []<-chan OutputMessage{chatbot1.GetMessageChan(), chatbot3.GetMessageChan()} {
		msgc, success = timeOutReadFromChatbotMessageChannel(c)
		assert.True(t, success, "Chatbots should receive a Message from Bob")
		assert.Equal(t, msgc.MessageType, pb.MessageType_SKIP, "Chatbots should receive a SKIP Message from Bob")
		assert.NotEqual(t, "Only for Chatbot2.", string(msgc.Message), "Chatbots should receive a SKIP Message from Bob")
	}

	// All members should receive the validation message from chatbot 2 and chatbot 3.
	//for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), david.GetMessageChan()} {
	//	for i := 0; i < 2; i++ {
	//		msg, success = timeOutReadFromUserMessageChannel(c)
	//		assert.True(t, success, "Should receive a validation message from the group")
	//		assert.Equal(t, pb.MessageType_VALIDATION_MESSAGE, msg.MessageType, "Should receive a validation message from the group")
	//	}
	//}

	// All chatbots should match all members Multi-TreeKEM state.
	for _, sessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, davidSessionDriver} {
		for _, otherSessionDriver := range []*client.ServerSideGroupSessionDriver{aliceSessionDriver, bobSessionDriver, davidSessionDriver} {
			assert.True(t, multiTreeKemEqual(sessionDriver.GetMultiTreeKEM(), otherSessionDriver.GetMultiTreeKEM()), "All users should have the same Multi-TreeKEM state")
		}
	}

	// David send a hide trigger message to chatbot 3
	err = david.SendServerSideGroupMessage(groupId, []byte("Only for Chatbot3."), pb.MessageType_TEXT_MESSAGE, []string{chatbot3.GetChatbotID()}, true)
	assert.Nil(t, err, "David should be able to send a Message to the group")

	// Alice and Bob should receive the Message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from David")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from David")
		assert.Equal(t, "Only for Chatbot3.", string(msg.Message), "Should receive a group text Message from David")
	}

	// Chatbot 3 should receive the original message.
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot3.GetMessageChan())
	assert.True(t, success, "Chatbot3 should receive a Message from David")
	assert.Equal(t, pb.MessageType_TEXT_MESSAGE, msgc.MessageType, "Chatbot3 should receive a group text Message from David")
	assert.Equal(t, "Only for Chatbot3.", string(msgc.Message), "Chatbot3 should receive a group text Message from David")

	// Chatbot 1 and Chatbot 2 should receive the message with type SKIP and the message content should not be the original one.
	for _, c := range []<-chan OutputMessage{chatbot1.GetMessageChan(), chatbot2.GetMessageChan()} {
		msgc, success = timeOutReadFromChatbotMessageChannel(c)
		assert.True(t, success, "Chatbots should receive a Message from David")
		assert.Equal(t, msgc.MessageType, pb.MessageType_SKIP, "Chatbots should receive a SKIP Message from David")
		assert.NotEqual(t, "Only for Chatbot3.", string(msgc.Message), "Chatbots should receive a SKIP Message from David")
	}
}

// TestMlsGroupMessage test the MLS group Message.
func TestMlsGroupMessage(t *testing.T) {
	setup()

	// Alice is the initiator of the group
	groupId, err := alice.CreateGroup(pb.GroupType_MLS)
	assert.Nil(t, err, "Alice should be able to create a group")

	// Alice should have the group session
	aliceSessionDriver, err := alice.Client.GetMlsGroupSessionDriver(groupId)
	assert.Nil(t, err, "Alice should have the group session")

	assert.True(t, aliceSessionDriver.GetGroupState().Equals(**aliceSessionDriver.GetMlsMultiTree().MlsState), "Alice and Alice's MlsMultiTree should have the same group state")

	// Invite Bob to the group.
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_MLS, bob.GetUserID())
	msg, success := timeOutReadFromUserMessageChannel(bob.GetMessageChan())
	assert.True(t, success, "Bob should receive a group invitation from Alice")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "Bob should receive a group invitation from Alice")

	// Bob should have the group session
	bobSessionDriver, err := bob.Client.GetMlsGroupSessionDriver(groupId)
	assert.Nil(t, err, "Bob should have the group session")

	// Alice should receive a GROUP_ADDITION event.
	msg, success = timeOutReadFromUserMessageChannel(alice.GetMessageChan())
	assert.True(t, success, "Alice should receive a group addition event")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Alice should receive a group addition event")

	// Alice and Bob should have the same state
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*bobSessionDriver.GetGroupState()), "Alice and Bob should have the same group state")

	// Alice send a message to the group
	err = alice.SendMlsGroupMessage(groupId, []byte("Hello everyone! I'm Alice."), pb.MessageType_TEXT_MESSAGE, nil, false)
	assert.Nil(t, err, "Alice should be able to send a message to the group")

	// Bob should receive the message
	msg, success = timeOutReadFromUserMessageChannel(bob.GetMessageChan())
	assert.True(t, success, "Bob should receive a Message from Alice")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Bob should receive a group text message from Alice")
	assert.Equal(t, "Hello everyone! I'm Alice.", string(msg.Message), "Bob should receive a group text message from Alice")

	// Alice and Bob should have the same state
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*bobSessionDriver.GetGroupState()), "Alice and Bob should have the same group state")

	// Bob send a message to the group
	err = bob.SendMlsGroupMessage(groupId, []byte("Hello everyone! I'm Bob."), pb.MessageType_TEXT_MESSAGE, nil, false)
	assert.Nil(t, err, "Bob should be able to send a message to the group")

	// Alice should receive the message
	msg, success = timeOutReadFromUserMessageChannel(alice.GetMessageChan())
	assert.True(t, success, "Alice should receive a message from Bob")
	assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Alice should receive a group text message from Bob")
	assert.Equal(t, "Hello everyone! I'm Bob.", string(msg.Message), "Alice should receive a group text message from Bob")

	// Alice and Bob should have the same state
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*bobSessionDriver.GetGroupState()), "Alice and Bob should have the same group state")

	assert.True(t, aliceSessionDriver.GetGroupState().Equals(**aliceSessionDriver.GetMlsMultiTree().MlsState), "Alice and Alice's MlsMultiTree should have the same group state")

	// Invite Carol to the group.
	alice.RequestInviteUserToGroup(groupId, pb.GroupType_MLS, carol.GetUserID())
	msg, success = timeOutReadFromUserMessageChannel(carol.GetMessageChan())
	assert.True(t, success, "Carol should receive a group invitation from Alice")
	assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "Carol should receive a group invitation from Alice")

	// Carol should have the group session
	carolSessionDriver, err := carol.Client.GetMlsGroupSessionDriver(groupId)
	assert.Nil(t, err, "Carol should have the group session")

	// Alice and Bob should receive a GROUP_ADDITION event.
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Alice and Bob should receive a group addition event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "Alice and Bob should receive a group addition event")
	}

	// Alice, Bob, and Carol should have the same state
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*bobSessionDriver.GetGroupState()), "Alice and Bob should have the same group state")
	assert.True(t, aliceSessionDriver.GetGroupState().Equals(*carolSessionDriver.GetGroupState()), "Alice and Carol should have the same group state")
	assert.True(t, bobSessionDriver.GetGroupState().Equals(*carolSessionDriver.GetGroupState()), "Bob and Carol should have the same group state")

	assert.True(t, aliceSessionDriver.GetGroupState().Equals(**aliceSessionDriver.GetMlsMultiTree().MlsState), "Alice and Alice's MlsMultiTree should have the same group state")

	// Alice, Bob, and Carol should have the same MLS tree root
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.Equal(t, aliceSessionDriver.GetGroupState().Tree.RootHash(), sessionDriver.GetGroupState().Tree.RootHash(), "Alice, Bob, and Carol should have the same MLS tree root")

		// Compare current root
		aliceTreeRoot, err := aliceSessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Alice should have the tree root")
		otherTreeRoot, err := sessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Other should have the tree root")
		assert.Equal(t, aliceTreeRoot.Secret, otherTreeRoot.Secret, "Alice, Bob, and Carol should have the same MLS tree root")
	}

	// Alice invite chatbot1 to the group
	alice.RequestInviteChatbotToGroup(groupId, pb.GroupType_MLS, chatbot1.GetChatbotID(), false, false)

	// Alice, Bob, and Carol should receive GROUP_CHATBOT_ADDITION event
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a GROUP_CHATBOT_ADDITION event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "Should receive a GROUP_CHATBOT_ADDITION event")
	}

	// Chatbot1 should receive GROUP_CHATBOT_INVIATION event
	msgc, success := timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a GROUP_CHATBOT_INVIATION event")
	assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_CHATBOT_INVITATION, "Chatbot1 should receive a GROUP_CHATBOT_INVIATION event")

	// Chatbot1 should be in the group
	chatbot1SessionDriver, err := chatbot1.Client.GetMlsGroupSessionDriver(groupId)
	assert.Nil(t, err, "Chatbot1 should have the group session")
	assert.Contains(t, aliceSessionDriver.GetGroupChatbots(), chatbot1.GetChatbotID(), "Chatbot1 should be in the chatbot list")

	// Chatbot1's state should match Alice, Bob, and Carol's state
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, chatbot1SessionDriver.GetGroupState().Equals(*sessionDriver.GetGroupState()), "Chatbot1's state should match Alice, Bob, and Carol's state")
	}

	// Carol send a message
	err = carol.SendMlsGroupMessage(groupId, []byte("Hello everyone! I'm Carol."), pb.MessageType_TEXT_MESSAGE, []string{chatbot1.GetChatbotID()}, false)
	assert.Nil(t, err, "Carol should be able to send a Message to the group")

	// Alice and Bob should receive the Message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Carol")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Carol")
		assert.Equal(t, "Hello everyone! I'm Carol.", string(msg.Message), "Should receive a group text Message from Carol")
	}

	// Chatbot1 should receive the Message
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a Message from Alice")
	assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot1 should receive a group text Message from Alice")
	assert.Equal(t, "Hello everyone! I'm Carol.", string(msgc.Message), "Chatbot1 should receive a group text Message from Alice")

	// Chatbot1's state should match Alice, Bob, and Carol's state
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, chatbot1SessionDriver.GetGroupState().Equals(*sessionDriver.GetGroupState()), "Chatbot1's state should match Alice, Bob, and Carol's state")
	}

	// Alice, Bob, and Carol should have the same MLS tree root
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.Equal(t, aliceSessionDriver.GetGroupState().Tree.RootHash(), sessionDriver.GetGroupState().Tree.RootHash(), "Alice, Bob, and Carol should have the same MLS tree root")

		// Compare current root
		aliceTreeRoot, err := aliceSessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Alice should have the tree root")
		otherTreeRoot, err := sessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Other should have the tree root")
		assert.Equal(t, aliceTreeRoot.Secret, otherTreeRoot.Secret, "Alice, Bob, and Carol should have the same MLS tree root")
	}

	// Add chatbot 4 to the group.
	alice.RequestInviteChatbotToGroup(groupId, pb.GroupType_MLS, chatbot4.GetChatbotID(), false, false)

	// Alice, Bob, and Carol should receive GROUP_CHATBOT_ADDITION event
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a GROUP_CHATBOT_ADDITION event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "Should receive a GROUP_CHATBOT_ADDITION event")
	}

	// Chatbot4 should receive GROUP_CHATBOT_INVIATION event
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot4.GetMessageChan())
	assert.True(t, success, "Chatbot4 should receive a GROUP_CHATBOT_INVIATION event")

	// Chatbot4 should be in the group
	chatbot4SessionDriver, err := chatbot4.Client.GetMlsGroupSessionDriver(groupId)
	assert.Nil(t, err, "Chatbot4 should have the group session")

	// Alice, Bob, and Carol should have the same MLS tree root
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.Equal(t, aliceSessionDriver.GetGroupState().Tree.RootHash(), sessionDriver.GetGroupState().Tree.RootHash(), "Alice, Bob, and Carol should have the same MLS tree root")

		// Compare current root
		aliceTreeRoot, err := aliceSessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Alice should have the tree root")
		otherTreeRoot, err := sessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Other should have the tree root")
		assert.Equal(t, aliceTreeRoot.Secret, otherTreeRoot.Secret, "Alice, Bob, and Carol should have the same MLS tree root")
	}

	// Chatbots should have the same MLS tree root
	assert.Equal(t, aliceSessionDriver.GetGroupState().Tree.RootHash(), chatbot1SessionDriver.GetGroupState().Tree.RootHash(), "Alice and Chatbot1 should have the same MLS tree root")
	assert.Equal(t, aliceSessionDriver.GetGroupState().Tree.RootHash(), chatbot4SessionDriver.GetGroupState().Tree.RootHash(), "Alice and Chatbot4 should have the same MLS tree root")

	// Add Chatbot 2 to the group and set as IGA bot.
	alice.RequestInviteChatbotToGroup(groupId, pb.GroupType_MLS, chatbot2.GetChatbotID(), true, false)

	// Alice, Bob, and Carol should have the same MLS tree root
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.Equal(t, aliceSessionDriver.GetGroupState().Tree.RootHash(), sessionDriver.GetGroupState().Tree.RootHash(), "Alice, Bob, and Carol should have the same MLS tree root")

		// Compare current root
		aliceTreeRoot, err := aliceSessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Alice should have the tree root")
		otherTreeRoot, err := sessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Bob should have the tree root")
		assert.Equal(t, aliceTreeRoot, otherTreeRoot, "Alice, Bob, and Carol should have the same MLS tree root")
	}

	// Alice, Bob, and Carol should receive GROUP_CHATBOT_ADDITION event
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a GROUP_CHATBOT_ADDITION event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "Should receive a GROUP_CHATBOT_ADDITION event")
	}

	// Chatbot2 should receive GROUP_CHATBOT_INVIATION event
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot2.GetMessageChan())
	assert.True(t, success, "Chatbot2 should receive a GROUP_CHATBOT_INVIATION event")
	assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_CHATBOT_INVITATION, "Chatbot2 should receive a GROUP_CHATBOT_INVIATION event")

	// Chatbot2 should be in the group
	chatbot2SessionDriver, err := chatbot2.Client.GetMlsGroupSessionDriver(groupId)
	assert.Nil(t, err, "Chatbot2 should have the group session")
	assert.Contains(t, aliceSessionDriver.GetGroupChatbots(), chatbot2.GetChatbotID(), "Chatbot2 should be in the chatbot list")

	// Alice, Bob, and Carol should have the same MLS tree root
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.Equal(t, aliceSessionDriver.GetGroupState().Tree.RootHash(), sessionDriver.GetGroupState().Tree.RootHash(), "Alice, Bob, and Carol should have the same MLS tree root")

		// Compare current root
		aliceTreeRoot, err := aliceSessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Alice should have the tree root")
		otherTreeRoot, err := sessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Bob should have the tree root")
		assert.Equal(t, aliceTreeRoot, otherTreeRoot, "Alice, Bob, and Carol should have the same MLS tree root")
	}

	// Alice, Bob, and Carol should have the same MlsMultiTree state.
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.Equal(t, sessionDriver.GetMlsMultiTree().GetRootPublic(chatbot2.chatbotID), chatbot2SessionDriver.GetMlsMultiTreeExternal().GetRootPublic(), "Alice, Bob, and Carol should have the same MlsMultiTree state")
		assert.Equal(t, sessionDriver.GetMlsMultiTree().GetRootSignPublic(chatbot2.chatbotID), chatbot2SessionDriver.GetMlsMultiTreeExternal().GetRootSignPublic(), "Alice, Bob, and Carol should have the same MlsMultiTree state")
		assert.Equal(t, sessionDriver.GetMlsMultiTree().GetRootSecret(chatbot2.chatbotID), chatbot2SessionDriver.GetMlsMultiTreeExternal().GetRootSecret(), "Alice, Bob, and Carol should have the same MlsMultiTree state")

		assert.True(t, sessionDriver.GetGroupState().Equals(*chatbot1SessionDriver.GetGroupState()), "Chatbot1 should match Alice, Bob, and Carol's MLS state")
	}

	// Chatbot2 tries to send a message to the group
	err = chatbot2.SendMlsGroupMessage(groupId, []byte("Hello everyone! I'm Chatbot2."), pb.MessageType_TEXT_MESSAGE)
	assert.Nil(t, err, "Chatbot2 should be able to send a Message to the group")

	// Alice, Bob, and Carol should receive the Message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Chatbot2")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Chatbot2")
		assert.Equal(t, "Hello everyone! I'm Chatbot2.", string(msg.Message), "Should receive a group text Message from Chatbot2")
	}

	// Alice, Bob, and Carol should have the same MLS tree root
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.Equal(t, aliceSessionDriver.GetGroupState().Tree.RootHash(), sessionDriver.GetGroupState().Tree.RootHash(), "Alice, Bob, and Carol should have the same MLS tree root")

		// Compare current root
		aliceTreeRoot, err := aliceSessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Alice should have the tree root")
		otherTreeRoot, err := sessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Bob should have the tree root")
		assert.Equal(t, aliceTreeRoot, otherTreeRoot, "Alice, Bob, and Carol should have the same MLS tree root")
	}

	// Chatbots should match Alice, Bob, and Carol's MlsMultiTree state
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.True(t, sessionDriver.GetGroupState().Equals(*chatbot1SessionDriver.GetGroupState()), "Chatbot1 should match Alice, Bob, and Carol's MLS state")
		assert.Equal(t, chatbot2SessionDriver.GetMlsMultiTreeExternal().GetRootSecret(), sessionDriver.GetMlsMultiTree().GetRootSecret(chatbot2.GetChatbotID()), "Chatbot2 should match Alice, Bob, and Carol's MlsMultiTree state")
	}

	// Bob tries to send a message to both chatbot
	err = bob.SendMlsGroupMessage(groupId, []byte("This is an IGA message. Chatbot 2 should receive it without learning my identity"), pb.MessageType_TEXT_MESSAGE, []string{chatbot1.GetChatbotID(), chatbot2.GetChatbotID(), chatbot4.GetChatbotID()}, false)
	assert.Nil(t, err, "Bob should be able to send an IGA message to chatbots")

	// Alice and Carol should receive the message.
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Bob")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Bob")
		assert.Equal(t, "This is an IGA message. Chatbot 2 should receive it without learning my identity", string(msg.Message), "Should receive a group text Message from Bob")
	}

	// Chatbot1 should receive the message
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot1.GetMessageChan())
	assert.True(t, success, "Chatbot1 should receive a Message from Bob")
	assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot2 should receive a group text Message from Bob")
	assert.Equal(t, "This is an IGA message. Chatbot 2 should receive it without learning my identity", string(msgc.Message), "Chatbot1 should receive a group text Message from Bob")

	// Chatbot2 should receive the message
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot2.GetMessageChan())
	assert.True(t, success, "Chatbot2 should receive a Message from Bob")
	assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot2 should receive a group text Message from Bob")
	assert.Equal(t, "This is an IGA message. Chatbot 2 should receive it without learning my identity", string(msgc.Message), "Chatbot2 should receive a group text Message from Bob")

	// Chatbot4 should receive the message
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot4.GetMessageChan())
	assert.True(t, success, "Chatbot4 should receive a Message from Bob")
	assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot4 should receive a group text Message from Bob")
	assert.Equal(t, "This is an IGA message. Chatbot 2 should receive it without learning my identity", string(msgc.Message), "Chatbot4 should receive a group text Message from Bob")

	// Alice, Bob, and Carol should receive the validation message.
	//for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
	//	msg, success = timeOutReadFromUserMessageChannel(c)
	//	assert.True(t, success, "Should receive a validation message from the group")
	//	assert.Equal(t, msg.MessageType, pb.MessageType_VALIDATION_MESSAGE, "Should receive a validation message from the group")
	//}

	// Bob invites Chatbot3 with pseudonymity
	bob.RequestInviteChatbotToGroup(groupId, pb.GroupType_MLS, chatbot3.GetChatbotID(), true, true)

	// Alice, Bob, and Carol should receive GROUP_CHATBOT_ADDITION event
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a GROUP_CHATBOT_ADDITION event")
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "Should receive a GROUP_CHATBOT_ADDITION event")
	}

	// Chatbot3 should receive GROUP_CHATBOT_INVIATION event
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot3.GetMessageChan())
	assert.True(t, success, "Chatbot3 should receive a GROUP_CHATBOT_INVIATION event")
	assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_CHATBOT_INVITATION, "Chatbot3 should receive a GROUP_CHATBOT_INVIATION event")

	// Chatbot3 should be in the group
	chatbot3SessionDriver, err := chatbot3.Client.GetMlsGroupSessionDriver(groupId)
	assert.Nil(t, err, "Chatbot3 should have the group session")

	// Alice issue a pseudonym
	err = alice.CreateAndRegisterMlsPseudonym(groupId, chatbot3.GetChatbotID())
	assert.Nil(t, err, "Alice should be able to issue a pseudonym")

	// Chatbot3 should receive a pseudonym registration message from Alice
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot3.GetMessageChan())
	assert.True(t, success, "Chatbot3 should receive a pseudonym registration message from Alice")
	assert.Equal(t, msgc.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "Chatbot3 should receive a pseudonym registration message from Alice")

	//// Bob and Carol should also receive a pseudonym registration message from Alice
	//for _, c := range []<-chan user.OutputMessage{bob.GetMessageChan(), carol.GetMessageChan()} {
	//	msg, success = timeOutReadFromUserMessageChannel(c)
	//	assert.True(t, success, "Should receive a pseudonym registration message from Alice")
	//	assert.Equal(t, msg.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "Should receive a pseudonym registration message from Alice")
	//}

	// Alice, Bob, and Carol should have the same MLS tree root
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.Equal(t, aliceSessionDriver.GetGroupState().Tree.RootHash(), sessionDriver.GetGroupState().Tree.RootHash(), "Alice, Bob, and Carol should have the same MLS tree root")

		// Compare current root
		aliceTreeRoot, err := aliceSessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Alice should have the tree root")
		otherTreeRoot, err := sessionDriver.GetMlsMultiTree().GetTreeKEMRoot()
		assert.Nil(t, err, "Bob should have the tree root")
		assert.Equal(t, aliceTreeRoot, otherTreeRoot, "Alice, Bob, and Carol should have the same MLS tree root")
	}

	// Alice, Bob, and Carol should have the same MlsMultiTree state.
	for _, sessionDriver := range []*client.MlsGroupSessionDriver{aliceSessionDriver, bobSessionDriver, carolSessionDriver} {
		assert.Equal(t, sessionDriver.GetMlsMultiTree().GetRootPublic(chatbot2.chatbotID), chatbot2SessionDriver.GetMlsMultiTreeExternal().GetRootPublic(), "Alice, Bob, and Carol should have the same MlsMultiTree state as chatbot 2")
		assert.Equal(t, sessionDriver.GetMlsMultiTree().GetRootSignPublic(chatbot2.chatbotID), chatbot2SessionDriver.GetMlsMultiTreeExternal().GetRootSignPublic(), "Alice, Bob, and Carol should have the same MlsMultiTree state as chatbot 2")
		assert.Equal(t, sessionDriver.GetMlsMultiTree().GetRootSecret(chatbot2.chatbotID), chatbot2SessionDriver.GetMlsMultiTreeExternal().GetRootSecret(), "Alice, Bob, and Carol should have the same MlsMultiTree state as chatbot 2")

		assert.Equal(t, sessionDriver.GetMlsMultiTree().GetRootPublic(chatbot3.chatbotID), chatbot3SessionDriver.GetMlsMultiTreeExternal().GetRootPublic(), "Alice, Bob, and Carol should have the same MlsMultiTree state as chatbot 3")
		assert.Equal(t, sessionDriver.GetMlsMultiTree().GetRootSignPublic(chatbot3.chatbotID), chatbot3SessionDriver.GetMlsMultiTreeExternal().GetRootSignPublic(), "Alice, Bob, and Carol should have the same MlsMultiTree state as chatbot 3")
		assert.Equal(t, sessionDriver.GetMlsMultiTree().GetRootSecret(chatbot3.chatbotID), chatbot3SessionDriver.GetMlsMultiTreeExternal().GetRootSecret(), "Alice, Bob, and Carol should have the same MlsMultiTree state as chatbot 3")

		assert.True(t, sessionDriver.GetGroupState().Equals(*chatbot1SessionDriver.GetGroupState()), "Chatbot1 should match Alice, Bob, and Carol's MLS state")
		assert.True(t, sessionDriver.GetGroupState().Equals(*chatbot4SessionDriver.GetGroupState()), "Chatbot4 should match Alice, Bob, and Carol's MLS state")
	}

	// Alice send a message to the group
	err = alice.SendMlsGroupMessage(groupId, []byte("Hello everyone! Chatbot 3 would only see my pseudonym."), pb.MessageType_TEXT_MESSAGE, []string{chatbot1.GetChatbotID(), chatbot2.GetChatbotID(), chatbot3.GetChatbotID(), chatbot4.GetChatbotID()}, false)
	assert.Nil(t, err, "Alice should be able to send a message to the group")

	// Bob and Carol should receive the Message
	for _, c := range []<-chan user.OutputMessage{bob.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Alice")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Alice")
		assert.Equal(t, "Hello everyone! Chatbot 3 would only see my pseudonym.", string(msg.Message), "Should receive a group text Message from Alice")
	}

	// Chatbots should receive the Message
	for _, c := range []<-chan OutputMessage{chatbot1.GetMessageChan(), chatbot2.GetMessageChan(), chatbot3.GetMessageChan(), chatbot4.GetMessageChan()} {
		msgc, success = timeOutReadFromChatbotMessageChannel(c)
		assert.True(t, success, "Chatbots should receive a Message from Alice")
		assert.Equal(t, msgc.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbots should receive a group text Message from Alice")
		assert.Equal(t, "Hello everyone! Chatbot 3 would only see my pseudonym.", string(msgc.Message), "Chatbots should receive a group text Message from Alice")
	}

	// Should receive validation messages from the chatbot 2 and 3
	//for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
	//	for i := 0; i < 2; i++ {
	//		msg, success = timeOutReadFromUserMessageChannel(c)
	//		assert.True(t, success, "Should receive a validation message from the group")
	//		assert.Equal(t, msg.MessageType, pb.MessageType_VALIDATION_MESSAGE, "Should receive a validation message from the group")
	//	}
	//}

	// Bob register a pseudonym
	err = bob.CreateAndRegisterMlsPseudonym(groupId, chatbot3.GetChatbotID())
	assert.Nil(t, err, "Bob should be able to issue a pseudonym")

	// Chatbot3 should receive a pseudonym registration message from Bob
	msgc, success = timeOutReadFromChatbotMessageChannel(chatbot3.GetMessageChan())
	assert.True(t, success, "Chatbot3 should receive a pseudonym registration message from Bob")
	assert.Equal(t, msgc.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "Chatbot3 should receive a pseudonym registration message from Bob")

	// Bob sends a message with hiding triggers
	err = bob.SendMlsGroupMessage(groupId, []byte("Only for chatbot1 and chatbot4."), pb.MessageType_TEXT_MESSAGE, []string{chatbot1.GetChatbotID(), chatbot4.GetChatbotID()}, true)
	assert.Nil(t, err, "Bob should be able to send a Message to the group")

	// Alice and Carol should receive the Message
	for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), carol.GetMessageChan()} {
		msg, success = timeOutReadFromUserMessageChannel(c)
		assert.True(t, success, "Should receive a Message from Bob")
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Should receive a group text Message from Bob")
		assert.Equal(t, "Only for chatbot1 and chatbot4.", string(msg.Message), "Should receive a group text Message from Bob")
	}

	// Chatbot1 and chatbot 4 should receive the original message.
	for _, c := range []<-chan OutputMessage{chatbot1.GetMessageChan(), chatbot4.GetMessageChan()} {
		msgc, success = timeOutReadFromChatbotMessageChannel(c)
		assert.True(t, success, "Chatbots should receive a Message from Bob")
		assert.Equal(t, pb.MessageType_TEXT_MESSAGE, msgc.MessageType, "Chatbots should receive a group text Message from Bob")
		assert.Equal(t, "Only for chatbot1 and chatbot4.", string(msgc.Message), "Chatbots should receive a group text Message from Bob")
	}

	// Chatbot2 and chatbot3 should receive the message with type SKIP and the message content should not be the original one.
	for _, c := range []<-chan OutputMessage{chatbot2.GetMessageChan(), chatbot3.GetMessageChan()} {
		msgc, success = timeOutReadFromChatbotMessageChannel(c)
		assert.True(t, success, "Chatbots should receive a Message from Bob")
		assert.Equal(t, pb.MessageType_SKIP, msgc.MessageType, "Chatbots should receive a SKIP Message from Bob")
		assert.NotEqual(t, "Only for chatbot1 and chatbot4.", string(msgc.Message), "Chatbots should receive a SKIP Message from Bob")
	}

	// Should receive validation messages from the chatbot 1 and 4
	//for _, c := range []<-chan user.OutputMessage{alice.GetMessageChan(), bob.GetMessageChan(), carol.GetMessageChan()} {
	//	for i := 0; i < 2; i++ {
	//		msg, success = timeOutReadFromUserMessageChannel(c)
	//		assert.True(t, success, "Should receive a validation message from the group")
	//		assert.Equal(t, pb.MessageType_VALIDATION_MESSAGE, msg.MessageType, "Should receive a validation message from the group")
	//	}
	//}

}

func TestSendMessageToNonExistentUser(t *testing.T) {
	// TODO
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
func createClientSideUserWithRandomUserID(prefix string) *user.ClientSideUser {
	userID := prefix + randomString(8)
	//user, _ := user.NewClientSideUser(userID, "localhost:50051", true)
	return user.NewClientSideUserBufconn(userID, dialer(), true)
}

// createClientSideChatbotWithRandomUserID create a client side chatbot with a random user ID.
func createClientSideChatbotWithRandomUserID(prefix string) *ClientSideChatbot {
	chatbotID := prefix + "-" + randomString(8)
	//chatbot, _ := NewClientSideChatbot(chatbotID, "localhost:50051", true)
	chatbot := NewClientSideChatbotBufconn(chatbotID, dialer(), true)
	return chatbot
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

// timeOutReadFromUserMessageChannel read from the given channel and return false if it times out.
func timeOutReadFromChatbotMessageChannel(ch <-chan OutputMessage) (OutputMessage, bool) {
	for {
		select {
		case msg := <-ch:
			return msg, true
		case <-time.After(1 * time.Second):
			return OutputMessage{}, false
		}
	}
}

// timeOutReadFromUserMessageChannel read from the given channel and return false if it times out.
func timeOutReadFromUserMessageChannel(ch <-chan user.OutputMessage) (user.OutputMessage, bool) {
	for {
		select {
		case msg := <-ch:
			return msg, true
		case <-time.After(1 * time.Second):
			return user.OutputMessage{}, false
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
		if !bytes.Equal(root.Public, mt2.GetRootPublic(key)) {
			return false
		}
	}

	return true
}

func multiTreeKemExternalEqual(mt *treekem.MultiTreeKEM, mtRootID string, mte *treekem.MultiTreeKEMExternal) bool {
	return bytes.Equal(mte.GetRootPublic(), mt.GetRootPublic(mtRootID))
}
