package server

import (
	_ "embed"
	"encoding/json"
	"net/http"

	"gopkg.in/yaml.v3"
)

//go:embed openapi.yaml
var openapiYAML []byte

type openapiHandlers struct {
	yamlBytes []byte
	jsonBytes []byte
}

func newOpenAPIHandlers() (*openapiHandlers, error) {
	var doc any
	if err := yaml.Unmarshal(openapiYAML, &doc); err != nil {
		return nil, err
	}
	doc = convertMapKeys(doc)
	jsonBytes, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return &openapiHandlers{
		yamlBytes: openapiYAML,
		jsonBytes: jsonBytes,
	}, nil
}

func (h *openapiHandlers) yaml() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(h.yamlBytes)
	}
}

func (h *openapiHandlers) json() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(h.jsonBytes)
	}
}

func convertMapKeys(v any) any {
	switch x := v.(type) {
	case map[any]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[stringKey(k)] = convertMapKeys(val)
		}
		return out
	case map[string]any:
		for k, val := range x {
			x[k] = convertMapKeys(val)
		}
		return x
	case []any:
		for i, val := range x {
			x[i] = convertMapKeys(val)
		}
		return x
	default:
		return v
	}
}

func stringKey(k any) string {
	if s, ok := k.(string); ok {
		return s
	}
	return ""
}
