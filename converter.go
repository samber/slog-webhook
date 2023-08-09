package slogwebhook

import (
	"net/http"
	"reflect"
	"strings"

	"log/slog"

	"github.com/samber/lo"
)

type Converter func(loggerAttr []slog.Attr, record slog.Record) map[string]any

func DefaultConverter(loggerAttr []slog.Attr, record slog.Record) map[string]any {
	extra := attrsToValue(loggerAttr)

	record.Attrs(func(attr slog.Attr) bool {
		extra[attr.Key] = attrToValue(attr)
		return true
	})

	payload := map[string]any{
		"logger":    "samber/slog-webhook",
		"timestamp": record.Time.UTC(),
		"level":     record.Level.String(),
		"message":   record.Message,
	}

	if v, ok := extra["error"]; ok {
		if err, ok := v.(error); ok {
			payload["error"] = buildExceptions(err)
			delete(extra, "error")
		}
	}

	if v, ok := extra["request"]; ok {
		if req, ok := v.(*http.Request); ok {
			payload["request"] = buildRequest(req)
			delete(extra, "request")
		}
	}

	if user, ok := extra["user"]; ok {
		payload["user"] = user
		delete(extra, "user")
	}

	payload["extra"] = extra

	return payload
}

func attrsToValue(attrs []slog.Attr) map[string]any {
	output := map[string]any{}
	for i := range attrs {
		output[attrs[i].Key] = attrToValue(attrs[i])
	}
	return output
}

func attrToValue(attr slog.Attr) any {
	v := attr.Value
	kind := attr.Value.Kind()

	switch kind {
	case slog.KindAny:
		return v.Any()
	case slog.KindLogValuer:
		return v.LogValuer().LogValue().Any()
	case slog.KindGroup:
		return attrsToValue(v.Group())
	case slog.KindInt64:
		return v.Int64()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindString:
		return v.String()
	case slog.KindBool:
		return v.Bool()
	case slog.KindDuration:
		return v.Duration()
	case slog.KindTime:
		return v.Time().UTC()
	default:
		return v.Any()
	}
}

func buildExceptions(err error) map[string]any {
	return map[string]any{
		"kind":  reflect.TypeOf(err).String(),
		"error": err.Error(),
		"stack": nil, // @TODO
	}
}

func buildRequest(req *http.Request) map[string]any {
	return map[string]any{
		"host":   req.Host,
		"method": req.Method,
		"url": map[string]any{
			"url":       req.URL.String(),
			"scheme":    req.URL.Scheme,
			"host":      req.URL.Host,
			"path":      req.URL.Path,
			"raw_query": req.URL.RawQuery,
			"fragment":  req.URL.Fragment,
			"query": lo.MapEntries(req.URL.Query(), func(key string, values []string) (string, string) {
				return key, strings.Join(values, ",")
			}),
		},
		"headers": lo.MapEntries(req.Header, func(key string, values []string) (string, string) {
			return key, strings.Join(values, ",")
		}),
	}
}
