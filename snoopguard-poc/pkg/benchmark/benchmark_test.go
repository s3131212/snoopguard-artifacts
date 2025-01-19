package benchmark

import (
	"chatbot-poc-go/pkg/chatbot"
	pb "chatbot-poc-go/pkg/protos/services"
	"chatbot-poc-go/pkg/server"
	"chatbot-poc-go/pkg/user"
	"chatbot-poc-go/pkg/util"
	"context"
	"fmt"
	"github.com/loov/hrtime"
	"github.com/stretchr/testify/assert"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/libsignal/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"
)

var users map[int]*user.ClientSideUser
var chatbots map[int]*chatbot.ClientSideChatbot
var numberOfExperiments = 1000

const bufSize = 1024 * 1024

var listener *bufconn.Listener

func createUsersAndChatbots(t *testing.T, userSize, chatbotSize int) {
	emptyLogger := util.EmptyLogger{}
	logger.Logger = &emptyLogger
	log.SetOutput(io.Discard)

	users = make(map[int]*user.ClientSideUser)
	chatbots = make(map[int]*chatbot.ClientSideChatbot)

	for i := 0; i < userSize; i++ {
		users[i] = createClientSideUserWithRandomUserID(fmt.Sprintf("user%v", i))
	}

	for i := 0; i < chatbotSize; i++ {
		chatbots[i] = createClientSideChatbotWithRandomUserID(fmt.Sprintf("chatbot%v", i))
	}

	fmt.Println("Create Users and Chatbots: ", userSize, " users, ", chatbotSize, " chatbots finished")
}

func ensureUsersAndChatbots(t *testing.T, userSize, chatbotSize int) {
	if len(users) < userSize && len(chatbots) < chatbotSize {
		t.Error("Not enough users and chatbots. Please create them first.")
	}
}

func deactivateUserById(t *testing.T, id int) {
	users[id].Deactivate()
	msg, success := timeOutReadFromUserMessageChannel(users[id].GetMessageChan())
	assert.True(t, success, "User %v should receive a deactivate message", id)
	assert.Equal(t, msg.Message, []byte("Deactivate"), "User %v should receive a deactivate message", id)
}

func deactivateChatbotById(t *testing.T, id int) {
	chatbots[id].Deactivate()
	// Don't wait for the deactivate message from the chatbot because we don't want to mess up the benchmark.
}

func benchmarkSendIndividualMessage(t *testing.T, userSize int) {
	ensureUsersAndChatbots(t, userSize, 0)
	bench := hrtime.NewBenchmark(numberOfExperiments)
	for i := 0; bench.Next(); i++ {
		// User i % userSize sends a message to User (i+1) % userSize
		err := users[i%userSize].SendIndividualMessage(protocol.NewSignalAddress(users[(i+1)%userSize].GetUserID(), 1), []byte(fmt.Sprintf("Hello User%v! I'm User%v.", (i+1)%userSize, i%userSize)), pb.MessageType_TEXT_MESSAGE)
		assert.Nil(t, err, "User%v should be able to send a Message to User%v", i%userSize, (i+1)%userSize)
		msg, success := timeOutReadFromUserMessageChannel(users[(i+1)%userSize].GetMessageChan())
		assert.True(t, success, "User%v should receive a Message from User%v", (i+1)%userSize, i%userSize)
		assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "User%v should receive a text Message from User%v", (i+1)%userSize, i%userSize)
		assert.Equal(t, fmt.Sprintf("Hello User%v! I'm User%v.", (i+1)%userSize, i%userSize), string(msg.Message), "User%v should receive a text Message from User%v", (i+1)%userSize, i%userSize)
	}
	fmt.Println("========== Send Individual Message: ", userSize, " users ==========")
	fmt.Println(bench.Histogram(10))
}

func createServerSideGroupOfSize(t *testing.T, memberSize int, chatbotSize int, isIGA bool, isPseudo bool) string {
	ensureUsersAndChatbots(t, memberSize, chatbotSize)

	// User 0 is the initiator of the group
	groupId, err := users[0].CreateGroup(pb.GroupType_SERVER_SIDE)
	assert.Nil(t, err, "User 0 should be able to create a group")

	for i := 1; i < memberSize; i++ {
		// Invite user i to the group
		users[0].RequestInviteUserToGroup(groupId, pb.GroupType_SERVER_SIDE, users[i].GetUserID())

		// User i should receive a group invitation
		msg, success := timeOutReadFromUserMessageChannel(users[i].GetMessageChan())
		assert.True(t, success, "User %v should receive a group invitation from User 0", i)
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "User %v should receive a group invitation from User 0", i)

		// User 0 ~ i-1 should receive a GROUP_ADDITION event.
		for j := 0; j < i; j++ {
			msg, success = timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
			assert.True(t, success, "User %v should receive a group addition event", j)
			assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "User %v should receive a group addition event", j)
		}

		// User i distributes its sender key.
		err = users[i].DistributeSelfSenderKeyToAll(groupId)
		assert.Nil(t, err, "User %v should be able to distribute its sender key", i)

		// User 0 ~ i-1 should receive sender key distribution messages from User i.
		for j := 0; j < i; j++ {
			msg, success = timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
			assert.True(t, success, "User %v should receive a sender key distribution message from User %v", j, i)
			assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "User %v should receive a sender key distribution message from User %v", j, i)
		}

		// User i should receive sender key distribution messages from all other members.
		for j := 0; j < i; j++ {
			msg, success = timeOutReadFromUserMessageChannel(users[i].GetMessageChan())
			assert.True(t, success, "User %v should receive a sender key distribution message from User %v", i, j)
			assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "User %v should receive a sender key distribution message from User %v", i, j)
		}

		// Each member should have each other's receiving sessions.
		for j := 0; j < i; j++ {
			// Get session drivers
			sessionDriver1, err := users[i].Client.GetServerSideGroupSessionDriver(groupId)
			assert.Nil(t, err, "User %v should have the group session", i)
			sessionDriver2, err := users[j].Client.GetServerSideGroupSessionDriver(groupId)
			assert.Nil(t, err, "User %v should have the group session", j)

			assert.True(t, sessionDriver2.HasUserReceivingSession(users[i].GetUserID()), "User %v should have User %v's receiving session", j, i)
			assert.True(t, sessionDriver1.HasUserReceivingSession(users[j].GetUserID()), "User %v should have User %v's receiving session", i, j)
		}
	}

	fmt.Println("Create Server Side Group: ", memberSize, " members, ", chatbotSize, " chatbots finished")

	for i := 0; i < chatbotSize; i++ {
		// Invite chatbot i to the group
		users[0].RequestInviteChatbotToGroup(groupId, pb.GroupType_SERVER_SIDE, chatbots[i].GetChatbotID(), isIGA, isPseudo)

		// Chatbot i should receive a group invitation
		msgc, success := timeOutReadFromChatbotMessageChannel(chatbots[i].GetMessageChan())
		assert.True(t, success, "Chatbot %v should receive a group invitation from User 0", i)
		assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_CHATBOT_INVITATION, "Chatbot %v should receive a group chatbot invitation from User 0", i)

		// All users should receive a GROUP_CHATBOT_ADDITION event.
		for j := 0; j < memberSize; j++ {
			msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
			assert.True(t, success, "User %v should receive a group addition event", j)
			assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "User %v should receive a group chatbot addition event", j)
		}

		if !isIGA {
			// Chatbot i distributes its sender key.
			err = chatbots[i].DistributeSelfSenderKeyToAll(groupId)
			assert.Nil(t, err, "Chatbot %v should be able to distribute its sender key", i)

			// Users should receive sender key distribution messages from the new chatbot.
			for j := 0; j < memberSize; j++ {
				msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
				assert.True(t, success, "User %v should receive a sender key distribution message from Chatbot %v", j, i)
				assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "User %v should receive a sender key distribution message from Chatbot %v", j, i)
			}

			// Chatbot i should receive sender key distribution messages from all users.
			for j := 0; j < memberSize; j++ {
				msg, success := timeOutReadFromChatbotMessageChannel(chatbots[i].GetMessageChan())
				assert.True(t, success, "Chatbot %v should receive a sender key distribution message from User %v", i, j)
				assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "Chatbot %v should receive a sender key distribution message from User %v", i, j)
			}

			// Verify if the chatbot has the receiving sessions of all users and vice versa.
			chatbotSessionDriver, err := chatbots[i].Client.GetServerSideGroupSessionDriver(groupId)
			assert.Nil(t, err, "Chatbot %v should have the group session", i)
			for j := 0; j < memberSize; j++ {
				assert.True(t, chatbotSessionDriver.HasUserReceivingSession(users[j].GetUserID()), "Chatbot %v should have User %v's receiving session", i, j)

				userSessionDriver, err := users[j].Client.GetServerSideGroupSessionDriver(groupId)
				assert.Nil(t, err, "User %v should have the group session", j)
				assert.True(t, userSessionDriver.HasUserReceivingSession(chatbots[i].GetChatbotID()), "User %v should have Chatbot %v's receiving session", j, i)
			}
		}
	}

	if isPseudo {
		// Issue pseudonyms to each chatbot
		for i := 0; i < chatbotSize; i++ {
			for j := 0; j < memberSize; j++ {
				// User j issues a pseudonym to chatbot i
				err = users[j].CreateAndRegisterServerSidePseudonym(groupId, chatbots[i].GetChatbotID())
				assert.Nil(t, err, "User %v should be able to create and register a pseudonym", j)

				// Chatbot i should receive a pseudonym registration message
				msgc, success := timeOutReadFromChatbotMessageChannel(chatbots[i].GetMessageChan())
				assert.True(t, success, "Chatbot %v should receive a pseudonym registration message", i)
				assert.Equal(t, msgc.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "Chatbot %v should receive a pseudonym registration message", i)

				// All users except user j should receive a PSEUDONYM_REGISTRATION event.
				for k := 0; k < memberSize; k++ {
					if k == j {
						continue
					}
					msg, success := timeOutReadFromUserMessageChannel(users[k].GetMessageChan())
					assert.True(t, success, "User %v should receive a pseudonym registration event", k)
					assert.Equal(t, msg.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "User %v should receive a pseudonym registration event", k)
				}
			}
		}
	}

	return groupId
}

func createMlsGroupOfSize(t *testing.T, memberSize int, chatbotSize int, isIGA bool, isPseudo bool) string {
	ensureUsersAndChatbots(t, memberSize, chatbotSize)

	// User 0 is the initiator of the group
	groupId, err := users[0].CreateGroup(pb.GroupType_MLS)
	assert.Nil(t, err, "User 0 should be able to create a group")

	for i := 1; i < memberSize; i++ {
		// Invite user i to the group
		users[0].RequestInviteUserToGroup(groupId, pb.GroupType_MLS, users[i].GetUserID())

		// User i should receive a group invitation
		msg, success := timeOutReadFromUserMessageChannel(users[i].GetMessageChan())
		assert.True(t, success, "User %v should receive a group invitation from User 0", i)
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_INVITATION, "User %v should receive a group invitation from User 0", i)

		// User 0 ~ i-1 should receive a GROUP_ADDITION event.
		for j := 0; j < i; j++ {
			msg, success = timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
			assert.True(t, success, "User %v should receive a group addition event", j)
			assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_ADDITION, "User %v should receive a group addition event", j)
		}
	}

	// Ensure that all users have the same group state
	for i := 0; i < memberSize; i++ {
		user1SessionDriver, err := users[i].Client.GetMlsGroupSessionDriver(groupId)
		assert.Nil(t, err, "User %v should have the group session", i)
		for j := 0; j < memberSize; j++ {
			user2SessionDriver, err := users[j].Client.GetMlsGroupSessionDriver(groupId)
			assert.Nil(t, err, "User %v should have the group session", j)
			assert.True(t, user1SessionDriver.GetGroupState().Equals(*user2SessionDriver.GetGroupState()), "User's MLS state should be consistent")
		}
	}

	for i := 0; i < chatbotSize; i++ {
		// Invite chatbot i to the group
		users[0].RequestInviteChatbotToGroup(groupId, pb.GroupType_MLS, chatbots[i].GetChatbotID(), isIGA, isPseudo)

		// Chatbot i should receive a group invitation
		msgc, success := timeOutReadFromChatbotMessageChannel(chatbots[i].GetMessageChan())
		assert.True(t, success, "Chatbot %v should receive a group invitation from User 0", i)
		assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_CHATBOT_INVITATION, "Chatbot %v should receive a group chatbot invitation from User 0", i)

		// All users should receive a GROUP_CHATBOT_ADDITION event.
		for j := 0; j < memberSize; j++ {
			msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
			assert.True(t, success, "User %v should receive a group addition event", j)
			assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "User %v should receive a group chatbot addition event", j)
		}
	}

	if isIGA {
		// Verify that all users and chatbots have the same MLS Multi tree root
		for i := 0; i < memberSize; i++ {
			for j := 0; j < chatbotSize; j++ {
				userSessionDriver, err := users[i].Client.GetMlsGroupSessionDriver(groupId)
				assert.Nil(t, err, "User %v should have the group session", i)
				chatbotSessionDriver, err := chatbots[j].Client.GetMlsGroupSessionDriver(groupId)
				assert.Nil(t, err, "Chatbot %v should have the group session", j)
				assert.Equal(t, userSessionDriver.GetMlsMultiTree().GetRootSecret(chatbots[j].GetChatbotID()), chatbotSessionDriver.GetMlsMultiTreeExternal().GetRootSecret(), "User %v and Chatbot %v should have the same MLS tree root", i, j)
			}
		}
	} else {
		// Verify that all users and chatbots have the same MLS state
		// for i := 0; i < memberSize; i++ {
		// 	for j := 0; j < chatbotSize; j++ {
		// 		userSessionDriver, err := users[i].Client.GetMlsGroupSessionDriver(groupId)
		// 		assert.Nil(t, err, "User %v should have the group session", i)
		// 		chatbotSessionDriver, err := chatbots[j].Client.GetMlsGroupSessionDriver(groupId)
		// 		assert.Nil(t, err, "Chatbot %v should have the group session", j)
		// 		assert.True(t, userSessionDriver.GetGroupState().Equals(*chatbotSessionDriver.GetGroupState()), "User %v and Chatbot %v should have the same MLS state", i, j)
		// 	}
		// }
	}

	if isPseudo {
		// Issue pseudonyms to each chatbot
		for i := 0; i < chatbotSize; i++ {
			for j := 0; j < memberSize; j++ {
				// User j issues a pseudonym to chatbot i
				err = users[j].CreateAndRegisterMlsPseudonym(groupId, chatbots[i].GetChatbotID())
				assert.Nil(t, err, "User %v should be able to create and register a pseudonym", j)

				// Chatbot i should receive a pseudonym registration message
				msgc, success := timeOutReadFromChatbotMessageChannel(chatbots[i].GetMessageChan())
				assert.True(t, success, "Chatbot %v should receive a pseudonym registration message", i)
				assert.Equal(t, msgc.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "Chatbot %v should receive a pseudonym registration message", i)

				// All users except user j should receive a PSEUDONYM_REGISTRATION event.
				//for k := 0; k < memberSize; k++ {
				//	if k == j {
				//		continue
				//	}
				//	msg, success := timeOutReadFromUserMessageChannel(users[k].GetMessageChan())
				//	assert.True(t, success, "User %v should receive a pseudonym registration event", k)
				//	assert.Equal(t, msg.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "User %v should receive a pseudonym registration event", k)
				//}
			}
		}
	}

	fmt.Println("Create MLS Group: ", memberSize, " members, ", chatbotSize, " chatbots finished")

	return groupId
}

func benchmarkChatbotAddition(t *testing.T, memberSize int, isIGA bool, isPseudo bool, headerMessage string) {
	numberOfExperiments := 100
	createUsersAndChatbots(t, memberSize, numberOfExperiments)
	groupId := createServerSideGroupOfSize(t, memberSize, 0, isIGA, isPseudo)

	bench := hrtime.NewBenchmark(numberOfExperiments)
	for i := 0; bench.Next(); i++ {
		// Invite chatbot i to the group
		users[0].RequestInviteChatbotToGroup(groupId, pb.GroupType_SERVER_SIDE, chatbots[i].GetChatbotID(), isIGA, isPseudo)

		// Chatbot i should receive a group invitation
		msgc, success := timeOutReadFromChatbotMessageChannel(chatbots[i].GetMessageChan())
		assert.True(t, success, "Chatbot %v should receive a group invitation from User 0", i)
		assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_CHATBOT_INVITATION, "Chatbot %v should receive a group chatbot invitation from User 0", i)

		// All users should receive a GROUP_CHATBOT_ADDITION event.
		for j := 0; j < memberSize; j++ {
			msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
			assert.True(t, success, "User %v should receive a group addition event", j)
			assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "User %v should receive a group chatbot addition event", j)
		}

		if !isIGA {
			// Chatbot i distributes its sender key.
			err := chatbots[i].DistributeSelfSenderKeyToAll(groupId)
			assert.Nil(t, err, "Chatbot %v should be able to distribute its sender key", i)

			// Users should receive sender key distribution messages from the new chatbot.
			for j := 0; j < memberSize; j++ {
				msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
				assert.True(t, success, "User %v should receive a sender key distribution message from Chatbot %v", j, i)
				assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "User %v should receive a sender key distribution message from Chatbot %v", j, i)
			}

			// Chatbot i should receive sender key distribution messages from all users.
			for j := 0; j < memberSize; j++ {
				msg, success := timeOutReadFromChatbotMessageChannel(chatbots[i].GetMessageChan())
				assert.True(t, success, "Chatbot %v should receive a sender key distribution message from User %v", i, j)
				assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "Chatbot %v should receive a sender key distribution message from User %v", i, j)
			}

			// Verify if the chatbot has the receiving sessions of all users and vice versa.
			chatbotSessionDriver, err := chatbots[i].Client.GetServerSideGroupSessionDriver(groupId)
			assert.Nil(t, err, "Chatbot %v should have the group session", i)
			for j := 0; j < memberSize; j++ {
				assert.True(t, chatbotSessionDriver.HasUserReceivingSession(users[j].GetUserID()), "Chatbot %v should have User %v's receiving session", i, j)

				userSessionDriver, err := users[j].Client.GetServerSideGroupSessionDriver(groupId)
				assert.Nil(t, err, "User %v should have the group session", j)
				assert.True(t, userSessionDriver.HasUserReceivingSession(chatbots[i].GetChatbotID()), "User %v should have Chatbot %v's receiving session", j, i)
			}
		}

		if isPseudo {
			// Issue pseudonyms to each chatbot
			for j := 0; j < memberSize; j++ {
				// User j issues a pseudonym to chatbot i
				err := users[j].CreateAndRegisterServerSidePseudonym(groupId, chatbots[i].GetChatbotID())
				assert.Nil(t, err, "User %v should be able to create and register a pseudonym", j)

				// Chatbot i should receive a pseudonym registration message
				msgc, success := timeOutReadFromChatbotMessageChannel(chatbots[i].GetMessageChan())
				assert.True(t, success, "Chatbot %v should receive a pseudonym registration message", i)
				assert.Equal(t, msgc.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "Chatbot %v should receive a pseudonym registration message", i)

				// All users except user j should receive a PSEUDONYM_REGISTRATION event.
				for k := 0; k < memberSize; k++ {
					if k == j {
						continue
					}
					msg, success := timeOutReadFromUserMessageChannel(users[k].GetMessageChan())
					assert.True(t, success, "User %v should receive a pseudonym registration event", k)
					assert.Equal(t, msg.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "User %v should receive a pseudonym registration event", k)
				}
			}
		}
	}

	fmt.Println("====================  ", headerMessage, "  ====================")
	fmt.Println(bench.Histogram(10))
}

func benchmarkChatbotAdditionSingleUser(t *testing.T, memberSize int, isIGA bool, isPseudo bool, headerMessage string) {
	numberOfExperiments := 100
	createUsersAndChatbots(t, memberSize, numberOfExperiments)
	groupId := createServerSideGroupOfSize(t, memberSize, 0, isIGA, isPseudo)

	// Deactivate all users except user 0
	for i := 1; i < memberSize; i++ {
		deactivateUserById(t, i)
	}

	bench := hrtime.NewBenchmark(numberOfExperiments)
	for i := 0; bench.Next(); i++ {
		// Invite chatbot i to the group
		users[0].RequestInviteChatbotToGroup(groupId, pb.GroupType_SERVER_SIDE, chatbots[i].GetChatbotID(), isIGA, isPseudo)

		// Chatbot i should receive a group invitation
		msgc, success := timeOutReadFromChatbotMessageChannel(chatbots[i].GetMessageChan())
		assert.True(t, success, "Chatbot %v should receive a group invitation from User 0", i)
		assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_CHATBOT_INVITATION, "Chatbot %v should receive a group chatbot invitation from User 0", i)

		deactivateChatbotById(t, i)

		// User 0 should receive a GROUP_CHATBOT_ADDITION event.
		msg, success := timeOutReadFromUserMessageChannel(users[0].GetMessageChan())
		assert.True(t, success, "User 0 should receive a group chatbot addition event", 0)
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "User 0 should receive a group chatbot addition event", 0)

		if !isIGA {
			// Chatbot i distributes its sender key.
			err := chatbots[i].DistributeSelfSenderKeyToAll(groupId)
			assert.Nil(t, err, "Chatbot %v should be able to distribute its sender key", i)

			// User 0 should receive sender key distribution messages from the new chatbot.
			msg, success := timeOutReadFromUserMessageChannel(users[0].GetMessageChan())
			assert.True(t, success, "User 0 should receive a sender key distribution message from Chatbot %v", i)
			assert.Equal(t, msg.MessageType, pb.MessageType_SENDER_KEY_DISTRIBUTION_MESSAGE, "User 0 should receive a sender key distribution message from Chatbot %v", i)
		}

		if isPseudo {
			// User 0 issues a pseudonym to chatbot i
			err := users[0].CreateAndRegisterServerSidePseudonym(groupId, chatbots[i].GetChatbotID())
			assert.Nil(t, err, "User 0 should be able to create and register a pseudonym")
		}
	}

	fmt.Println("====================  ", headerMessage, "  ====================")
	fmt.Println(bench.Histogram(10))
}

func benchmarkMlsChatbotAddition(t *testing.T, memberSize int, isIGA bool, isPseudo bool, headerMessage string) {
	numberOfExperiments := 100
	createUsersAndChatbots(t, memberSize, numberOfExperiments)
	groupId := createMlsGroupOfSize(t, memberSize, 0, isIGA, isPseudo)

	bench := hrtime.NewBenchmark(numberOfExperiments)
	for chatbotId := 0; bench.Next(); chatbotId++ {
		// Invite chatbot i to the group
		users[0].RequestInviteChatbotToGroup(groupId, pb.GroupType_MLS, chatbots[chatbotId].GetChatbotID(), isIGA, isPseudo)

		// Chatbot i should receive a group invitation
		msgc, success := timeOutReadFromChatbotMessageChannel(chatbots[chatbotId].GetMessageChan())
		assert.True(t, success, "Chatbot %v should receive a group invitation from User 0", chatbotId)
		assert.Equal(t, msgc.EventType, pb.ServerEventType_GROUP_CHATBOT_INVITATION, "Chatbot %v should receive a group chatbot invitation from User 0", chatbotId)

		// All users should receive a GROUP_CHATBOT_ADDITION event.
		for j := 0; j < memberSize; j++ {
			msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
			assert.True(t, success, "User %v should receive a group addition event", j)
			assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "User %v should receive a group chatbot addition event", j)
		}

		if isPseudo {
			// Issue pseudonyms to each chatbot
			for i := 0; i <= chatbotId; i++ {
				for j := 0; j < memberSize; j++ {
					// User j issues a pseudonym to chatbot chatbotId
					err := users[j].CreateAndRegisterMlsPseudonym(groupId, chatbots[i].GetChatbotID())
					assert.Nil(t, err, "User %v should be able to create and register a pseudonym", j)

					// Chatbot chatbotId should receive a pseudonym registration message
					msgc, success := timeOutReadFromChatbotMessageChannel(chatbots[i].GetMessageChan())
					assert.True(t, success, "Chatbot %v should receive a pseudonym registration message", i)
					assert.Equal(t, msgc.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "Chatbot %v should receive a pseudonym registration message", i)

					// All users except user j should receive a PSEUDONYM_REGISTRATION event.
					//for k := 0; k < memberSize; k++ {
					//	if k == j {
					//		continue
					//	}
					//	msg, success := timeOutReadFromUserMessageChannel(users[k].GetMessageChan())
					//	assert.True(t, success, "User %v should receive a pseudonym registration event", k)
					//	assert.Equal(t, msg.MessageType, pb.MessageType_PSEUDONYM_REGISTRATION_MESSAGE, "User %v should receive a pseudonym registration event", k)
					//}
				}
			}
		}
	}

	fmt.Println("====================  ", headerMessage, "  ====================")
	fmt.Println(bench.Histogram(10))
}

func benchmarkMlsChatbotAdditionSingleUser(t *testing.T, memberSize int, isIGA bool, isPseudo bool, headerMessage string) {
	numberOfExperiments := 100
	createUsersAndChatbots(t, memberSize, numberOfExperiments)
	groupId := createMlsGroupOfSize(t, memberSize, 0, isIGA, isPseudo)

	// Deactivate all users except user 0
	for i := 1; i < memberSize; i++ {
		deactivateUserById(t, i)
	}

	// Deactivate all chatbots
	for i := 0; i < numberOfExperiments; i++ {
		deactivateChatbotById(t, i)
	}

	bench := hrtime.NewBenchmark(numberOfExperiments)
	for chatbotId := 0; bench.Next(); chatbotId++ {
		// Invite chatbot i to the group
		users[0].RequestInviteChatbotToGroup(groupId, pb.GroupType_MLS, chatbots[chatbotId].GetChatbotID(), isIGA, isPseudo)

		// User 0 should receive a GROUP_CHATBOT_ADDITION event.
		msg, success := timeOutReadFromUserMessageChannel(users[0].GetMessageChan())
		assert.True(t, success, "User 0 should receive a group chatbot addition event", 0)
		assert.Equal(t, msg.EventType, pb.ServerEventType_GROUP_CHATBOT_ADDITION, "User 0 should receive a group chatbot addition event", 0)

		if isPseudo {
			// Issue pseudonyms to each chatbot
			for i := 0; i <= chatbotId; i++ {
				// User 0 issues a pseudonym to chatbot chatbotId
				err := users[0].CreateAndRegisterMlsPseudonym(groupId, chatbots[i].GetChatbotID())
				assert.Nil(t, err, "User 0 should be able to create and register a pseudonym")
			}
		}
	}

	fmt.Println("====================  ", headerMessage, "  ====================")
	fmt.Println(bench.Histogram(10))
}

func benchmarkUserSendServerSideGroupMessage(t *testing.T, memberSize int, chatbotSize int, isIGA bool, isPseudo bool, headerMessage string) {
	createUsersAndChatbots(t, memberSize, chatbotSize)
	groupId := createServerSideGroupOfSize(t, memberSize, chatbotSize, isIGA, isPseudo)

	// Get chatbots list from user 0's session driver
	sessionDriver, err := users[0].Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "User 0 should have the group session")
	chatbotIdList := sessionDriver.GetGroupChatbots()

	bench := hrtime.NewBenchmark(numberOfExperiments)
	for i := 0; bench.Next(); i++ {
		// User i % memberSize sends a message to the group
		err := users[i%memberSize].SendServerSideGroupMessage(groupId, []byte(fmt.Sprintf("Hello everyone! I'm User %v.", i%memberSize)), pb.MessageType_TEXT_MESSAGE, chatbotIdList, false)
		assert.Nil(t, err, "User %v should be able to send a message to the group", i%memberSize)

		// All members should receive the message
		for j := 0; j < memberSize; j++ {
			if j == i%memberSize {
				continue
			}

			msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
			assert.True(t, success, "User %v should receive a message from User %v", j, i%memberSize)
			assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "User %v should receive a text message from User %v", j, i%memberSize)
			assert.Equal(t, fmt.Sprintf("Hello everyone! I'm User %v.", i%memberSize), string(msg.Message), "User %v should receive the same message from User %v", j, i%memberSize)
		}

		// All chatbots should receive the message
		for j := 0; j < chatbotSize; j++ {
			msg, success := timeOutReadFromChatbotMessageChannel(chatbots[j].GetMessageChan())
			assert.True(t, success, "Chatbot %v should receive a message from User %v", j, i%memberSize)
			assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot %v should receive a text message from User %v", j, i%memberSize)
			assert.Equal(t, fmt.Sprintf("Hello everyone! I'm User %v.", i%memberSize), string(msg.Message), "Chatbot %v should receive the same message from User %v", j, i%memberSize)
		}

		if isIGA {
			/* We no longer need to send a validation message after the protocol update.
			// All members should receive the validation message from all chatbots.
			for j := 0; j < memberSize; j++ {
				for k := 0; k < chatbotSize; k++ {
					msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
					assert.True(t, success, "User %v should receive a validation message", j)
					assert.Equal(t, msg.MessageType, pb.MessageType_VALIDATION_MESSAGE, "User %v should receive a validation message from Chatbot %v", j, k)
				}
			}
			*/
		}
	}

	fmt.Println("====================  ", headerMessage, "  ====================")
	fmt.Println(bench.Histogram(10))
}

func benchmarkUserSendServerSideGroupHideTriggerMessage(t *testing.T, memberSize int, chatbotSize int, isIGA bool, isPseudo bool, headerMessage string) {
	createUsersAndChatbots(t, memberSize, chatbotSize)
	groupId := createServerSideGroupOfSize(t, memberSize, chatbotSize, isIGA, isPseudo)

	// Get chatbots list from user 0's session driver
	//sessionDriver, err := users[0].Client.GetServerSideGroupSessionDriver(groupId)
	//assert.Nil(t, err, "User 0 should have the group session")
	//chatbotIdList := sessionDriver.GetGroupChatbots()

	bench := hrtime.NewBenchmark(numberOfExperiments)
	for i := 0; bench.Next(); i++ {
		// User i % memberSize sends a message to the group
		err := users[i%memberSize].SendServerSideGroupMessage(groupId, []byte(fmt.Sprintf("Hello everyone! I'm User %v.", i%memberSize)), pb.MessageType_TEXT_MESSAGE, []string{}, true)
		assert.Nil(t, err, "User %v should be able to send a message to the group", i%memberSize)

		// All members should receive the message
		for j := 0; j < memberSize; j++ {
			if j == i%memberSize {
				continue
			}

			msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
			assert.True(t, success, "User %v should receive a message from User %v", j, i%memberSize)
			assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "User %v should receive a text message from User %v", j, i%memberSize)
			assert.Equal(t, fmt.Sprintf("Hello everyone! I'm User %v.", i%memberSize), string(msg.Message), "User %v should receive the same message from User %v", j, i%memberSize)
		}

		// All chatbots should receive the SKIP message
		for j := 0; j < chatbotSize; j++ {
			msg, success := timeOutReadFromChatbotMessageChannel(chatbots[j].GetMessageChan())
			assert.True(t, success, "Chatbot %v should receive a message from User %v", j, i%memberSize)
			assert.Equal(t, msg.MessageType, pb.MessageType_SKIP, "Chatbot %v should receive a SKIP message from User %v", j, i%memberSize)
		}

		/* We no longer need to send a validation message after the protocol update.
		// All members should receive the validation message from all chatbots.
		for j := 0; j < memberSize; j++ {
			for k := 0; k < chatbotSize; k++ {
				msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
				assert.True(t, success, "User %v should receive a validation message", j)
				assert.Equal(t, msg.MessageType, pb.MessageType_VALIDATION_MESSAGE, "User %v should receive a validation message from Chatbot %v", j, k)
			}
		}
		*/
	}

	fmt.Println("====================  ", headerMessage, "  ====================")
	fmt.Println(bench.Histogram(10))
}

func benchmarkUserGenerateServerSideGroupMessage(t *testing.T, memberSize int, chatbotSize int, isIGA bool, isPseudo bool, hideTrigger bool, headerMessage string) {
	createUsersAndChatbots(t, memberSize, chatbotSize)
	groupId := createServerSideGroupOfSize(t, memberSize, chatbotSize, isIGA, isPseudo)

	// Get chatbots list from user 0's session driver
	sessionDriver, err := users[0].Client.GetServerSideGroupSessionDriver(groupId)
	assert.Nil(t, err, "User 0 should have the group session")
	chatbotIdList := sessionDriver.GetGroupChatbots()

	bench := hrtime.NewBenchmark(numberOfExperiments)
	for i := 0; bench.Next(); i++ {
		// User 0 sends a message to the group
		_, err := users[0].GenerateServerSideGroupMessageCipherText(groupId, []byte(fmt.Sprintf("Hello everyone! This is message %v.", i)), pb.MessageType_TEXT_MESSAGE, chatbotIdList, hideTrigger)
		assert.Nil(t, err, "User 0 should be able to send message %v to the group", i)
	}

	fmt.Println("====================  ", headerMessage, "  ====================")
	fmt.Println(bench.Histogram(10))
}

func benchmarkUserSendMlsGroupMessage(t *testing.T, memberSize int, chatbotSize int, isIGA bool, isPseudo bool, headerMessage string) {
	createUsersAndChatbots(t, memberSize, chatbotSize)
	groupId := createMlsGroupOfSize(t, memberSize, chatbotSize, isIGA, isPseudo)

	// Get chatbots list from user 0's session driver
	sessionDriver, err := users[0].Client.GetMlsGroupSessionDriver(groupId)
	assert.Nil(t, err, "User 0 should have the group session")
	chatbotIdList := sessionDriver.GetGroupChatbots()

	bench := hrtime.NewBenchmark(numberOfExperiments)
	for i := 0; bench.Next(); i++ {
		// User i % memberSize sends a message to the group
		err := users[i%memberSize].SendMlsGroupMessage(groupId, []byte(fmt.Sprintf("Hello everyone! I'm User %v.", i%memberSize)), pb.MessageType_TEXT_MESSAGE, chatbotIdList, false)
		assert.Nil(t, err, "User %v should be able to send a message to the group", i%memberSize)

		// All members should receive the message
		for j := 0; j < memberSize; j++ {
			if j == i%memberSize {
				continue
			}

			msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
			assert.True(t, success, "User %v should receive a message from User %v", j, i%memberSize)
			assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "User %v should receive a text message from User %v", j, i%memberSize)
			assert.Equal(t, fmt.Sprintf("Hello everyone! I'm User %v.", i%memberSize), string(msg.Message), "User %v should receive the same message from User %v", j, i%memberSize)
		}

		// All chatbots should receive the message
		for j := 0; j < chatbotSize; j++ {
			msg, success := timeOutReadFromChatbotMessageChannel(chatbots[j].GetMessageChan())
			assert.True(t, success, "Chatbot %v should receive a message from User %v", j, i%memberSize)
			assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "Chatbot %v should receive a text message from User %v", j, i%memberSize)
			assert.Equal(t, fmt.Sprintf("Hello everyone! I'm User %v.", i%memberSize), string(msg.Message), "Chatbot %v should receive the same message from User %v", j, i%memberSize)
		}

		if isIGA {
			/* We no longer need to send a validation message after the protocol update.
			// All members should receive the validation message from all chatbots.
			for j := 0; j < memberSize; j++ {
				for k := 0; k < chatbotSize; k++ {
					msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
					assert.True(t, success, "User %v should receive a validation message", j)
					assert.Equal(t, msg.MessageType, pb.MessageType_VALIDATION_MESSAGE, "User %v should receive a validation message from Chatbot %v", j, k)
				}
			}
			*/
		}
	}

	fmt.Println("====================  ", headerMessage, "  ====================")
	fmt.Println(bench.Histogram(10))
}

func benchmarkUserSendMlsGroupHideTriggerMessage(t *testing.T, memberSize int, chatbotSize int, isIGA bool, isPseudo bool, headerMessage string) {
	createUsersAndChatbots(t, memberSize, chatbotSize)
	groupId := createMlsGroupOfSize(t, memberSize, chatbotSize, isIGA, isPseudo)

	bench := hrtime.NewBenchmark(numberOfExperiments)
	for i := 0; bench.Next(); i++ {
		// User i % memberSize sends a message to the group
		err := users[i%memberSize].SendMlsGroupMessage(groupId, []byte(fmt.Sprintf("Hello everyone! I'm User %v.", i%memberSize)), pb.MessageType_TEXT_MESSAGE, []string{}, true)
		assert.Nil(t, err, "User %v should be able to send a message to the group", i%memberSize)

		// All members should receive the message
		for j := 0; j < memberSize; j++ {
			if j == i%memberSize {
				continue
			}

			msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
			assert.True(t, success, "User %v should receive a message from User %v", j, i%memberSize)
			assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "User %v should receive a text message from User %v", j, i%memberSize)
			assert.Equal(t, fmt.Sprintf("Hello everyone! I'm User %v.", i%memberSize), string(msg.Message), "User %v should receive the same message from User %v", j, i%memberSize)
		}

		// All chatbots should receive the SKIP message
		for j := 0; j < chatbotSize; j++ {
			msg, success := timeOutReadFromChatbotMessageChannel(chatbots[j].GetMessageChan())
			assert.True(t, success, "Chatbot %v should receive a message from User %v", j, i%memberSize)
			assert.Equal(t, msg.MessageType, pb.MessageType_SKIP, "Chatbot %v should receive a SKIP message from User %v", j, i%memberSize)
		}

		/* We no longer need to send a validation message after the protocol update.
		// All members should receive the validation message from all chatbots.
		for j := 0; j < memberSize; j++ {
			for k := 0; k < chatbotSize; k++ {
				msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
				assert.True(t, success, "User %v should receive a validation message", j)
				assert.Equal(t, msg.MessageType, pb.MessageType_VALIDATION_MESSAGE, "User %v should receive a validation message from Chatbot %v", j, k)
			}
		}
		*/
	}

	fmt.Println("====================  ", headerMessage, "  ====================")
	fmt.Println(bench.Histogram(10))
}

func benchmarkUserGenerateMlsGroupMessage(t *testing.T, memberSize int, chatbotSize int, isIGA bool, isPseudo bool, hideTrigger bool, headerMessage string) {
	createUsersAndChatbots(t, memberSize, chatbotSize)
	groupId := createMlsGroupOfSize(t, memberSize, chatbotSize, isIGA, isPseudo)

	// Get chatbots list from user 0's session driver
	sessionDriver, err := users[0].Client.GetMlsGroupSessionDriver(groupId)
	assert.Nil(t, err, "User 0 should have the group session")
	chatbotIdList := sessionDriver.GetGroupChatbots()

	bench := hrtime.NewBenchmark(numberOfExperiments)
	for i := 0; bench.Next(); i++ {
		// User 0 sends a message to the group
		_, err := users[0].GenerateMlsGroupMessageCipherText(groupId, []byte(fmt.Sprintf("Hello everyone! This is message %v.", i)), pb.MessageType_TEXT_MESSAGE, chatbotIdList, hideTrigger)
		assert.Nil(t, err, "User 0 should be able to send message %v to the group", i)
	}

	fmt.Println("====================  ", headerMessage, "  ====================")
	fmt.Println(bench.Histogram(10))
}

func benchmarkChatbotSendServerSideGroupMessage(t *testing.T, memberSize int, chatbotSize int) {
	groupId := createServerSideGroupOfSize(t, memberSize, chatbotSize, false, false)
	bench := hrtime.NewBenchmark(numberOfExperiments)

	for i := 0; bench.Next(); i++ {
		// Chatbot i % chatbotSize sends a message to the group
		err := chatbots[i%chatbotSize].SendServerSideGroupMessage(groupId, []byte(fmt.Sprintf("Hello everyone! I'm Chatbot %v.", i%chatbotSize)), pb.MessageType_TEXT_MESSAGE)
		assert.Nil(t, err, "Chatbot %v should be able to send a message to the group", i%chatbotSize)

		// All members should receive the message
		for j := 0; j < memberSize; j++ {
			msg, success := timeOutReadFromUserMessageChannel(users[j].GetMessageChan())
			assert.True(t, success, "User %v should receive a message from Chatbot %v", j, i%chatbotSize)
			assert.Equal(t, msg.MessageType, pb.MessageType_TEXT_MESSAGE, "User %v should receive a text message from Chatbot %v", j, i%chatbotSize)
			assert.Equal(t, fmt.Sprintf("Hello everyone! I'm Chatbot %v.", i%chatbotSize), string(msg.Message), "User %v should receive the same message from Chatbot %v", j, i%chatbotSize)
		}
	}

	fmt.Println("========== Send Server Side Group Message: ", memberSize, " members, ", chatbotSize, " chatbots ==========")
	fmt.Println(bench.Histogram(10))
}

func TestBenchmarkSendIndividualMessage(t *testing.T) {
	createUsersAndChatbots(t, 10, 0)
	benchmarkSendIndividualMessage(t, 10)
}

func TestBenchmarkChatbotAddition(t *testing.T) {
	for memberSize := 5; memberSize <= 50; memberSize += 5 {
		benchmarkChatbotAddition(t, memberSize, false, false, fmt.Sprintf("Chatbot Addition: %v members", memberSize))
		benchmarkChatbotAddition(t, memberSize, true, false, fmt.Sprintf("Chatbot Addition: %v members, IGA", memberSize))
		benchmarkChatbotAddition(t, memberSize, true, true, fmt.Sprintf("Chatbot Addition: %v members, IGA, Pseudo", memberSize))
	}
}

func TestBenchmarkChatbotAdditionSingleUser(t *testing.T) {
	for memberSize := 5; memberSize <= 50; memberSize += 5 {
		benchmarkChatbotAdditionSingleUser(t, memberSize, false, false, fmt.Sprintf("Chatbot Addition: %v members", memberSize))
		benchmarkChatbotAdditionSingleUser(t, memberSize, true, false, fmt.Sprintf("Chatbot Addition: %v members, IGA", memberSize))
		benchmarkChatbotAdditionSingleUser(t, memberSize, true, true, fmt.Sprintf("Chatbot Addition: %v members, IGA, Pseudo", memberSize))
	}
}

func TestBenchmarkMlsChatbotAddition(t *testing.T) {
	for memberSize := 5; memberSize <= 50; memberSize += 5 {
		benchmarkMlsChatbotAddition(t, memberSize, false, false, fmt.Sprintf("MLS Chatbot Addition: %v members", memberSize))
		benchmarkMlsChatbotAddition(t, memberSize, true, false, fmt.Sprintf("MLS Chatbot Addition: %v members, IGA", memberSize))
		benchmarkMlsChatbotAddition(t, memberSize, true, true, fmt.Sprintf("MLS Chatbot Addition: %v members, IGA, Pseudo", memberSize))
	}
}

func TestBenchmarkMlsChatbotAdditionSingleUser(t *testing.T) {
	for memberSize := 5; memberSize <= 50; memberSize += 5 {
		benchmarkMlsChatbotAdditionSingleUser(t, memberSize, false, false, fmt.Sprintf("MLS Chatbot Addition: %v members", memberSize))
		benchmarkMlsChatbotAdditionSingleUser(t, memberSize, true, false, fmt.Sprintf("MLS Chatbot Addition: %v members, IGA", memberSize))
		benchmarkMlsChatbotAdditionSingleUser(t, memberSize, true, true, fmt.Sprintf("MLS Chatbot Addition: %v members, IGA, Pseudo", memberSize))
	}
}

func TestBenchmarkUserSendServerSideGroupMessageWithoutHideTrigger(t *testing.T) {
	benchmarkUserSendServerSideGroupMessage(t, 50, 1, false, false, "Send Server Side Group Message: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserSendServerSideGroupMessage(t, 50, chatbotSize, false, false, fmt.Sprintf("Send Server Side Group Message: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserGenerateServerSideGroupMessageWithoutHideTrigger(t *testing.T) {
	benchmarkUserGenerateServerSideGroupMessage(t, 50, 1, false, false, false, "Generate Server Side Group Message: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserGenerateServerSideGroupMessage(t, 50, chatbotSize, false, false, false, fmt.Sprintf("Generate Server Side Group Message: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserSendServerSideIGAGroupMessageWithoutHideTrigger(t *testing.T) {
	benchmarkUserSendServerSideGroupMessage(t, 50, 1, true, false, "Send Server Side IGA Group Message: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserSendServerSideGroupMessage(t, 50, chatbotSize, true, false, fmt.Sprintf("Send Server Side IGA Group Message: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserGenerateServerSideIGAGroupMessageWithoutHideTrigger(t *testing.T) {
	benchmarkUserGenerateServerSideGroupMessage(t, 50, 1, true, false, false, "Generate Server Side IGA Group Message: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserGenerateServerSideGroupMessage(t, 50, chatbotSize, true, false, false, fmt.Sprintf("Generate Server Side IGA Group Message: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserSendServerSidePseudoGroupMessageWithoutHideTrigger(t *testing.T) {
	benchmarkUserSendServerSideGroupMessage(t, 50, 1, true, true, "Send Server Side Pseudo Group Message: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserSendServerSideGroupMessage(t, 50, chatbotSize, true, true, fmt.Sprintf("Send Server Side Pseudo Group Message: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserGenerateServerSidePseudoGroupMessageWithoutHideTrigger(t *testing.T) {
	benchmarkUserGenerateServerSideGroupMessage(t, 50, 1, true, true, false, "Generate Server Side Pseudo Group Message: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserGenerateServerSideGroupMessage(t, 50, chatbotSize, true, true, false, fmt.Sprintf("Generate Server Side Pseudo Group Message: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserSendServerSideIGAGroupMessageWithHideTrigger(t *testing.T) {
	benchmarkUserSendServerSideGroupHideTriggerMessage(t, 50, 1, true, false, "Send Server Side IGA Group Message with Hide Trigger: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserSendServerSideGroupHideTriggerMessage(t, 50, chatbotSize, true, false, fmt.Sprintf("Send Server Side IGA Group Message with Hide Trigger: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserGenerateServerSideIGAGroupMessageWithHideTrigger(t *testing.T) {
	benchmarkUserGenerateServerSideGroupMessage(t, 50, 1, true, false, true, "Generate Server Side IGA Group Message with Hide Trigger: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserGenerateServerSideGroupMessage(t, 50, chatbotSize, true, false, true, fmt.Sprintf("Generate Server Side IGA Group Message with Hide Trigger: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserSendServerSidePseudoGroupMessageWithHideTrigger(t *testing.T) {
	benchmarkUserSendServerSideGroupHideTriggerMessage(t, 50, 1, true, true, "Send Server Side Pseudo Group Message with Hide Trigger: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserSendServerSideGroupHideTriggerMessage(t, 50, chatbotSize, true, true, fmt.Sprintf("Send Server Side Pseudo Group Message with Hide Trigger: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserGenerateServerSidePseudoGroupMessageWithHideTrigger(t *testing.T) {
	benchmarkUserGenerateServerSideGroupMessage(t, 50, 1, true, true, true, "Generate Server Side Pseudo Group Message with Hide Trigger: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserGenerateServerSideGroupMessage(t, 50, chatbotSize, true, true, true, fmt.Sprintf("Generate Server Side Pseudo Group Message with Hide Trigger: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserSendMlsGroupMessageWithoutHideTrigger(t *testing.T) {
	benchmarkUserSendMlsGroupMessage(t, 50, 1, false, false, "Send MLS Group Message: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserSendMlsGroupMessage(t, 50, chatbotSize, false, false, fmt.Sprintf("Send MLS Group Message: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserGenerateMlsGroupMessageWithoutHideTrigger(t *testing.T) {
	benchmarkUserGenerateMlsGroupMessage(t, 50, 1, false, false, false, "Generate MLS Group Message: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserGenerateMlsGroupMessage(t, 50, chatbotSize, false, false, false, fmt.Sprintf("Generate MLS Group Message: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserSendMlsIGAGroupMessageWithoutHideTrigger(t *testing.T) {
	benchmarkUserSendMlsGroupMessage(t, 50, 1, true, false, "Send MLS IGA Group Message: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserSendMlsGroupMessage(t, 50, chatbotSize, true, false, fmt.Sprintf("Send MLS IGA Group Message: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserGenerateMlsIGAGroupMessageWithoutHideTrigger(t *testing.T) {
	benchmarkUserGenerateMlsGroupMessage(t, 50, 1, true, false, false, "Generate MLS IGA Group Message: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserGenerateMlsGroupMessage(t, 50, chatbotSize, true, false, false, fmt.Sprintf("Generate MLS IGA Group Message: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserSendMlsIGAGroupMessageWithHideTrigger(t *testing.T) {
	benchmarkUserSendMlsGroupHideTriggerMessage(t, 50, 1, true, false, "Send MLS IGA Group Message with Hide Trigger: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserSendMlsGroupHideTriggerMessage(t, 50, chatbotSize, true, false, fmt.Sprintf("Send MLS IGA Group Message with Hide Trigger: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserGenerateMlsIGAGroupMessageWithHideTrigger(t *testing.T) {
	benchmarkUserGenerateMlsGroupMessage(t, 50, 1, true, false, true, "Generate MLS IGA Group Message with Hide Trigger: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserGenerateMlsGroupMessage(t, 50, chatbotSize, true, false, true, fmt.Sprintf("Generate MLS IGA Group Message with Hide Trigger: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserSendMlsPseudoGroupMessageWithoutHideTrigger(t *testing.T) {
	benchmarkUserSendMlsGroupMessage(t, 50, 1, true, true, "Send MLS Pseudo Group Message: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserSendMlsGroupMessage(t, 50, chatbotSize, true, true, fmt.Sprintf("Send MLS Pseudo Group Message: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserGenerateMlsPseudoGroupMessageWithoutHideTrigger(t *testing.T) {
	benchmarkUserGenerateMlsGroupMessage(t, 50, 1, true, true, false, "Generate MLS Pseudo Group Message: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserGenerateMlsGroupMessage(t, 50, chatbotSize, true, true, false, fmt.Sprintf("Generate MLS Pseudo Group Message: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserSendMlsPseudoGroupMessageWithHideTrigger(t *testing.T) {
	benchmarkUserSendMlsGroupHideTriggerMessage(t, 50, 1, true, true, "Send MLS Pseudo Group Message with Hide Trigger: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserSendMlsGroupHideTriggerMessage(t, 50, chatbotSize, true, true, fmt.Sprintf("Send MLS Pseudo Group Message with Hide Trigger: 50 members, %v chatbots", chatbotSize))
	}
}

func TestBenchmarkUserGenerateMlsPseudoGroupMessageWithHideTrigger(t *testing.T) {
	benchmarkUserGenerateMlsGroupMessage(t, 50, 1, true, true, true, "Generate MLS Pseudo Group Message with Hide Trigger: 50 members, 1 chatbot")

	for chatbotSize := 5; chatbotSize <= 50; chatbotSize += 5 {
		benchmarkUserGenerateMlsGroupMessage(t, 50, chatbotSize, true, true, true, fmt.Sprintf("Generate MLS Pseudo Group Message with Hide Trigger: 50 members, %v chatbots", chatbotSize))
	}
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
	userID := prefix + "-" + randomString(8)
	//user, _ := user.NewClientSideUser(userID, "localhost:50051", true)
	return user.NewClientSideUserBufconn(userID, dialer(), true)
}

// createClientSideChatbotWithRandomUserID create a client side chatbot with a random user ID.
func createClientSideChatbotWithRandomUserID(prefix string) *chatbot.ClientSideChatbot {
	chatbotID := prefix + "-" + randomString(8)
	//chatbot, _ := NewClientSideChatbot(chatbotID, "localhost:50051", true)
	return chatbot.NewClientSideChatbotBufconn(chatbotID, dialer(), true)
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
func timeOutReadFromChatbotMessageChannel(ch <-chan chatbot.OutputMessage) (chatbot.OutputMessage, bool) {
	for {
		select {
		case msg := <-ch:
			return msg, true
		case <-time.After(3 * time.Second):
			return chatbot.OutputMessage{}, false
		}
	}
}

// timeOutReadFromUserMessageChannel read from the given channel and return false if it times out.
func timeOutReadFromUserMessageChannel(ch <-chan user.OutputMessage) (user.OutputMessage, bool) {
	for {
		select {
		case msg := <-ch:
			return msg, true
		case <-time.After(3 * time.Second):
			return user.OutputMessage{}, false
		}
	}
}
