package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

// --- Misskey API Structs ---

type Note struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	User User   `json:"user"`
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// --- Config ---

type Config struct {
	InstanceURL string `json:"instance_url"`
	AccessToken string `json:"access_token"`
}

func loadConfig() (*Config, error) {
	file, err := os.ReadFile("config.json")
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// --- API Functions ---

func fetchTimeline(client *http.Client, config *Config, timelineType string) ([]Note, error) {
	endpointMap := map[string]string{
		"home":   "/api/notes/timeline",
		"local":  "/api/notes/local-timeline",
		"social": "/api/notes/hybrid-timeline",
		"global": "/api/notes/global-timeline",
	}
	endpoint, err := url.JoinPath(config.InstanceURL, endpointMap[timelineType])
	if err != nil {
		return nil, err
	}

	reqBody, err := json.Marshal(map[string]any{"i": config.AccessToken, "limit": 30})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed: %s", resp.Status)
	}

	var notes []Note
	if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil {
		return nil, err
	}

	return notes, nil
}

func createNote(client *http.Client, config *Config, text string, replyId string) error {
	endpoint, err := url.JoinPath(config.InstanceURL, "/api/notes/create")
	if err != nil {
		return err
	}

	payload := map[string]any{
		"i":    config.AccessToken,
		"text": text,
	}
	if replyId != "" {
		payload["replyId"] = replyId
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed: %s", resp.Status)
	}

	return nil
}

func createReaction(client *http.Client, config *Config, noteId string, reaction string) error {
	endpoint, err := url.JoinPath(config.InstanceURL, "/api/notes/reactions/create")
	if err != nil {
		return err
	}

	reqBody, err := json.Marshal(map[string]any{"i": config.AccessToken, "noteId": noteId, "reaction": reaction})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("API request failed: %s", resp.Status)
	}

	return nil
}

func fetchNoteConversation(client *http.Client, config *Config, noteId string) ([]Note, error) {
	endpoint, err := url.JoinPath(config.InstanceURL, "/api/notes/conversation")
	if err != nil {
		return nil, err
	}

	reqBody, err := json.Marshal(map[string]any{"i": config.AccessToken, "noteId": noteId})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed: %s", resp.Status)
	}

	var notes []Note
	if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil {
		return nil, err
	}

	return notes, nil
}
