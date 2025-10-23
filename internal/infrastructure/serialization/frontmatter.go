package serialization

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	yamlDelimiter = "---"
)

// FrontmatterDocument represents a document with YAML frontmatter
type FrontmatterDocument struct {
	Frontmatter map[string]interface{}
	Content     string
}

// ParseFrontmatter parses a markdown file with YAML frontmatter
func ParseFrontmatter(data []byte) (*FrontmatterDocument, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	doc := &FrontmatterDocument{
		Frontmatter: make(map[string]interface{}),
	}

	// Check if file starts with delimiter
	if !scanner.Scan() {
		return doc, nil // Empty file
	}

	firstLine := strings.TrimSpace(scanner.Text())
	if firstLine != yamlDelimiter {
		// No frontmatter, entire content is body
		doc.Content = string(data)
		return doc, nil
	}

	// Read frontmatter until next delimiter
	var frontmatterLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == yamlDelimiter {
			break
		}
		frontmatterLines = append(frontmatterLines, line)
	}

	// Parse YAML frontmatter
	if len(frontmatterLines) > 0 {
		frontmatterYAML := strings.Join(frontmatterLines, "\n")
		if err := yaml.Unmarshal([]byte(frontmatterYAML), &doc.Frontmatter); err != nil {
			return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
		}
	}

	// Read remaining content
	var contentLines []string
	for scanner.Scan() {
		contentLines = append(contentLines, scanner.Text())
	}
	doc.Content = strings.TrimSpace(strings.Join(contentLines, "\n"))

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading document: %w", err)
	}

	return doc, nil
}

// SerializeFrontmatter serializes a document with YAML frontmatter
func SerializeFrontmatter(frontmatter map[string]interface{}, content string) ([]byte, error) {
	var buf bytes.Buffer

	// Write frontmatter delimiter
	buf.WriteString(yamlDelimiter)
	buf.WriteString("\n")

	// Write YAML frontmatter
	if len(frontmatter) > 0 {
		yamlData, err := yaml.Marshal(frontmatter)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal frontmatter: %w", err)
		}
		buf.Write(yamlData)
	}

	// Write closing delimiter
	buf.WriteString(yamlDelimiter)
	buf.WriteString("\n")

	// Write content
	if content != "" {
		buf.WriteString(content)
		buf.WriteString("\n")
	}

	return buf.Bytes(), nil
}

// GetString safely gets a string value from frontmatter
func (d *FrontmatterDocument) GetString(key string) string {
	if val, ok := d.Frontmatter[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetInt safely gets an int value from frontmatter
func (d *FrontmatterDocument) GetInt(key string) int {
	if val, ok := d.Frontmatter[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 0
}

// GetStringSlice safely gets a string slice from frontmatter
func (d *FrontmatterDocument) GetStringSlice(key string) []string {
	if val, ok := d.Frontmatter[key]; ok {
		if slice, ok := val.([]interface{}); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return []string{}
}

// MarkdownDocument represents a markdown document with title and content
type MarkdownDocument struct {
	Title   string
	Content string
}

// ParseMarkdownWithTitle parses a markdown document expecting an H1 title
func ParseMarkdownWithTitle(data []byte) (*MarkdownDocument, error) {
	content := string(data)
	lines := strings.Split(content, "\n")

	doc := &MarkdownDocument{}

	if len(lines) == 0 {
		return doc, nil
	}

	// Check if first line is H1
	if strings.HasPrefix(strings.TrimSpace(lines[0]), "# ") {
		doc.Title = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[0]), "# "))

		// Skip empty lines after title
		startIdx := 1
		for startIdx < len(lines) && strings.TrimSpace(lines[startIdx]) == "" {
			startIdx++
		}

		// Join remaining lines as content
		if startIdx < len(lines) {
			doc.Content = strings.TrimSpace(strings.Join(lines[startIdx:], "\n"))
		}
	} else {
		// No H1 found, entire content is description
		doc.Content = strings.TrimSpace(content)
	}

	return doc, nil
}

// SerializeMarkdownWithTitle serializes a markdown document with H1 title
func SerializeMarkdownWithTitle(title, content string) []byte {
	var buf bytes.Buffer

	if title != "" {
		buf.WriteString("# ")
		buf.WriteString(title)
		buf.WriteString("\n\n")
	}

	if content != "" {
		buf.WriteString(content)
		if !strings.HasSuffix(content, "\n") {
			buf.WriteString("\n")
		}
	}

	return buf.Bytes()
}

// SerializeYaml serializes data to YAML format
func SerializeYaml(data interface{}) ([]byte, error) {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return yamlData, nil
}

// ParseYaml parses YAML data into the provided target
func ParseYaml(data []byte, target interface{}) error {
	if err := yaml.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	return nil
}
