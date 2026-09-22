package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/auth"
	"github.com/hostodo/odo-cli/v2/pkg/config"
)

// Persist only the quote fields sent back to the API. In particular, never
// persist checkout responses, hosted URLs, credentials or saved-card IDs.
type poolRetryRecord struct {
	Version     int                            `json:"version"`
	RequestHash string                         `json:"request_hash"`
	Quote       *api.ResourcePoolExpectedQuote `json:"expected_quote"`
	Checksum    string                         `json:"checksum"`
}

type poolRetryCache struct {
	path        string
	requestHash string
}

func poolRetryHash(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func newPoolRetryCache(client *api.Client, req api.ResourcePoolCheckoutRequest) (*poolRetryCache, error) {
	path, err := config.GetConfigPath()
	if err != nil {
		return nil, err
	}
	token, err := auth.GetToken()
	if err != nil {
		return nil, api.ErrNotAuthenticated
	}
	// Bind the key to all checkout inputs and the authenticated endpoint without
	// writing any of those potentially sensitive inputs to disk. A changed login
	// fails closed rather than risking a new purchase under another account.
	req.ExpectedQuote = nil
	data, err := json.Marshal(struct {
		Endpoint string
		Token    string
		Request  api.ResourcePoolCheckoutRequest
	}{client.BaseURL, token, req})
	if err != nil {
		return nil, err
	}
	return &poolRetryCache{
		path:        filepath.Join(filepath.Dir(path), "capacity-checkouts", poolRetryHash([]byte(req.IdempotencyKey))+".json"),
		requestHash: poolRetryHash(data),
	}, nil
}

func poolRetryChecksum(record poolRetryRecord) string {
	record.Checksum = ""
	data, _ := json.Marshal(record)
	return poolRetryHash(data)
}

func (cache *poolRetryCache) failure(err error) error {
	return fmt.Errorf("Capacity retry cache %s: %w; checkout aborted without fetching a replacement quote. Keep this record and reconcile any prior checkout before using a new key", cache.path, err)
}

func checkPoolRetryPermissions(path string, directory bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || info.IsDir() != directory || (!directory && !info.Mode().IsRegular()) {
		return fmt.Errorf("cache path must be a regular file or directory, without symlinks: %s", path)
	}
	if info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("insecure permissions on %s (require owner-only access)", path)
	}
	return nil
}

func syncPoolRetryDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func (cache *poolRetryCache) load() (*api.ResourcePoolExpectedQuote, error) {
	// A missing directory/file is first use. All other failures are fatal: a
	// damaged retry record must never turn into a fresh checkout.
	for _, dir := range []string{filepath.Dir(filepath.Dir(cache.path)), filepath.Dir(cache.path)} {
		if err := checkPoolRetryPermissions(dir, true); err != nil {
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
	if record.Version != 1 || record.Quote == nil || record.Checksum != poolRetryChecksum(record) {
		return nil, cache.failure(fmt.Errorf("corrupt or unsupported record"))
	}
	if record.RequestHash != cache.requestHash {
		return nil, cache.failure(fmt.Errorf("record does not match checkout arguments, API endpoint, or login credential; retry with the original inputs and login"))
	}
	if err := validatePoolRetryQuote(record.Quote); err != nil {
		return nil, cache.failure(err)
	}
	// Also complete durability if another process just published this record.
	if err := cache.syncDirs(); err != nil {
		return nil, err
	}
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
	if err := validatePoolRetryQuote(quote); err != nil {
		return err
	}
	if err := config.EnsureConfigDir(); err != nil {
		return cache.failure(err)
	}
	configDir, dir := filepath.Dir(filepath.Dir(cache.path)), filepath.Dir(cache.path)
	if err := checkPoolRetryPermissions(configDir, true); err != nil {
		return cache.failure(err)
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return cache.failure(err)
	}
	if err := checkPoolRetryPermissions(dir, true); err != nil {
		return cache.failure(err)
	}
	record := poolRetryRecord{Version: 1, RequestHash: cache.requestHash, Quote: quote}
	record.Checksum = poolRetryChecksum(record)
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
	if err := os.Link(file.Name(), cache.path); err != nil {
		return cache.failure(fmt.Errorf("could not save confirmed quote (if the key already exists, retry to load it): %w", err))
	}
	return cache.syncDirs()
}

func (cache *poolRetryCache) syncDirs() error {
	dir := filepath.Dir(cache.path)
	configDir := filepath.Dir(dir)
	for _, path := range []string{filepath.Dir(configDir), configDir, dir} {
		if err := syncPoolRetryDir(path); err != nil {
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
