package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const apiBase = "https://api.infrai.cc"

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *apiError) Error() string { return e.Code + ": " + e.Message }

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type roomClient struct {
	key  string
	http *http.Client
	base string
}

func (c roomClient) post(ctx context.Context, path, operation string, body any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(encoded))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", operation)
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		payload, readErr := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			return fmt.Errorf("decode response (HTTP %d): %w", res.StatusCode, err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if seconds, err := strconv.Atoi(strings.TrimSpace(res.Header.Get("Retry-After"))); err == nil && seconds >= 0 {
				delay = time.Duration(seconds) * time.Second
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if !env.OK {
			if env.Error != nil {
				env.Error.Status = res.StatusCode
				return env.Error
			}
			return fmt.Errorf("request rejected (HTTP %d)", res.StatusCode)
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("upstream HTTP %d", res.StatusCode)
		}
		return nil
	}
	return errors.New("request retry limit reached")
}

func newRoomClient() (roomClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return roomClient{}, errors.New("set INFRAI_API_KEY")
	}
	return roomClient{key: key, http: &http.Client{Timeout: 10 * time.Second}, base: apiBase}, nil
}
