package cmd

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hostodo/odo-cli/v2/pkg/config"
	"github.com/zalando/go-keyring"
)

const poolKeyService = "odo-cli"

// The key-source marker (or 0600 fallback secret) lives outside the record
// directory. Once selected, a missing/locked/corrupt key never generates a new
// key or switches storage. Old unauthenticated records cannot be migrated safely.
func poolRetrySecret(dir string) ([]byte, error) {
	if err := config.EnsureConfigDir(); err != nil {
		return nil, err
	}
	if err := checkPoolRetryPermissions(dir, true); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "capacity-retry-key")
	account := "capacity-retry-hmac-v2-" + poolRetryHash([]byte(dir))
	read := func() ([]byte, error) {
		if err := checkPoolRetryPermissions(path, false); err != nil {
			return nil, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		encoded := string(data)
		if encoded == "keyring" {
			encoded, err = keyring.Get(poolKeyService, account)
			if err != nil {
				return nil, fmt.Errorf("read OS keychain: %w", err)
			}
		} else if strings.HasPrefix(encoded, "fallback:") {
			encoded = strings.TrimPrefix(encoded, "fallback:")
		} else {
			return nil, fmt.Errorf("invalid key source")
		}
		return decodePoolSecret(encoded)
	}
	if _, err := os.Lstat(path); err == nil {
		return read()
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	lock, err := acquirePoolRetryKeyLock(path + ".init-lock")
	if err != nil {
		return nil, fmt.Errorf("key initialization lock: %w", err)
	}
	defer lock.Close() // Closing (including on process exit) releases the OS lock.
	// A competing initializer may have published the key while we waited.
	if _, err := os.Lstat(path); err == nil {
		return read()
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if _, err := os.Lstat(filepath.Join(dir, poolRetryCacheDir)); !os.IsNotExist(err) {
		return nil, fmt.Errorf("key source missing with existing retry cache; recover the original key")
	}
	encoded, getErr := keyring.Get(poolKeyService, account)
	source := "keyring"
	if getErr != nil {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, err
		}
		encoded = base64.StdEncoding.EncodeToString(secret)
		// Only initialize a keychain entry when it is known to be absent. An
		// unavailable keychain on first use selects the durable file fallback.
		if getErr != keyring.ErrNotFound {
			source = "fallback:" + encoded
		} else if err := keyring.Set(poolKeyService, account, encoded); err != nil {
			source = "fallback:" + encoded
		} else {
			check, err := keyring.Get(poolKeyService, account)
			if err != nil || check != encoded {
				return nil, fmt.Errorf("OS keychain write verification failed")
			}
		}
	}
	secret, err := decodePoolSecret(encoded)
	if err != nil {
		return nil, err
	}
	file, err := os.CreateTemp(dir, ".capacity-key-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err := file.WriteString(source); err != nil {
		return nil, err
	}
	if err := file.Sync(); err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	if err := publishPoolRetryFile(file.Name(), path); err != nil {
		return nil, err
	}
	if err := syncPoolRetryDir(dir); err != nil {
		return nil, err
	}
	if err := syncPoolRetryDir(filepath.Dir(dir)); err != nil {
		return nil, err
	}
	return secret, nil
}

// Keep this file permanently: unlinking it could let waiters lock different
// inodes. The old mkdir-based .lock artifact is deliberately left untouched.
func acquirePoolRetryKeyLock(path string) (*os.File, error) {
	if err := checkPoolRetryPermissions(path, false); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*os.File, error) {
		file.Close()
		return nil, err
	}
	if err := checkPoolRetryPermissions(path, false); err != nil {
		return fail(err)
	}
	opened, err := file.Stat()
	if err != nil {
		return fail(err)
	}
	current, err := os.Lstat(path)
	if err != nil {
		return fail(err)
	}
	if !os.SameFile(opened, current) {
		return fail(fmt.Errorf("key initialization lock file changed"))
	}
	if err := lockPoolRetryFile(file); err != nil {
		return fail(err)
	}
	return file, nil
}

func decodePoolSecret(encoded string) ([]byte, error) {
	secret, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(secret) != 32 {
		return nil, fmt.Errorf("invalid retry authentication key (expected 32 bytes)")
	}
	return secret, nil
}

// Generic phrases are opaque server text and could include sensitive inputs.
// Encrypt them at rest rather than persisting card IDs or provider text in clear.
// The record HMAC authenticates this ciphertext and every other retry field.
func (cache *poolRetryCache) confirmationCipher() (cipher.AEAD, error) {
	if len(cache.secret) != 32 {
		return nil, fmt.Errorf("invalid retry authentication key")
	}
	derivation := hmac.New(sha256.New, cache.secret)
	derivation.Write([]byte("odo Capacity confirmation encryption v2"))
	block, err := aes.NewCipher(derivation.Sum(nil))
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (cache *poolRetryCache) encryptConfirmation(phrase string) (string, error) {
	aead, err := cache.confirmationCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	data := aead.Seal(nonce, nonce, []byte(phrase), []byte(cache.requestHash))
	return base64.StdEncoding.EncodeToString(data), nil
}

func (cache *poolRetryCache) decryptConfirmation(encoded string) (string, error) {
	aead, err := cache.confirmationCipher()
	if err != nil {
		return "", err
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(data) < aead.NonceSize() {
		return "", fmt.Errorf("invalid cached confirmation")
	}
	plain, err := aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], []byte(cache.requestHash))
	if err != nil {
		return "", fmt.Errorf("cached confirmation authentication failed")
	}
	return string(plain), nil
}
