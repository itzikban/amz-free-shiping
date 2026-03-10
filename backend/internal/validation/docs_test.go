package validation

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestREADMEExists(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Fatalf("README.md does not exist at expected location: %s", readmePath)
	}
}

func TestREADMENotEmpty(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		t.Fatal("README.md should not be empty")
	}
}

func TestREADMEHasTitle(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	hasH1 := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "# ") {
			hasH1 = true
			break
		}
	}

	if !hasH1 {
		t.Error("README.md should have at least one H1 title (starting with '# ')")
	}
}

func TestREADMEHasProjectName(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "amz-free-shiping") {
		t.Error("README.md should contain the project name 'amz-free-shiping'")
	}
}

func TestREADMEMonorepoStructure(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := strings.ToLower(string(data))
	if !strings.Contains(content, "monorepo") {
		t.Error("README.md should mention that this is a monorepo")
	}
}

func TestREADMEHasBackendSection(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "backend") && !strings.Contains(content, "Backend") {
		t.Error("README.md should mention the backend service")
	}
}

func TestREADMEHasQuickStart(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := strings.ToLower(string(data))
	hasQuickStart := strings.Contains(content, "quick start") || strings.Contains(content, "quickstart")

	if !hasQuickStart {
		t.Error("README.md should have a quick start section")
	}
}

func TestREADMECodeBlocks(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := string(data)
	codeBlockPattern := regexp.MustCompile("```[a-z]*\n[\\s\\S]*?\n```")
	matches := codeBlockPattern.FindAllString(content, -1)

	if len(matches) == 0 {
		t.Error("README.md should contain at least one code block for examples")
	}
}

func TestREADMECodeBlocksWellFormed(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := string(data)
	openBlocks := strings.Count(content, "```")

	if openBlocks%2 != 0 {
		t.Error("README.md has unmatched code block delimiters (```)")
	}
}

func TestREADMEHasBashCommands(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := string(data)
	hasBashBlock := strings.Contains(content, "```bash")

	if !hasBashBlock {
		t.Error("README.md should contain bash code blocks for command examples")
	}
}

func TestREADMEGoModTidy(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "go mod tidy") {
		t.Error("README.md quick start should mention 'go mod tidy'")
	}
}

func TestREADMEEnvironmentSetup(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := string(data)
	hasEnvSetup := strings.Contains(content, ".env") || strings.Contains(content, "environment")

	if !hasEnvSetup {
		t.Error("README.md should mention environment setup or .env file")
	}
}

func TestREADMEAPIPort(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "8085") {
		t.Error("README.md should mention the API port (8085)")
	}
}

func TestREADMELayoutSection(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := strings.ToLower(string(data))
	hasLayout := strings.Contains(content, "layout") || strings.Contains(content, "structure")

	if !hasLayout {
		t.Error("README.md should describe the repository layout/structure")
	}
}

func TestREADMECurrentAndPlannedLayout(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := strings.ToLower(string(data))
	hasCurrent := strings.Contains(content, "current")
	hasPlanned := strings.Contains(content, "planned")

	if !hasCurrent {
		t.Error("README.md should have a 'Current layout' section")
	}

	if !hasPlanned {
		t.Error("README.md should have a 'Planned layout' section")
	}
}

func TestREADMENoLongLines(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	maxLineLength := 120

	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "http") && len(line) > maxLineLength {
			t.Logf("line %d exceeds recommended length of %d characters (has %d)", i+1, maxLineLength, len(line))
		}
	}
}

func TestREADMENoTrailingWhitespace(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if len(line) > 0 && (line[len(line)-1] == ' ' || line[len(line)-1] == '\t') {
			t.Errorf("line %d has trailing whitespace", i+1)
		}
	}
}

func TestREADMEProperHeadingHierarchy(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	hasH1 := false
	h1Count := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") && !strings.HasPrefix(trimmed, "## ") {
			hasH1 = true
			h1Count++
		}
	}

	if !hasH1 {
		t.Error("README.md should have at least one H1 heading")
	}

	if h1Count > 1 {
		t.Logf("Warning: README.md has %d H1 headings; typically one is recommended", h1Count)
	}
}

func TestREADMEBackendDirectoryReference(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "backend/") {
		t.Error("README.md should reference the backend/ directory")
	}
}

func TestREADMEServiceDescription(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := strings.ToLower(string(data))
	hasShipping := strings.Contains(content, "shipping") || strings.Contains(content, "ship")

	if !hasShipping {
		t.Error("README.md should describe what the service does (shipping-related)")
	}
}

func TestREADMEMentionsGo(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := string(data)
	hasGo := strings.Contains(content, "Go ") || strings.Contains(content, "go ") ||
		strings.Contains(content, "go run") || strings.Contains(content, "go mod")

	if !hasGo {
		t.Error("README.md should mention that the backend is written in Go")
	}
}

func TestREADMEHasFrontendPlanned(t *testing.T) {
	readmePath := filepath.Join("..", "..", "..", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	content := string(data)
	hasFrontend := strings.Contains(content, "frontend")

	if !hasFrontend {
		t.Error("README.md should mention planned frontend based on monorepo structure")
	}
}