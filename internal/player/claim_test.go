package player

import "testing"

func TestGenerateClaimCodeFormat(t *testing.T) {
	code, err := GenerateClaimCode()
	if err != nil {
		t.Fatal(err)
	}
	if !ValidClaimCode(code) {
		t.Fatalf("generated code not valid: %q", code)
	}
	display := FormatClaimCode(code)
	if len(display) != 14 { // XXXX-XXXX-XXXX
		t.Fatalf("display len = %d, want 14: %q", len(display), display)
	}
	parsed, err := ParseClaimCode(display)
	if err != nil {
		t.Fatal(err)
	}
	if parsed != NormalizeClaimCode(code) {
		t.Fatalf("parse round-trip: got %q want %q", parsed, code)
	}
}

func TestClaimCodeHashVerify(t *testing.T) {
	code := "ABCD234567EF"
	hash, err := HashClaimCode(code)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" || hash == code {
		t.Fatal("hash should not equal plaintext")
	}
	if !VerifyClaimCode(code, hash) {
		t.Fatal("expected verify ok")
	}
	if VerifyClaimCode("ABCD234567EG", hash) {
		t.Fatal("wrong code should fail")
	}
	// Crockford: I/L/O/U forbidden; 0/1 map from O/I visually — accept lowercase
	if !VerifyClaimCode("abcd234567ef", hash) {
		t.Fatal("case-insensitive verify")
	}
}

func TestParseClaimCodeRejectsBad(t *testing.T) {
	for _, in := range []string{"", "short", "XXXX-XXXX-XXX!", "UUUUUUUUUUUU"} {
		if _, err := ParseClaimCode(in); err == nil {
			t.Fatalf("expected error for %q", in)
		}
	}
}
