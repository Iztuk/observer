package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type HostRulesDoc struct {
	Rules map[string]HostRule `json:"rules" yaml:"rules"`
}

type HostRule struct {
	Name        string   `json:"name" yaml:"name"`
	Enabled     bool     `json:"enabled" yaml:"enabled"`
	JobType     JobType  `json:"job_type" yaml:"job_type"`
	Type        RuleType `json:"type" yaml:"type"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`

	Match   RuleMatch   `json:"match" yaml:"match"`
	Finding RuleFinding `json:"finding" yaml:"finding"`
}

type RuleType string

const (
	RuleTypePath      RuleType = "path"
	RuleTypeQuery     RuleType = "query"
	RuleTypeHeader    RuleType = "header"
	RuleTypeBodyField RuleType = "body_field"
)

type RuleMatch struct {
	Paths        []string          `json:"paths,omitempty" yaml:"paths,omitempty"`
	Methods      []string          `json:"methods,omitempty" yaml:"methods,omitempty"`
	Headers      map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	QueryParams  map[string]string `json:"query_params,omitempty" yaml:"query_params,omitempty"`
	ContentTypes []string          `json:"content_types,omitempty" yaml:"content_types,omitempty"`
	Fields       []string          `json:"fields,omitempty" yaml:"fields,omitempty"`
	Patterns     []RulePattern     `json:"patterns,omitempty" yaml:"patterns,omitempty"`
}

type RulePattern struct {
	Target  string `json:"target" yaml:"target"`                 // query, header, path, body, field
	Name    string `json:"name,omitempty" yaml:"name,omitempty"` // param/header/field name
	Pattern string `json:"pattern" yaml:"pattern"`               // regex
}

type RuleFinding struct {
	Title   string `json:"title" yaml:"title"`
	Message string `json:"message" yaml:"message"`
}

func LoadRulesDocument(path string) (HostRulesDoc, error) {
	var doc HostRulesDoc

	data, err := os.ReadFile(path)
	if err != nil {
		return HostRulesDoc{}, err
	}

	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &doc); err != nil {
			return HostRulesDoc{}, fmt.Errorf("parse Host Rules JSON document: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return HostRulesDoc{}, fmt.Errorf("parse Host Rules YAML document: %w", err)
		}
	default:
		return HostRulesDoc{}, fmt.Errorf("unsupported Host Rules document type: %s", path)
	}

	return doc, nil
}
