package crypto

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
)

var (
	ErrHmacMismatch     = errors.New("Integrity check failed: HMAC does not match decrypted data")
	ErrUnknowThumbprint = errors.New("No private key found for the given certificate thumbprint")
)

// KeyPair bundles a private key with the SHA-256 thumbprint of its
// corresponding certificate -- used to select the right key during
// rotation, when multiple keys may be valid simultaneously.
type KeyPair struct {
	PrivateKey *rsa.PrivateKey
	Thumbprint string // base64 SHA-256 hash of the DER-encoded certificate
}

// Decrypter holds all currently-valid key pairs, keyed by thumbprint,
// so a request encrypted under an older (not-yet-expired) certificate
// during a rotation window can still be decrypted correctly.
type Decrypter struct {
	keysByThumbprint map[string]*rsa.PrivateKey
}

func NewDecrypter(keyPairs ...KeyPair) *Decrypter {
	m := make(map[string]*rsa.PrivateKey, len(keyPairs))
	for _, kp := range keyPairs {
		m[kp.Thumbprint] = kp.PrivateKey
	}
	return &Decrypter{keysByThumbprint: m}
}

func (d *Decrypter) Decrypt(payload EncryptedPayload) ([]byte, error) {
	privateKey, ok := d.keysByThumbprint[payload.Thumbprint]
	if !ok {
		return nil, ErrUnknowThumbprint
	}

	sessionKey, err := decryptRSASessionKey(payload.EncryptedSessionKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt session key: %w", err)
	}

	ivBytes, err := base64.StdEncoding.DecodeString(payload.IV)
	if err != nil {
		return nil, fmt.Errorf("failed to decode IV: %w", err)
	}

	plaintext, err := decryptAESGCM(payload.EncryptedData, sessionKey, ivBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt AES-GCM data: %w", err)
	}

	hmacBytes, err := base64.StdEncoding.DecodeString(payload.Hmac)
	if err != nil {
		return nil, fmt.Errorf("failed to decode HMAC: %w", err)
	}

	decryptedHash, err := decryptAESGCM(hmacBytes, sessionKey, ivBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt HMAC: %w", err)
	}

	actualHash := sha256.Sum256(plaintext)

	if !hmacEqual(actualHash[:], decryptedHash) {
		return nil, ErrHmacMismatch
	}

	return plaintext, nil
}

// Go's rsa.DecryptOAEP with sha256.New for both the hash and MGF1 hash
func decryptRSASessionKey(encryptedSessionKeyB64 string, privateKey *rsa.PrivateKey) ([]byte, error) {
	encrypted, err := base64.StdEncoding.DecodeString(encryptedSessionKeyB64)

	if err != nil {
		return nil, fmt.Errorf("Invalid Sesssion Key Encoding: %w", err)
	}
	hash := sha256.New()

	sessionKey, err := rsa.DecryptOAEP(hash, nil, privateKey, encrypted, nil)

	if err != nil {
		return nil, err
	}

	return sessionKey, nil
}

func decryptAESGCM(ciphertext, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := aesgcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func hmacEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

type EncryptedPayload struct {
	EncryptedData       []byte // raw AES-GCM ciphertext of the image
	EncryptedSessionKey string // base64, RSA-OAEP-SHA256 encrypted AES-256 key
	IV                  string // base64, AES-GCM nonce 12 bytes
	Hmac                string // base64, AES-GCM encrypted SHA-256 hash of plaintext
	Thumbprint          string // base64, SHA-256 hash of the certificate used
}

// CertificateThumbprint computes the base64 SHA-256 hash of a DER-encoded
func CertificateThumbprint(certDER []byte) string {
	sum := sha256.Sum256(certDER)
	return base64.StdEncoding.EncodeToString(sum[:])
}

func LoadKeyPairFromPEM(privateKeyPEM, certPEM []byte) (KeyPair, error) {
	keyBlock, _ := pem.Decode(privateKeyPEM)
	if keyBlock == nil {
		return KeyPair{}, errors.New("failed to decode private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)

	if err != nil {
		// fall back to PKCS8 in case the key was generated in that format
		keyAny, err2 := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err2 != nil {
			return KeyPair{}, errors.New("failed to parse private key: " + err.Error() + "; " + err2.Error())
		}

		var ok bool
		privateKey, ok = keyAny.(*rsa.PrivateKey)
		if !ok {
			return KeyPair{}, errors.New("private key is not an RSA key")
		}
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return KeyPair{}, errors.New("failed to decode certificate PEM")
	}

	return KeyPair{
		PrivateKey: privateKey,
		Thumbprint: CertificateThumbprint(certBlock.Bytes),
	}, nil
}

var _ = crypto.SHA256
