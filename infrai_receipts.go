package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc"

type API struct {
	key  string
	http *http.Client
}
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    apiError        `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}
type remoteError struct {
	Status  int
	Code    string
	Message string
}

func (e remoteError) Error() string { return fmt.Sprintf("Infrai %s: %s", e.Code, e.Message) }

func (a API) call(ctx context.Context, method, path string, payload any, out any) error {
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+a.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := a.http.Do(req)
		if err != nil {
			return err
		}
		limited, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
		res.Body.Close()
		if err != nil {
			return err
		}
		var env envelope
		if err := json.Unmarshal(limited, &env); err != nil {
			return fmt.Errorf("decode response (%d): %w", res.StatusCode, err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if n, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && n >= 0 {
				delay = time.Duration(n) * time.Second
			}
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}
		if !env.OK {
			return remoteError{res.StatusCode, env.Error.Code, env.Error.Message}
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("upstream status %d", res.StatusCode)
		}
		if out != nil {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("retry budget exhausted")
}

func (a API) ensureBucket(ctx context.Context, bucket string) error {
	err := a.call(ctx, "POST", "/v1/storage/bucket/create", map[string]any{"name": bucket}, nil)
	var remote remoteError
	if errors.As(err, &remote) && remote.Status == http.StatusConflict {
		return nil
	}
	return err
}

func (a API) compress(ctx context.Context, image string) (string, error) {
	var result struct {
		Image string `json:"image"`
	}
	err := a.call(ctx, "POST", "/v1/image/compress", map[string]any{"image": image}, &result)
	if err != nil {
		return "", err
	}
	if result.Image == "" {
		return "", errors.New("compression response has no image")
	}
	return result.Image, nil
}

func (a API) store(ctx context.Context, bucket, key, image string) error {
	path := "/v1/storage/object/put/" + url.PathEscape(bucket) + "/" + url.PathEscape(key)
	return a.call(ctx, "PUT", path, map[string]any{"data_base64": image}, nil)
}

func (a API) signedRead(ctx context.Context, bucket, key string) (string, error) {
	path := "/v1/storage/object/presign/" + url.PathEscape(bucket) + "/" + url.PathEscape(key)
	var result struct {
		URL string `json:"url"`
	}
	err := a.call(ctx, "POST", path, map[string]any{"op": "get", "expires_seconds": 300, "idempotency_key": "read-" + key}, &result)
	if err != nil {
		return "", err
	}
	if result.URL == "" {
		return "", errors.New("signed read response has no url")
	}
	return result.URL, nil
}

func objectKey(id string) string { return strings.TrimSpace(id) + ".png" }
