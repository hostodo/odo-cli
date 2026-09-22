package cmd

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/config"
)

const poolRetryCacheDir = "capacity-checkouts-v2"

var poolRetrySyncDir = syncPoolRetryDirPlatform

// Persist the quote, safe card ending, and encrypted generic confirmation. Never
// persist full responses or plaintext credentials, card IDs, or provider secrets.
type poolRetryRecord struct {
	Version      int                            `json:"version"`
	RequestHash  string                         `json:"request_hash"`
	Quote        *api.ResourcePoolExpectedQuote `json:"expected_quote"`
	CardLastFour string                         `json:"card_last_four,omitempty"`
	Confirmation string                         `json:"confirmation"`
	MAC          string                         `json:"hmac"`
}

type poolRetryCache struct {
	path         string
	requestHash  string
	secret       []byte
	confirmation string
	cardLastFour string
}

func poolStableIdentity(user *api.User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("current user identity is unavailable")
	}
	userID := user.UserID
	if userID <= 0 {
		userID = user.ID // Backward compatibility with older API responses.
	}
	if userID > 0 {
		return fmt.Sprintf("id:%d", userID), nil
	}
	if email := strings.ToLower(strings.TrimSpace(user.Email)); email != "" && user.IsEmailVerified {
		return "email:" + email, nil
	}
	return "", fmt.Errorf("current user is missing a stable identity")
}

func poolRetryHash(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func newPoolRetryCache(client *api.Client, req api.ResourcePoolCheckoutRequest) (*poolRetryCache, error) {
	path, err := config.GetConfigPath()
	if err != nil {
		return nil, err
	}
	cache := &poolRetryCache{path: filepath.Join(filepath.Dir(path), poolRetryCacheDir, poolRetryHash([]byte(req.IdempotencyKey))+".json")}
	secret, err := poolRetrySecret(filepath.Dir(path))
	if err != nil {
		return nil, cache.failure(fmt.Errorf("authentication key unavailable: %w", err))
	}
	cache.secret = secret
	user, err := client.GetCurrentUser()
	if err != nil {
		return nil, cache.failure(err)
	}
	identity, err := poolStableIdentity(user)
	if err != nil {
		return nil, cache.failure(err)
	}
	origin, err := url.Parse(client.BaseURL)
	if err != nil || origin.Host == "" || origin.User != nil || (origin.Scheme != "https" && origin.Scheme != "http") {
		return nil, cache.failure(fmt.Errorf("invalid API origin"))
	}
	endpoint := strings.ToLower(origin.Scheme + "://" + origin.Host)
	endpoint = strings.TrimSuffix(endpoint, map[string]string{"https": ":443", "http": ":80"}[origin.Scheme])
	// Bind the key to all checkout inputs and the authenticated endpoint without
	// writing any of those potentially sensitive inputs to disk. Token rotation
	// is safe for the same stable account identity and API origin.
	req.ExpectedQuote = nil
	req.Confirmation = ""
	req.PaymentConfirmation = ""
	req.ApprovedChargeAmount = ""
	data, err := json.Marshal(struct {
		Endpoint string
		Identity string
		Request  api.ResourcePoolCheckoutRequest
	}{endpoint, identity, req})
	if err != nil {
		return nil, err
	}
	cache.requestHash = cache.mac(data)
	return cache, nil
}

func (cache *poolRetryCache) mac(data []byte) string {
	mac := hmac.New(sha256.New, cache.secret)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

func (cache *poolRetryCache) recordMAC(record poolRetryRecord) string {
	record.MAC = ""
	data, _ := json.Marshal(record)
	return cache.mac(data)
}

func (cache *poolRetryCache) failure(err error) error {
	return fmt.Errorf("Capacity retry cache %s: %w; checkout aborted without fetching a replacement quote. Keep this record and reconcile any prior checkout before using a new key", cache.path, err)
}

func checkPoolRetryPath(path string, directory bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || info.IsDir() != directory || (!directory && !info.Mode().IsRegular()) {
		return fmt.Errorf("cache path must be a regular file or directory, without symlinks: %s", path)
	}
	return checkPoolRetryOwnership(path, info)
}

func checkPoolRetryPermissions(path string, directory bool) error {
	if err := checkPoolRetryPath(path, directory); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	return checkPoolRetryOwnerPermissions(path, info)
}

func (cache *poolRetryCache) load() (*api.ResourcePoolExpectedQuote, error) {
	if len(cache.secret) != 32 {
		return nil, cache.failure(fmt.Errorf("invalid retry authentication key"))
	}
	// A missing directory/file is first use. All other failures are fatal: a
	// damaged retry record must never turn into a fresh checkout.
	for i, dir := range []string{filepath.Dir(filepath.Dir(cache.path)), filepath.Dir(cache.path)} {
		check := checkPoolRetryPermissions
		if i == 0 {
			check = checkPoolRetryPath
		}
		if err := check(dir, true); err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, cache.failure(err)
		}
	}
	if err := checkPoolRetryPermissions(cache.path, false); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, cache.failure(err)
	}
	data, err := os.ReadFile(cache.path)
	if err != nil {
		return nil, cache.failure(err)
	}
	var record poolRetryRecord
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return nil, cache.failure(fmt.Errorf("corrupt record: %w", err))
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, cache.failure(fmt.Errorf("corrupt record: trailing data"))
	}
	var fields struct {
		Quote map[string]json.RawMessage `json:"expected_quote"`
	}
	if err := json.Unmarshal(data, &fields); err != nil || len(fields.Quote) != 5 {
		return nil, cache.failure(fmt.Errorf("corrupt record: incomplete expected_quote"))
	}
	if record.Version != 2 || record.Quote == nil || !hmac.Equal([]byte(record.MAC), []byte(cache.recordMAC(record))) {
		return nil, cache.failure(fmt.Errorf("corrupt or unsupported record"))
	}
	if record.RequestHash != cache.requestHash {
		return nil, cache.failure(fmt.Errorf("record does not match checkout arguments, API origin, or account; retry with the original inputs and account"))
	}
	if err := validatePoolRetryQuote(record.Quote); err != nil {
		return nil, cache.failure(err)
	}
	// Also complete durability if another process just published this record.
	if err := cache.syncDirs(false); err != nil {
		return nil, err
	}
	confirmation, err := cache.decryptConfirmation(record.Confirmation)
	if err != nil {
		return nil, cache.failure(err)
	}
	if err := validatePoolConfirmation(confirmation); err != nil {
		return nil, cache.failure(err)
	}
	cache.confirmation = confirmation
	cache.cardLastFour = record.CardLastFour
	return record.Quote, nil
}

func validatePoolRetryQuote(quote *api.ResourcePoolExpectedQuote) error {
	if quote.AmountDueAfterCredit.String() == "" {
		return fmt.Errorf("Capacity quote is missing amount_due_after_credit; purchase aborted")
	}
	if (quote.Mode != "purchase" && quote.Mode != "upgrade") || quote.UnitPrice.String() == "" || quote.RecurringAmount.String() == "" {
		return fmt.Errorf("Capacity quote is missing required snapshot fields; purchase aborted")
	}
	if _, err := json.Marshal(quote); err != nil {
		return fmt.Errorf("invalid Capacity quote: %w", err)
	}
	return nil
}

func (cache *poolRetryCache) save(quote *api.ResourcePoolExpectedQuote) error {
	if len(cache.secret) != 32 {
		return cache.failure(fmt.Errorf("invalid retry authentication key"))
	}
	if err := validatePoolRetryQuote(quote); err != nil {
		return err
	}
	if err := validatePoolConfirmation(cache.confirmation); err != nil {
		return cache.failure(err)
	}
	if _, err := ensurePoolRetryConfigDir(configDirForCache(cache.path)); err != nil {
		return cache.failure(err)
	}
	configDir, dir := filepath.Dir(filepath.Dir(cache.path)), filepath.Dir(cache.path)
	if err := checkPoolRetryPath(configDir, true); err != nil {
		return cache.failure(err)
	}
	cacheDirCreated := false
	if err := os.Mkdir(dir, 0700); err == nil {
		cacheDirCreated = true
	} else if !os.IsExist(err) {
		return cache.failure(err)
	}
	if err := checkPoolRetryPermissions(dir, true); err != nil {
		return cache.failure(err)
	}
	encrypted, err := cache.encryptConfirmation(cache.confirmation)
	if err != nil {
		return cache.failure(err)
	}
	record := poolRetryRecord{Version: 2, RequestHash: cache.requestHash, Quote: quote, CardLastFour: cache.cardLastFour, Confirmation: encrypted}
	record.MAC = cache.recordMAC(record)
	data, err := json.Marshal(record)
	if err != nil {
		return cache.failure(err)
	}
	file, err := os.CreateTemp(dir, ".quote-*") // 0600, including before publication
	if err != nil {
		return cache.failure(err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err := file.Write(data); err != nil {
		return cache.failure(err)
	}
	if err := file.Sync(); err != nil {
		return cache.failure(err)
	}
	if err := file.Close(); err != nil {
		return cache.failure(err)
	}
	// Publish atomically without overwriting a record from a concurrent process.
	// If the key appeared while confirming, abort and let a retry load its quote.
	if err := publishPoolRetryFile(file.Name(), cache.path); err != nil {
		return cache.failure(fmt.Errorf("could not save confirmed quote (if the key already exists, retry to load it): %w", err))
	}
	return cache.syncDirs(cacheDirCreated)
}

func configDirForCache(path string) string {
	return filepath.Dir(filepath.Dir(path))
}

func (cache *poolRetryCache) syncDirs(cacheDirCreated bool) error {
	dir := filepath.Dir(cache.path)
	configDir := filepath.Dir(dir)
	paths := []string{dir}
	if cacheDirCreated {
		paths = append([]string{configDir}, paths...)
	}
	for _, path := range paths {
		if err := poolRetrySyncDir(path); err != nil {
			return cache.failure(err)
		}
	}
	return nil
}

func poolRetryExistingID(id *string) string {
	if id == nil {
		return "none"
	}
	return *id
}
