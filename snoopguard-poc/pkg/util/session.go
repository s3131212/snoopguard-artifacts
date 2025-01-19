package util

import (
	"errors"
	"go.mau.fi/libsignal/keys/identity"
	"go.mau.fi/libsignal/keys/prekey"
	"go.mau.fi/libsignal/logger"
	"go.mau.fi/libsignal/protocol"
	"go.mau.fi/libsignal/serialize"
	"go.mau.fi/libsignal/session"
)

type SessionWrapper struct {
	address *protocol.SignalAddress

	identityKeyPair *identity.KeyPair
	registrationID  uint32

	selfUser *User

	serializer *serialize.Serializer

	sessionBuilder *session.Builder
	sessionCipher  *session.Cipher
}

func NewSessionWrapper(address *protocol.SignalAddress, selfUser *User, preKeyBundle *prekey.Bundle, serializer *serialize.Serializer) *SessionWrapper {
	sw := &SessionWrapper{}
	sw.address = address
	sw.selfUser = selfUser
	sw.sessionBuilder = session.NewBuilder(
		selfUser.sessionStore,
		selfUser.preKeyStore,
		selfUser.signedPreKeyStore,
		selfUser.identityStore,
		address,
		serializer,
	)

	sw.sessionCipher = session.NewCipher(sw.sessionBuilder, address)
	sw.serializer = serializer

	if preKeyBundle != nil {
		sw.sessionBuilder.ProcessBundle(preKeyBundle)
	}

	return sw
}

func (sw *SessionWrapper) EncryptMsg(message []byte) protocol.CiphertextMessage {
	logger.Debug("Encrypting message: ", message)
	encrypted, err := sw.sessionCipher.Encrypt(message)
	if err != nil {
		logger.Error("Unable to encrypt message: ", err)
		panic("")
	}

	return encrypted
}

func (sw *SessionWrapper) DecryptMsg(message protocol.CiphertextMessage) ([]byte, error) {
	switch message.(type) {
	case *protocol.PreKeySignalMessage:
		plain, err := sw.sessionCipher.DecryptMessage(message.(*protocol.PreKeySignalMessage))
		if err != nil {
			logger.Error("Unable to decrypt prekey message: ", err)
			return nil, err
		}
		return plain, nil
	case *protocol.SignalMessage:
		plain, err := sw.sessionCipher.Decrypt(message.(*protocol.SignalMessage))
		if err != nil {
			logger.Error("Unable to decrypt message: ", err)
			return nil, err
		}
		return plain, nil
	default:
		logger.Error("Unknown message type")
		return nil, errors.New("unknown message type")
	}
}

func (sw *SessionWrapper) ParseRawMessage(rawMessage []byte, hasPreKey bool) protocol.CiphertextMessage {
	if hasPreKey {
		encryptedMessage, err := protocol.NewPreKeySignalMessageFromBytes(rawMessage, sw.serializer.PreKeySignalMessage, sw.serializer.SignalMessage)
		if err != nil {
			logger.Error("Unable to restore message (with prekey) as JSON: ", err)
			panic("")
		}
		return encryptedMessage
	} else {
		encryptedMessage, err := protocol.NewSignalMessageFromBytes(rawMessage, sw.serializer.SignalMessage)
		if err != nil {
			logger.Error("Unable to restore message (without prekey) as JSON: ", err)
			panic("")
		}
		return encryptedMessage
	}

}
