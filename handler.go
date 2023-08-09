package slogwebhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"log/slog"
)

type Option struct {
	// log level (default: debug)
	Level slog.Leveler

	// URL
	Endpoint string

	// optional: customize webhook event builder
	Converter Converter
}

func (o Option) NewWebhookHandler() slog.Handler {
	if o.Level == nil {
		o.Level = slog.LevelDebug
	}

	return &WebhookHandler{
		option: o,
		attrs:  []slog.Attr{},
		groups: []string{},
	}
}

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

	payload := converter(h.attrs, record)
	return send(h.option.Endpoint, payload)
}

func (h *WebhookHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &WebhookHandler{
		option: h.option,
		attrs:  appendAttrsToGroup(h.groups, h.attrs, attrs),
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

func send(endpoint string, payload map[string]any) error {
	client := http.Client{
		Timeout: time.Duration(10) * time.Second,
	}

	json, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	body := bytes.NewBuffer(json)

	req, err := http.NewRequest("POST", endpoint, body)
	if err != nil {
		return err
	}

	req.Header.Add("content-type", `application/json`)
	req.Header.Add("user-agent", `samber/slog-webhook`)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}
