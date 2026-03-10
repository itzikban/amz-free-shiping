package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type CodeRabbitConfig struct {
	Language    string `yaml:"language"`
	EarlyAccess bool   `yaml:"early_access"`
	Reviews     struct {
		Profile                string   `yaml:"profile"`
		RequestChangesWorkflow bool     `yaml:"request_changes_workflow"`
		HighLevelSummary       bool     `yaml:"high_level_summary"`
		Poem                   bool     `yaml:"poem"`
		ReviewStatus           bool     `yaml:"review_status"`
		CollapseWalkthrough    bool     `yaml:"collapse_walkthrough"`
		PathFilters            []string `yaml:"path_filters"`
		PathInstructions       []struct {
			Path         string `yaml:"path"`
			Instructions string `yaml:"instructions"`
		} `yaml:"path_instructions"`
	} `yaml:"reviews"`
	Chat struct {
		AutoReply bool `yaml:"auto_reply"`
	} `yaml:"chat"`
}

func TestCodeRabbitYAMLExists(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", ".coderabbit.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf(".coderabbit.yaml does not exist at expected location: %s", configPath)
	}
}

func TestCodeRabbitYAMLValidSyntax(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", ".coderabbit.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .coderabbit.yaml: %v", err)
	}

	var config CodeRabbitConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("invalid YAML syntax in .coderabbit.yaml: %v", err)
	}
}

func TestCodeRabbitYAMLRequiredFields(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", ".coderabbit.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .coderabbit.yaml: %v", err)
	}

	var config CodeRabbitConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse .coderabbit.yaml: %v", err)
	}

	if config.Language == "" {
		t.Error("language field is required but missing or empty")
	}

	if config.Reviews.Profile == "" {
		t.Error("reviews.profile field is required but missing or empty")
	}
}

func TestCodeRabbitYAMLLanguageValue(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", ".coderabbit.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .coderabbit.yaml: %v", err)
	}

	var config CodeRabbitConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse .coderabbit.yaml: %v", err)
	}

	validLanguages := map[string]bool{
		"en-US": true,
		"en-GB": true,
		"es":    true,
		"fr":    true,
		"de":    true,
	}

	if !validLanguages[config.Language] {
		t.Logf("Warning: language value '%s' may not be a standard locale", config.Language)
	}
}

func TestCodeRabbitYAMLPathFilters(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", ".coderabbit.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .coderabbit.yaml: %v", err)
	}

	var config CodeRabbitConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse .coderabbit.yaml: %v", err)
	}

	if len(config.Reviews.PathFilters) == 0 {
		t.Error("path_filters should not be empty")
	}

	expectedFilters := map[string]bool{
		"backend/**":        false,
		"README.md":         false,
		".coderabbit.yaml":  false,
	}

	for _, filter := range config.Reviews.PathFilters {
		if _, exists := expectedFilters[filter]; exists {
			expectedFilters[filter] = true
		}
	}

	for filter, found := range expectedFilters {
		if !found {
			t.Errorf("expected path_filter '%s' not found in configuration", filter)
		}
	}
}

func TestCodeRabbitYAMLPathFiltersSelfReference(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", ".coderabbit.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .coderabbit.yaml: %v", err)
	}

	var config CodeRabbitConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse .coderabbit.yaml: %v", err)
	}

	hasSelfReference := false
	for _, filter := range config.Reviews.PathFilters {
		if filter == ".coderabbit.yaml" {
			hasSelfReference = true
			break
		}
	}

	if !hasSelfReference {
		t.Error(".coderabbit.yaml should include itself in path_filters for self-documentation")
	}
}

func TestCodeRabbitYAMLPathInstructions(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", ".coderabbit.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .coderabbit.yaml: %v", err)
	}

	var config CodeRabbitConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse .coderabbit.yaml: %v", err)
	}

	if len(config.Reviews.PathInstructions) == 0 {
		t.Error("path_instructions should not be empty")
	}

	for i, instruction := range config.Reviews.PathInstructions {
		if instruction.Path == "" {
			t.Errorf("path_instructions[%d].path is empty", i)
		}
		if instruction.Instructions == "" {
			t.Errorf("path_instructions[%d].instructions is empty", i)
		}
	}
}

func TestCodeRabbitYAMLBackendPathInstructions(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", ".coderabbit.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .coderabbit.yaml: %v", err)
	}

	var config CodeRabbitConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse .coderabbit.yaml: %v", err)
	}

	hasBackendInstructions := false
	for _, instruction := range config.Reviews.PathInstructions {
		if instruction.Path == "backend/**" {
			hasBackendInstructions = true

			requiredKeywords := []string{"monorepo", "backend", "Go"}
			for _, keyword := range requiredKeywords {
				if !strings.Contains(instruction.Instructions, keyword) {
					t.Errorf("backend path instructions should mention '%s'", keyword)
				}
			}
			break
		}
	}

	if !hasBackendInstructions {
		t.Error("should have specific instructions for backend/** path")
	}
}

func TestCodeRabbitYAMLReviewProfile(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", ".coderabbit.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .coderabbit.yaml: %v", err)
	}

	var config CodeRabbitConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse .coderabbit.yaml: %v", err)
	}

	validProfiles := map[string]bool{
		"chill":     true,
		"assertive": true,
	}

	if !validProfiles[config.Reviews.Profile] {
		t.Errorf("review profile '%s' may not be valid; expected 'chill' or 'assertive'", config.Reviews.Profile)
	}
}

func TestCodeRabbitYAMLBooleanFields(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", ".coderabbit.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .coderabbit.yaml: %v", err)
	}

	var rawConfig map[string]interface{}
	if err := yaml.Unmarshal(data, &rawConfig); err != nil {
		t.Fatalf("failed to parse .coderabbit.yaml: %v", err)
	}

	reviews, ok := rawConfig["reviews"].(map[string]interface{})
	if !ok {
		t.Fatal("reviews section is malformed")
	}

	boolFields := []string{
		"request_changes_workflow",
		"high_level_summary",
		"poem",
		"review_status",
		"collapse_walkthrough",
	}

	for _, field := range boolFields {
		if val, exists := reviews[field]; exists {
			if _, isBool := val.(bool); !isBool {
				t.Errorf("field 'reviews.%s' should be a boolean, got %T", field, val)
			}
		}
	}
}

func TestCodeRabbitYAMLChatAutoReply(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", ".coderabbit.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .coderabbit.yaml: %v", err)
	}

	var config CodeRabbitConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse .coderabbit.yaml: %v", err)
	}

	t.Logf("chat.auto_reply is set to: %v", config.Chat.AutoReply)
}

func TestCodeRabbitYAMLNoTrailingWhitespace(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", ".coderabbit.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .coderabbit.yaml: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if len(line) > 0 && line[len(line)-1] == ' ' {
			t.Errorf("line %d has trailing whitespace", i+1)
		}
	}
}

func TestCodeRabbitYAMLConsistentIndentation(t *testing.T) {
	configPath := filepath.Join("..", "..", "..", ".coderabbit.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read .coderabbit.yaml: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "\t") {
			t.Errorf("line %d uses tabs instead of spaces for indentation", i+1)
		}
	}
}