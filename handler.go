package slogwebhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"log/slog"

	slogcommon "github.com/samber/slog-common"
)

type Option struct {
	// log level (default: debug)
	Level slog.Leveler

	// URL
	Endpoint string
	Timeout  time.Duration // default: 10s

	// optional: customize webhook event builder
	Converter Converter

	// optional: see slog.HandlerOptions
	AddSource   bool
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
}

func (o Option) NewWebhookHandler() slog.Handler {
	if o.Level == nil {
		o.Level = slog.LevelDebug
	}

	if o.Timeout == 0 {
		o.Timeout = 10 * time.Second
	}

	return &WebhookHandler{
		option: o,
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

var _ slog.Handler = (*WebhookHandler)(nil)

type WebhookHandler struct {
	option Option
	attrs  []slog.Attr
	groups []string
}

func (h *WebhookHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.option.Level.Level()
}

func (h *WebhookHandler) Handle(ctx context.Context, record slog.Record) error {
	converter := DefaultConverter
	if h.option.Converter != nil {
		converter = h.option.Converter
	}

	payload := converter(h.option.AddSource, h.option.ReplaceAttr, h.attrs, h.groups, &record)

	go func() {
		_ = send(h.option.Endpoint, h.option.Timeout, payload)
	}()

	return nil
}

func (h *WebhookHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &WebhookHandler{
		option: h.option,
		attrs:  slogcommon.AppendAttrsToGroup(h.groups, h.attrs, attrs...),
		groups: h.groups,
	}
}

func (h *WebhookHandler) WithGroup(name string) slog.Handler {
	return &WebhookHandler{
		option: h.option,
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
}

func send(endpoint string, timeout time.Duration, payload map[string]any) error {
	client := http.Client{
		Timeout: time.Duration(10) * time.Second,
	}

	json, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	body := bytes.NewBuffer(json)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, body)
	if err != nil {
		return err
	}

	req.Header.Add("content-type", `application/json`)
	req.Header.Add("user-agent", name)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}
