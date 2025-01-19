package treekem

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"io"
)

type Keypair struct {
	Private *ecdh.PrivateKey
	Public  *ecdh.PublicKey
}

func NewKeyPair() (*Keypair, error) {
	privateKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	publicKey := privateKey.PublicKey()

	return &Keypair{
		Private: privateKey,
		Public:  publicKey,
	}, nil
}

func NewKeyPairFromSecret(secret []byte) (*Keypair, error) {
	// Calculate SHA-256 of the secret
	digest := sha256.Sum256(secret)

	privateKey, err := ecdh.P256().NewPrivateKey(digest[:])
	if err != nil {
		return nil, err
	}

	publicKey := privateKey.PublicKey()
	return &Keypair{
		Private: privateKey,
		Public:  publicKey,
	}, nil
}

func NewSigningKeyPairFromSecret(secret []byte) (*Keypair, error) {
	return NewKeyPairFromSecret(append([]byte("signing-"), secret...))
}

func SecretFromBytes(privateKey []byte, publicKey []byte) ([]byte, error) {
	sk, err := ecdh.P256().NewPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	pk, err := ecdh.P256().NewPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	return sk.ECDH(pk)
}

type ECKEMCipherText struct {
	Public     []byte
	IV         []byte
	CipherText []byte
}

func ECKEMEncrypt(value []byte, publicKey []byte) (ECKEMCipherText, error) {
	//return ECKEMCipherText{
	//	Public:     nil,
	//	IV:         nil,
	//	CipherText: value,
	//}, nil

	kpE, err := NewKeyPair()
	if err != nil {
		return ECKEMCipherText{}, err
	}
	ek, err := SecretFromBytes(kpE.Private.Bytes(), publicKey)
	if err != nil {
		return ECKEMCipherText{}, err
	}

	// AES GCM
	block, err := aes.NewCipher(ek)
	if err != nil {
		return ECKEMCipherText{}, err
	}

	iv := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return ECKEMCipherText{}, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return ECKEMCipherText{}, err
	}

	cipherText := aesgcm.Seal(nil, iv, value, nil)

	return ECKEMCipherText{
		Public:     kpE.Public.Bytes(),
		IV:         iv,
		CipherText: cipherText,
	}, nil
}

func ECKEMDecrypt(ciphertext ECKEMCipherText, privateKey []byte) ([]byte, error) {
	//return ciphertext.CipherText, nil
	ek, err := SecretFromBytes(privateKey, ciphertext.Public)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(ek)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := aesgcm.Open(nil, ciphertext.IV, ciphertext.CipherText, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// GenerateRandomBytes returns securely generated random bytes.
func GenerateRandomBytes(n int) ([]byte, error) {
	token := make([]byte, n)
	_, err := rand.Read(token)
	return token, err
}
