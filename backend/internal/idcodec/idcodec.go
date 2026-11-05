package idcodec

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/capitalone/fpe/ff1"
	"golang.org/x/crypto/hkdf"
)

const (
	Alphabet    = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	Radix       = 36
	TotalLen    = 16
	PrefixWidth = 4
	IDWidth     = 10
)

// 36^10 - 1
const MaxID uint64 = 3656158440062975

// IDCodec provides Encrypt/Decrypt of 16-char tokens per README layout using FF1(AES)
type IDCodec struct {
	cipher ff1.Cipher
	tweak  []byte
}

// New creates a codec by deriving an AES-256 key from masterKey & tweak using HKDF-SHA256
// info: "zid.ff1.aes"
func New(masterKey, tweak []byte) (*IDCodec, error) {
	// Derive 32-byte key via HKDF-SHA256
	h := hkdf.New(sha256.New, masterKey, tweak, []byte("zid.ff1.aes"))
	var key [32]byte
	if _, err := io.ReadFull(h, key[:]); err != nil {
		return nil, fmt.Errorf("hkdf: %w", err)
	}
	c, err := ff1.NewCipher(Radix, TotalLen, key[:], tweak)
	if err != nil {
		return nil, fmt.Errorf("ff1: %w", err)
	}
	return &IDCodec{cipher: c, tweak: append([]byte(nil), tweak...)}, nil
}

// Encrypt returns a 16-char base36 token
func (c *IDCodec) Encrypt(prefix string, userID uint64) (string, error) {
	plain, err := buildPlain(prefix, userID)
	if err != nil {
		return "", err
	}
	// Normalize to lowercase for ff1 library (expects 0-9a-z)
	ctLower, err := c.cipher.Encrypt(strings.ToLower(plain))
	if err != nil {
		return "", err
	}
	return strings.ToUpper(ctLower), nil
}

// Decrypt returns (prefix, userID)
func (c *IDCodec) Decrypt(token string) (string, uint64, error) {
	if len(token) != TotalLen {
		return "", 0, errors.New("token must be 16 chars")
	}
	ptLower, err := c.cipher.Decrypt(strings.ToLower(token))
	if err != nil {
		return "", 0, err
	}
	// Parse with uppercase normalization to match checksum convention
	return parsePlain(strings.ToUpper(ptLower))
}

func buildPlain(prefix string, userID uint64) (string, error) {
	up := strings.ToUpper(prefix)
	if l := len(up); l < 1 || l > PrefixWidth {
		return "", errors.New("prefix length must be 1..4")
	}
	if userID == 0 || userID > MaxID {
		return "", fmt.Errorf("user_id must be in [1, %d]", MaxID)
	}
	lch := byte('0' + byte(len(up)))
	ppad := up + strings.Repeat("0", PrefixWidth-len(up))
	uid := fmtBase36(userID, IDWidth)
	p15 := string(lch) + ppad + uid
	c := checksumC(p15)
	return p15 + string(c), nil
}

func parsePlain(plain string) (string, uint64, error) {
	if len(plain) != TotalLen {
		return "", 0, errors.New("plain must be 16 chars")
	}
	if checksumC(plain[:15]) != plain[15] {
		return "", 0, errors.New("checksum mismatch")
	}
	L := int(plain[0] - '0')
	if L < 1 || L > PrefixWidth {
		return "", 0, errors.New("invalid prefix length nibble")
	}
	pp := plain[1 : 1+PrefixWidth]
	prefix := pp[:L]
	u := plain[1+PrefixWidth : 15]
	v, err := parseBase36(u)
	if err != nil {
		return "", 0, err
	}
	return prefix, v, nil
}

func fmtBase36(v uint64, width int) string {
	s := strings.ToUpper(strconv.FormatUint(v, Radix))
	if len(s) < width {
		return strings.Repeat("0", width-len(s)) + s
	}
	return s
}

func parseBase36(s string) (uint64, error) {
	var v uint64
	for _, ch := range s {
		p := strings.IndexRune(Alphabet, ch)
		if p < 0 {
			return 0, fmt.Errorf("invalid char: %c", ch)
		}
		v = v*36 + uint64(p)
	}
	return v, nil
}

func checksumC(prefix15 string) byte {
	h := sha256.Sum256([]byte(prefix15))
	idx := int(h[0]) % Radix
	return Alphabet[idx]
}
