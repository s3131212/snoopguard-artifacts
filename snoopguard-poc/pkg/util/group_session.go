package util

import (
	"go.mau.fi/libsignal/groups"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/libsignal/protocol"
	"go.mau.fi/libsignal/serialize"
)

type GroupSessionWrapper struct {
	senderKeyName *protocol.SenderKeyName

	groupId  string
	selfUser *User

	serializer *serialize.Serializer

	groupSessionBuilder *groups.SessionBuilder
	groupCipher         *groups.GroupCipher
}

func NewGroupSessionWrapper(selfUser *User, senderKeyName *protocol.SenderKeyName, serializer *serialize.Serializer) *GroupSessionWrapper {
	gsw := &GroupSessionWrapper{}
	gsw.groupId = senderKeyName.GroupID()
	gsw.selfUser = selfUser
	gsw.senderKeyName = senderKeyName
	gsw.groupSessionBuilder = groups.NewGroupSessionBuilder(
		gsw.selfUser.senderKeyStore,
		serializer,
	)
	gsw.groupCipher = groups.NewGroupCipher(gsw.groupSessionBuilder, senderKeyName, gsw.selfUser.senderKeyStore)
	gsw.serializer = serializer
	return gsw
}

func (gsw *GroupSessionWrapper) ProcessSenderKeyRaw(senderKeyName *protocol.SenderKeyName, distributionMessage []byte) {
	d, err := protocol.NewSenderKeyDistributionMessageFromBytes(distributionMessage, gsw.serializer.SenderKeyDistributionMessage)
	if err != nil {
		logger.Error("Unable to process sender key distribution message: ", err)
		return
	}
	gsw.ProcessSenderKey(senderKeyName, d)
}

func (gsw *GroupSessionWrapper) ProcessSenderKey(senderKeyName *protocol.SenderKeyName, distributionMessage *protocol.SenderKeyDistributionMessage) {
	gsw.groupSessionBuilder.Process(senderKeyName, distributionMessage)
}

func (gsw *GroupSessionWrapper) DistributeSenderKey() *protocol.SenderKeyDistributionMessage {
	distributionMessage, err := gsw.groupSessionBuilder.Create(gsw.senderKeyName)
	if err != nil {
		logger.Error("Unable to encrypt message: ", err)
		panic("")
	}
	return distributionMessage
}

// EncryptGroupMessage is a helper function to send encrypted messages with the given cipher.
func (gsw *GroupSessionWrapper) EncryptGroupMessage(message []byte) protocol.GroupCiphertextMessage {
	logger.Debug("Encrypting message: ", string(message))
	encrypted, err := gsw.groupCipher.Encrypt(message)
	if err != nil {
		logger.Error("Unable to encrypt message: ", err)
		panic("")
	}
	logger.Debug("Encrypted message: ", encrypted)

	return encrypted
}

// DecryptGroupMessage is a helper function to decrypt messages of a session.
func (gsw *GroupSessionWrapper) DecryptGroupMessage(message protocol.GroupCiphertextMessage) []byte {
	senderKeyMessage := message.(*protocol.SenderKeyMessage)

	msg, err := gsw.groupCipher.Decrypt(senderKeyMessage)
	if err != nil {
		logger.Error("Unable to decrypt message: ", err)
		panic("")
	}

	return msg
}

func (gsw *GroupSessionWrapper) ParseRawMessage(rawMessage []byte) *protocol.SenderKeyMessage {
	encryptedMessage, err := protocol.NewSenderKeyMessageFromBytes(rawMessage, gsw.serializer.SenderKeyMessage)
	if err != nil {
		logger.Error("Unable to decode message as JSON: ", err)
		panic("")
	}

	return encryptedMessage
}
