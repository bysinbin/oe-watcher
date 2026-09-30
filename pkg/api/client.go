package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"medya-filter/pkg/models"
)

const (
	DefaultBaseURL = "https://oelist-api.frax.tr/v1"
	CacheFileName  = "persons.json"
	SessionFile    = "session.json"
)

type Client struct {
	BaseURL    string
	DataDir    string
	HTTPClient *http.Client
}

type SessionData struct {
	Token      string    `json:"token"`
	LastSynced time.Time `json:"last_synced"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

func NewClient(baseURL, dataDir string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if dataDir == "" {
		dataDir = "data"
	}
	_ = os.MkdirAll(dataDir, 0755)

	return &Client{
		BaseURL: baseURL,
		DataDir: dataDir,
		HTTPClient: &http.Client{
			Timeout: 25 * time.Second,
		},
	}
}

// Authenticate obtains a new JWT token from the API.
func (c *Client) Authenticate() (string, error) {
	url := fmt.Sprintf("%s/owners/self", c.BaseURL)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth failed (status %d): %s", resp.StatusCode, string(body))
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("failed to decode auth response: %w", err)
	}

	if authResp.Token == "" {
		return "", fmt.Errorf("empty token received from API")
	}

	return authResp.Token, nil
}

// Sync triggers persons synchronization on the server.
func (c *Client) Sync(token string) error {
	url := fmt.Sprintf("%s/persons/sync", c.BaseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sync failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// FetchPersons retrieves all persons from the API.
func (c *Client) FetchPersons(token string) ([]models.Person, error) {
	url := fmt.Sprintf("%s/persons", c.BaseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch persons request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch persons failed (status %d): %s", resp.StatusCode, string(body))
	}

	var persons []models.Person
	if err := json.NewDecoder(resp.Body).Decode(&persons); err != nil {
		return nil, fmt.Errorf("failed to decode persons list: %w", err)
	}

	return persons, nil
}

// LoadCachedPersons reads persons from local cache file.
func (c *Client) LoadCachedPersons() ([]models.Person, time.Time, error) {
	cachePath := filepath.Join(c.DataDir, CacheFileName)
	info, err := os.Stat(cachePath)
	if err != nil {
		return nil, time.Time{}, err
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, time.Time{}, err
	}

	var persons []models.Person
	if err := json.Unmarshal(data, &persons); err != nil {
		return nil, time.Time{}, err
	}

	return persons, info.ModTime(), nil
}

// SaveCachedPersons writes persons to local cache file.
func (c *Client) SaveCachedPersons(persons []models.Person) error {
	cachePath := filepath.Join(c.DataDir, CacheFileName)
	data, err := json.MarshalIndent(persons, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

// SyncAndFetch authenticates, syncs, fetches from API, and caches the result.
func (c *Client) SyncAndFetch() ([]models.Person, error) {
	token, err := c.Authenticate()
	if err != nil {
		return nil, fmt.Errorf("authentication error: %w", err)
	}

	if err := c.Sync(token); err != nil {
		return nil, fmt.Errorf("sync error: %w", err)
	}

	persons, err := c.FetchPersons(token)
	if err != nil {
		return nil, fmt.Errorf("fetch error: %w", err)
	}

	// Cache to disk
	_ = c.SaveCachedPersons(persons)

	// Save session
	session := SessionData{
		Token:      token,
		LastSynced: time.Now(),
	}
	sessionData, _ := json.MarshalIndent(session, "", "  ")
	_ = os.WriteFile(filepath.Join(c.DataDir, SessionFile), sessionData, 0644)

	return persons, nil
}

// GetPersons returns persons from cache if recent, or syncs if needed.
func (c *Client) GetPersons(forceSync bool) ([]models.Person, error) {
	if !forceSync {
		persons, modTime, err := c.LoadCachedPersons()
		if err == nil && len(persons) > 0 {
			// If cached within the last 12 hours, use it
			if time.Since(modTime) < 12*time.Hour {
				return persons, nil
			}
		}
	}

	// Try API sync
	persons, err := c.SyncAndFetch()
	if err != nil {
		// If sync failed but we have cached data, fall back to cache
		cached, _, cacheErr := c.LoadCachedPersons()
		if cacheErr == nil && len(cached) > 0 {
			return cached, nil
		}
		return nil, err
	}

	return persons, nil
}
