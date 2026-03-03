package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"golang.org/x/crypto/argon2"
)

const (
	vaultSaltSize  = 16
	vaultNonceSize = 12
	vaultKeyLen    = 32
	vaultTime      = 1
	vaultMemory    = 64 * 1024
	vaultThreads   = 4
)

var (
	ErrVaultLocked      = errors.New("vault is locked")
	ErrVaultKeyNotFound = errors.New("vault key not found")
)

type Vault struct {
	path string
	key  []byte
	mu   sync.RWMutex
	data map[string]string
}

type vaultDisk struct {
	Salt  string `json:"salt"`
	Nonce string `json:"nonce"`
	Data  string `json:"data"`
}

func NewVault(path, passphrase string) (*Vault, error) {
	expandedPath, err := expandPath(path)
	if err != nil {
		return nil, err
	}

	passphraseBytes := []byte(passphrase)
	defer zeroBytes(passphraseBytes)

	v := &Vault{
		path: expandedPath,
		data: make(map[string]string),
	}

	_, statErr := os.Stat(v.path)
	if statErr == nil {
		salt, err := readVaultSalt(v.path)
		if err != nil {
			return nil, err
		}
		v.key = deriveKey(passphraseBytes, salt)
		if err := v.load(); err != nil {
			zeroBytes(v.key)
			v.key = nil
			return nil, err
		}
		return v, nil
	}

	if !errors.Is(statErr, os.ErrNotExist) {
		return nil, statErr
	}

	if err := os.MkdirAll(filepath.Dir(v.path), 0o700); err != nil {
		return nil, err
	}

	salt := make([]byte, vaultSaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	v.key = deriveKey(passphraseBytes, salt)
	if err := v.saveWithSalt(salt); err != nil {
		zeroBytes(v.key)
		v.key = nil
		return nil, err
	}

	return v, nil
}

func (v *Vault) Get(key string) (string, error) {
	if !v.isUnlocked() {
		return "", ErrVaultLocked
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	value, ok := v.data[key]
	if !ok {
		return "", ErrVaultKeyNotFound
	}

	return value, nil
}

func (v *Vault) Set(key, value string) error {
	if !v.isUnlocked() {
		return ErrVaultLocked
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	v.data[key] = value
	return v.save()
}

func (v *Vault) Delete(key string) error {
	if !v.isUnlocked() {
		return ErrVaultLocked
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	delete(v.data, key)
	return v.save()
}

func (v *Vault) List() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	keys := make([]string, 0, len(v.data))
	for key := range v.data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (v *Vault) save() error {
	if !v.isUnlocked() {
		return ErrVaultLocked
	}

	salt, err := readVaultSalt(v.path)
	if err != nil {
		return err
	}

	return v.saveWithSalt(salt)
}

func (v *Vault) saveWithSalt(salt []byte) error {
	if err := os.MkdirAll(filepath.Dir(v.path), 0o700); err != nil {
		return err
	}

	plaintext, err := json.Marshal(v.data)
	if err != nil {
		return err
	}

	blk, err := aes.NewCipher(v.key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(blk)
	if err != nil {
		return err
	}

	nonce := make([]byte, vaultNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	diskData := vaultDisk{
		Salt:  base64.StdEncoding.EncodeToString(salt),
		Nonce: base64.StdEncoding.EncodeToString(nonce),
		Data:  base64.StdEncoding.EncodeToString(ciphertext),
	}

	encoded, err := json.Marshal(diskData)
	if err != nil {
		return err
	}

	tmpPath := v.path + ".tmp"
	if err := os.WriteFile(tmpPath, encoded, 0o600); err != nil {
		return err
	}

	return os.Rename(tmpPath, v.path)
}

func (v *Vault) load() error {
	if !v.isUnlocked() {
		return ErrVaultLocked
	}

	raw, err := os.ReadFile(v.path)
	if err != nil {
		return err
	}

	var diskData vaultDisk
	if err := json.Unmarshal(raw, &diskData); err != nil {
		return err
	}

	nonce, err := base64.StdEncoding.DecodeString(diskData.Nonce)
	if err != nil {
		return err
	}
	if len(nonce) != vaultNonceSize {
		return fmt.Errorf("invalid nonce size: %d", len(nonce))
	}

	ciphertext, err := base64.StdEncoding.DecodeString(diskData.Data)
	if err != nil {
		return err
	}

	blk, err := aes.NewCipher(v.key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(blk)
	if err != nil {
		return err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}

	decryptedData := make(map[string]string)
	if len(plaintext) > 0 {
		if err := json.Unmarshal(plaintext, &decryptedData); err != nil {
			return err
		}
	}

	v.mu.Lock()
	v.data = decryptedData
	v.mu.Unlock()

	return nil
}

func (v *Vault) isUnlocked() bool {
	return v != nil && len(v.key) == vaultKeyLen
}

func deriveKey(passphrase, salt []byte) []byte {
	return argon2.IDKey(passphrase, salt, vaultTime, vaultMemory, vaultThreads, vaultKeyLen)
}

func readVaultSalt(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var diskData vaultDisk
	if err := json.Unmarshal(raw, &diskData); err != nil {
		return nil, err
	}

	salt, err := base64.StdEncoding.DecodeString(diskData.Salt)
	if err != nil {
		return nil, err
	}
	if len(salt) != vaultSaltSize {
		return nil, fmt.Errorf("invalid salt size: %d", len(salt))
	}

	return salt, nil
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func expandPath(path string) (string, error) {
	if path == "" {
		return "", errors.New("vault path is empty")
	}

	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if len(path) == 1 {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}

	return path, nil
}
