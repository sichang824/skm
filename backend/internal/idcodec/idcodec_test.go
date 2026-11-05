package idcodec

import (
	"testing"
)

func codec(t *testing.T) *IDCodec {
	t.Helper()
	key := []byte("prod-key-example-1234567890")
	tw := []byte("v1|zhiyu|wxgobk")
	c, err := New(key, tw)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return c
}

func TestRoundtrip(t *testing.T) {
	c := codec(t)
	cases := []struct {
		p  string
		id uint64
	}{
		{"Z", 1},
		{"ZH", 36},
		{"ZHI", 123456},
		{"ZHIY", 9876543210},
	}
	for _, cs := range cases {
		tok, err := c.Encrypt(cs.p, cs.id)
		if err != nil {
			t.Fatalf("enc: %v", err)
		}
		if len(tok) != TotalLen {
			t.Fatalf("len=%d", len(tok))
		}
		p2, id2, err := c.Decrypt(tok)
		if err != nil {
			t.Fatalf("dec: %v", err)
		}
		if p2 != cs.p || id2 != cs.id {
			t.Fatalf("got %s %d", p2, id2)
		}
	}
}

func TestBounds(t *testing.T) {
	c := codec(t)
	if _, err := c.Encrypt("", 1); err == nil {
		t.Fatal("want err")
	}
	if _, err := c.Encrypt("ABCDE", 1); err == nil {
		t.Fatal("want err")
	}
	if _, err := c.Encrypt("A", 0); err == nil {
		t.Fatal("want err")
	}
	if _, err := c.Encrypt("A", MaxID+1); err == nil {
		t.Fatal("want err")
	}
}

// Golden values from the previous external codec protect existing databases:
// changing an entity's ZID would break links and cache/audit references.
func TestExistingDatabaseIDsRemainCompatible(t *testing.T) {
	c, err := New([]byte("default-dev-key-change-in-production-32bytes"), []byte("v1|app|dev"))
	if err != nil {
		t.Fatal(err)
	}
	for prefix, token := range map[string]string{
		"PROV": "ZQYL05PCZPNQGIG3",
		"SKIL": "D9FDRRLL66TGLPWC",
		"EXEC": "OTYOYX5O1E92YV4U",
	} {
		got, err := c.Encrypt(prefix, 1)
		if err != nil || got != token {
			t.Fatalf("%s: got %q, %v; want %q", prefix, got, err, token)
		}
		p, id, err := c.Decrypt(token)
		if err != nil || p != prefix || id != 1 {
			t.Fatalf("decode existing ID: %q %d %v", p, id, err)
		}
	}
}
