package security

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

func TestVaultRoundTrip(t *testing.T) {
	t.Parallel()

	vaultPath := filepath.Join(t.TempDir(), "vault.json")
	vault, err := NewVault(vaultPath, "round-trip-pass")
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}

	secrets := map[string]string{
		"bot-token": "token-123",
		"llm-key":   "key-456",
		"db-pass":   "pass-789",
	}

	for key, value := range secrets {
		if err := vault.Set(key, value); err != nil {
			t.Fatalf("set %q: %v", key, err)
		}
	}

	reopened, err := NewVault(vaultPath, "round-trip-pass")
	if err != nil {
		t.Fatalf("reopen vault: %v", err)
	}

	for key, expected := range secrets {
		got, err := reopened.Get(key)
		if err != nil {
			t.Fatalf("get %q: %v", key, err)
		}
		if got != expected {
			t.Fatalf("get %q = %q, want %q", key, got, expected)
		}
	}
}

func TestVaultWrongPassphrase(t *testing.T) {
	t.Parallel()

	vaultPath := filepath.Join(t.TempDir(), "vault.json")
	vault, err := NewVault(vaultPath, "correct")
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}
	if err := vault.Set("secret", "value"); err != nil {
		t.Fatalf("set secret: %v", err)
	}

	if _, err := NewVault(vaultPath, "wrong"); err == nil {
		t.Fatal("expected error for wrong passphrase")
	}
}

func TestVaultListKeys(t *testing.T) {
	t.Parallel()

	vaultPath := filepath.Join(t.TempDir(), "vault.json")
	vault, err := NewVault(vaultPath, "list-pass")
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}

	if err := vault.Set("alpha", "value-a"); err != nil {
		t.Fatalf("set alpha: %v", err)
	}
	if err := vault.Set("charlie", "value-c"); err != nil {
		t.Fatalf("set charlie: %v", err)
	}
	if err := vault.Set("bravo", "value-b"); err != nil {
		t.Fatalf("set bravo: %v", err)
	}

	got := vault.List()
	sort.Strings(got)
	want := []string{"alpha", "bravo", "charlie"}

	if len(got) != len(want) {
		t.Fatalf("len(List()) = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("List()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestVaultDelete(t *testing.T) {
	t.Parallel()

	vaultPath := filepath.Join(t.TempDir(), "vault.json")
	vault, err := NewVault(vaultPath, "delete-pass")
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}

	if err := vault.Set("ephemeral", "value"); err != nil {
		t.Fatalf("set ephemeral: %v", err)
	}
	if err := vault.Delete("ephemeral"); err != nil {
		t.Fatalf("delete ephemeral: %v", err)
	}

	_, err = vault.Get("ephemeral")
	if !errors.Is(err, ErrVaultKeyNotFound) {
		t.Fatalf("Get after delete error = %v, want %v", err, ErrVaultKeyNotFound)
	}
}

func TestVaultFileOpaque(t *testing.T) {
	t.Parallel()

	vaultPath := filepath.Join(t.TempDir(), "vault.json")
	vault, err := NewVault(vaultPath, "opaque-pass")
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}

	secret := "my-super-secret-value"
	if err := vault.Set("secret", secret); err != nil {
		t.Fatalf("set secret: %v", err)
	}

	raw, err := os.ReadFile(vaultPath)
	if err != nil {
		t.Fatalf("read vault file: %v", err)
	}

	if bytes.Contains(raw, []byte(secret)) {
		t.Fatal("vault file contains plaintext secret")
	}
}

func TestVaultConcurrent(t *testing.T) {
	t.Parallel()

	vaultPath := filepath.Join(t.TempDir(), "vault.json")
	vault, err := NewVault(vaultPath, "concurrent-pass")
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}

	const workers = 10
	const iterations = 20

	var wg sync.WaitGroup
	errCh := make(chan error, workers*iterations*2)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := "k-" + string(rune('a'+worker))
				value := "v-" + string(rune('a'+worker))

				if err := vault.Set(key, value); err != nil {
					errCh <- err
					continue
				}

				got, err := vault.Get(key)
				if err != nil {
					errCh <- err
					continue
				}
				if got != value {
					errCh <- errors.New("mismatched value")
				}
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent operation failed: %v", err)
		}
	}
}

func TestVaultAutoCreateDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	vaultPath := filepath.Join(root, "does", "not", "exist", "vault.json")

	_, err := NewVault(vaultPath, "autocreate-pass")
	if err != nil {
		t.Fatalf("create vault: %v", err)
	}

	if _, err := os.Stat(filepath.Dir(vaultPath)); err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if _, err := os.Stat(vaultPath); err != nil {
		t.Fatalf("vault file not created: %v", err)
	}
}
