package mailqueue

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"

	"etheyl.local/backend/internal/config"
)

// Cipher encrypts outbox bodies with the first key; the remaining keys support rotation.
// Keys are base64-encoded 32-byte secrets, held outside the database and its backups.
type Cipher struct {
	active string
	keys   map[string]cipher.AEAD
}

func NewCipher(encoded string) (*Cipher, error) {
	c := &Cipher{keys: make(map[string]cipher.AEAD)}
	for _, value := range strings.Split(encoded, ",") {
		key, err := base64.StdEncoding.DecodeString(value)
		if err != nil || len(key) != 32 {
			return nil, &config.Error{Message: "OUTBOX_KEYS must contain base64-encoded 32-byte keys"}
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		aead, err := cipher.NewGCMWithRandomNonce(block)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(key)
		id := hex.EncodeToString(digest[:8])
		if c.active == "" {
			c.active = id
		}
		c.keys[id] = aead
	}
	return c, nil
}
func (c *Cipher) Seal(body, metadata string) string {
	data := c.keys[c.active].Seal(nil, nil, []byte(body), []byte(metadata))
	return "v1." + c.active + "." + base64.RawStdEncoding.EncodeToString(data)
}
func (c *Cipher) Open(body, metadata string) (string, error) {
	parts := strings.Split(body, ".")
	if len(parts) != 3 || parts[0] != "v1" {
		return "", errors.New("invalid outbox envelope")
	}
	aead := c.keys[parts[1]]
	if aead == nil {
		return "", errors.New("outbox key unavailable")
	}
	data, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return "", errors.New("invalid outbox ciphertext")
	}
	plain, err := aead.Open(nil, nil, data, []byte(metadata))
	if err != nil {
		return "", errors.New("outbox authentication failed")
	}
	return string(plain), nil
}
func Metadata(recipient, subject, tokenHash string) string {
	return recipient + "\x00" + subject + "\x00" + tokenHash
}
