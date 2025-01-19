package util

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	syntax "github.com/cisco/go-tls-syntax"
	"github.com/s3131212/go-mls"
	"io"
	"math/big"
)

/*
CipherText is the ciphertext struct that contains the IV and the ciphertext.
*/
type CipherText struct {
	IV         []byte
	Signature  []byte
	CipherText []byte
}

/*
Encrypt encrypts the plaintext with the given key using AES-256 GCM
*/
func Encrypt(plaintext []byte, key []byte, signPrivKey []byte) (CipherText, error) {
	// Create new AES cipher
	aesCipher, err := aes.NewCipher(key)
	if err != nil {
		return CipherText{}, err
	}

	// Create new GCM cipher
	aesGCM, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return CipherText{}, err
	}

	// Create new IV
	iv := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return CipherText{}, err
	}

	// Encrypt the plaintext
	ciphertext := aesGCM.Seal(nil, iv, plaintext, nil)

	// Signature
	var sig []byte
	if signPrivKey != nil {
		ecdhKey, err := ecdh.P256().NewPrivateKey(signPrivKey)
		if err != nil {
			return CipherText{}, err
		}
		ecdsaKey := ecdhToECDSAPrivateKey(ecdhKey)
		hash := sha256.Sum256(ciphertext)
		sig, err = ecdsa.SignASN1(rand.Reader, ecdsaKey, hash[:])
		if err != nil {
			return CipherText{}, err
		}
	}

	return CipherText{
		IV:         iv,
		CipherText: ciphertext,
		Signature:  sig,
	}, nil
}

/*
Decrypt decrypts the ciphertext with the given key using AES-256 GCM
*/
func Decrypt(ciphertext CipherText, key []byte, signPubKey []byte) ([]byte, error) {
	if len(ciphertext.Signature) == 0 && signPubKey != nil {
		return nil, errors.New("signature is missing")
	}

	if len(ciphertext.Signature) != 0 && signPubKey == nil {
		return nil, errors.New("public key is missing")
	}

	if signPubKey != nil {
		ecdhKey, err := ecdh.P256().NewPublicKey(signPubKey)
		if err != nil {
			return nil, err
		}
		ecdsaPubKey := ecdhToECDSAPublicKey(ecdhKey)
		hash := sha256.Sum256(ciphertext.CipherText)
		valid := ecdsa.VerifyASN1(ecdsaPubKey, hash[:], ciphertext.Signature)

		if !valid {
			return nil, errors.New("signature is not valid")
		}
	}

	// Create new AES cipher
	aesCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create new GCM cipher
	aesGCM, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return nil, err
	}

	// Decrypt the ciphertext
	plaintext, err := aesGCM.Open(nil, ciphertext.IV, ciphertext.CipherText, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

/*
Sign signs the message with the given private key
*/
func Sign(message []byte, signPrivKey []byte) ([]byte, error) {
	var sig []byte

	ecdhKey, err := ecdh.P256().NewPrivateKey(signPrivKey)
	if err != nil {
		return nil, err
	}
	ecdsaKey := ecdhToECDSAPrivateKey(ecdhKey)
	hash := sha256.Sum256(message)
	sig, err = ecdsa.SignASN1(rand.Reader, ecdsaKey, hash[:])
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func (ct CipherText) Serialize() []byte {
	sig := make([]byte, 73)
	sigLen := len(ct.Signature)
	// Pad with zeros on the left to make signature 72 bytes
	sig[0] = byte(72 - sigLen)
	copy(sig[73-sigLen:], ct.Signature)

	// Create copies of IV and CipherText to avoid modifying the original array
	ivCopy := make([]byte, len(ct.IV))
	copy(ivCopy, ct.IV)

	cipherTextCopy := make([]byte, len(ct.CipherText))
	copy(cipherTextCopy, ct.CipherText)

	// Append using the copied slices
	serialized := append(append(ivCopy, sig...), cipherTextCopy...)
	return serialized
}

func DeserializeCipherText(data []byte) CipherText {
	sig := data[12 : 12+73]
	sig = sig[1+int(sig[0]):]
	return CipherText{
		IV:         data[:12],
		Signature:  sig,
		CipherText: data[12+73:],
	}
}

// https://github.com/golang/go/issues/63963
func ecdhToECDSAPublicKey(key *ecdh.PublicKey) *ecdsa.PublicKey {
	rawKey := key.Bytes()
	switch key.Curve() {
	case ecdh.P256():
		return &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     big.NewInt(0).SetBytes(rawKey[1:33]),
			Y:     big.NewInt(0).SetBytes(rawKey[33:]),
		}
	case ecdh.P384():
		return &ecdsa.PublicKey{
			Curve: elliptic.P384(),
			X:     big.NewInt(0).SetBytes(rawKey[1:49]),
			Y:     big.NewInt(0).SetBytes(rawKey[49:]),
		}
	case ecdh.P521():
		return &ecdsa.PublicKey{
			Curve: elliptic.P521(),
			X:     big.NewInt(0).SetBytes(rawKey[1:67]),
			Y:     big.NewInt(0).SetBytes(rawKey[67:]),
		}
	default:
		panic("cannot convert non-NIST *ecdh.PublicKey to *ecdsa.PublicKey")
	}
}

func ecdhToECDSAPrivateKey(priv *ecdh.PrivateKey) *ecdsa.PrivateKey {
	return &ecdsa.PrivateKey{
		PublicKey: *ecdhToECDSAPublicKey(priv.PublicKey()),
		D:         big.NewInt(0).SetBytes(priv.Bytes()),
	}
}

// MLS KeyPackage Serialization
func SerializeMLSKeyPackage(kp mls.KeyPackage) ([]byte, error) {
	return syntax.Marshal(struct {
		Version     mls.ProtocolVersion
		CipherSuite mls.CipherSuite
		InitKey     mls.HPKEPublicKey
		Credential  mls.Credential
		Extensions  mls.ExtensionList
		Signature   mls.Signature
	}{
		Version:     kp.Version,
		CipherSuite: kp.CipherSuite,
		InitKey:     kp.InitKey,
		Credential:  kp.Credential,
		Extensions:  kp.Extensions,
		Signature:   kp.Signature,
	})
}

// MLS KeyPackage Deserialization
func DeserializeMLSKeyPackage(data []byte) (mls.KeyPackage, error) {
	var kp struct {
		Version     mls.ProtocolVersion
		CipherSuite mls.CipherSuite
		InitKey     mls.HPKEPublicKey
		Credential  mls.Credential
		Extensions  mls.ExtensionList
		Signature   mls.Signature
	}
	if _, err := syntax.Unmarshal(data, &kp); err != nil {
		return mls.KeyPackage{}, err
	}

	return mls.KeyPackage{
		Version:     kp.Version,
		CipherSuite: kp.CipherSuite,
		InitKey:     kp.InitKey,
		Credential:  kp.Credential,
		Extensions:  kp.Extensions,
		Signature:   kp.Signature,
	}, nil
}

// MLSCiphertext Serialization
func SerializeMLSCiphertext(ct mls.MLSCiphertext) ([]byte, error) {
	return syntax.Marshal(ct)
}

// MLSCiphertext Deserialization
func DeserializeMLSCiphertext(data []byte) (mls.MLSCiphertext, error) {
	var ct mls.MLSCiphertext
	if _, err := syntax.Unmarshal(data, &ct); err != nil {
		return mls.MLSCiphertext{}, err
	}

	return ct, nil
}
