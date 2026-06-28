package authcrypto

import "testing"

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !VerifyPassword("correct horse battery staple", hash) {
		t.Fatal("correct password rejected")
	}
	if VerifyPassword("wrong password", hash) {
		t.Fatal("wrong password accepted")
	}
}

func TestHashIsSalted(t *testing.T) {
	a, err := HashPassword("same input")
	if err != nil {
		t.Fatalf("hash a: %v", err)
	}
	b, err := HashPassword("same input")
	if err != nil {
		t.Fatalf("hash b: %v", err)
	}
	if a == b {
		t.Fatal("identical encoded hashes for same password — salt is missing")
	}
}

func TestVerifyRejectsMalformed(t *testing.T) {
	for _, bad := range []string{"", "notahash", "$argon2id$broken", "$2y$bcrypt$nope"} {
		if VerifyPassword("anything", bad) {
			t.Fatalf("malformed hash accepted: %q", bad)
		}
	}
}

func TestTokenHashDeterministicAndUnique(t *testing.T) {
	token, err := NewToken()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if HashToken(token) != HashToken(token) {
		t.Fatal("token hash is not deterministic")
	}
	other, err := NewToken()
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	if token == other {
		t.Fatal("NewToken produced a duplicate token")
	}
	if HashToken(token) == HashToken(other) {
		t.Fatal("distinct tokens hashed to the same value")
	}
}
