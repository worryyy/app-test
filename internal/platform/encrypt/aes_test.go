package encrypt

import "testing"

const (
	testAESKey        = "W0F7PePvolUJHmZtkv1MusWpwhpVJIJI"
	testAESPlaintext  = "test123"
	testAESCiphertext = "B9gMpgfaS/4RIi72YJa+tA=="
)

func TestAESEncryptMatchesJava(t *testing.T) {
	got, err := AESEncrypt(testAESPlaintext, testAESKey)
	if err != nil {
		t.Fatalf("AESEncrypt returned error: %v", err)
	}
	if got != testAESCiphertext {
		t.Fatalf("AESEncrypt mismatch: got %q want %q", got, testAESCiphertext)
	}
}

func TestAESDecryptMatchesJava(t *testing.T) {
	got, err := AESDecrypt(testAESCiphertext, testAESKey)
	if err != nil {
		t.Fatalf("AESDecrypt returned error: %v", err)
	}
	if got != testAESPlaintext {
		t.Fatalf("AESDecrypt mismatch: got %q want %q", got, testAESPlaintext)
	}
}
