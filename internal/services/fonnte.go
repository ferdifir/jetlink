package services

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type FonnteClient struct {
	httpClient *http.Client
	token      string
}

func NewFonnteClient(token string) *FonnteClient {
	return &FonnteClient{
		httpClient: &http.Client{},
		token:      token,
	}
}

func (f *FonnteClient) SendMessage(to, message string) error {
	if f.token == "" {
		log.Printf("[fonnte] no token set, skipping message to %s", to)
		return nil
	}

	data := url.Values{}
	data.Set("target", to)
	data.Set("message", message)

	req, err := http.NewRequest("POST", "https://api.fonnte.com/send", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("fonnte build request: %w", err)
	}

	req.Header.Set("Authorization", f.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fonnte send: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fonnte error %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("[fonnte] sent to %s: %s", to, string(body))
	return nil
}
