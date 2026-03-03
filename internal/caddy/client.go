package caddy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"forgejo.wally.mywire.org/diego/WallyDic.git/internal/logging"
	"github.com/hashicorp/go-retryablehttp"
)

const (
	defaultRequestTimeout            = 5 * time.Second
	defaultRetryMax                  = 3
	caddyReloaderAuthTokenHeaderName = "X-Reloader-Token"
)

type ReloaderClient struct {
	url       string
	authToken string

	signals chan struct{}
	client  *retryablehttp.Client
}

func NewReloaderClient(webhookURL string, token string) *ReloaderClient {
	client := retryablehttp.NewClient()
	client.RetryMax = defaultRetryMax
	client.Logger = nil // disable internal logging noise
	if client.HTTPClient == nil {
		client.HTTPClient = &http.Client{}
	}
	client.HTTPClient.Timeout = defaultRequestTimeout

	return &ReloaderClient{
		url:       webhookURL,
		authToken: token,
		signals:   make(chan struct{}, 1),
		client:    client,
	}
}

// NotifyChange enqueues a non-blocking signal that Caddy needs to be reloaded
func (n *ReloaderClient) NotifyChange(_ context.Context) {
	select {
	case n.signals <- struct{}{}:
	default:
	}
}

// Run processes change signals until the context is cancelled.
func (n *ReloaderClient) Run(ctx context.Context) error {
	ctx, logger := logging.Enrich(ctx,
		slog.String(logging.AttrKeyComponent, "caddy_reloader"),
		slog.String("webhook_url", n.url),
	)

	logger.Info("starting caddy reloader")

	for {
		select {
		case <-n.signals:
			if ctx.Err() != nil {
				return nil
			}
			if err := n.sendOnce(ctx); err != nil {
				logger.Error("failed to reload caddy",
					slog.Any(logging.AttrKeyError, err),
				)
				continue
			}
			logger.Info("whitelist change notification sent")
		case <-ctx.Done():
			logger.Info("stopping caddy reloader")
			return nil
		}
	}
}

// sendOnce Sends a single request to the caddy reload endpoint. If error request is retried up to defaultRetryMax
func (n *ReloaderClient) sendOnce(ctx context.Context) error {
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, n.url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if n.authToken != "" {
		// Update authToken name
		req.Header.Set(caddyReloaderAuthTokenHeaderName, n.authToken)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
