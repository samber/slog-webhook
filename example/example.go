package main

import (
	"fmt"
	"net/http"
	"time"

	slogwebhook "github.com/samber/slog-webhook"

	"golang.org/x/exp/slog"
)

func main() {
	url := "https://webhook.site/xxxxxx"

	logger := slog.New(slogwebhook.Option{Level: slog.LevelDebug, Endpoint: url}.NewWebhookHandler())
	logger = logger.With("release", "v1.0.0")

	req, _ := http.NewRequest(http.MethodGet, "https://api.screeb.app", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TOKEN", "1234567890")

	logger.
		With(
			slog.Group("user",
				slog.String("id", "user-123"),
				slog.Time("created_at", time.Now()),
			),
		).
		With("request", req).
		With("error", fmt.Errorf("an error")).
		Error("a message")
}
