package flexmsg

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v3"
)

// FlexTemplate holds the alt text and the template body for a flex message.
type FlexTemplate struct {
	Template *template.Template
	AltText  string
}

type messageYAML struct {
	AltText  string `yaml:"altText"`
	Template string `yaml:"template"`
}

var (
	flexTmpls map[string]FlexTemplate
)

// Load reads the YAML file from the given path and loads the templates into memory.
func Load(path string) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	var yamlTmpls map[string]messageYAML
	if err := yaml.Unmarshal(data, &yamlTmpls); err != nil {
		return fmt.Errorf("failed to unmarshal templates: %w", err)
	}

	flexTmpls = make(map[string]FlexTemplate, len(yamlTmpls))
	for name, t := range yamlTmpls {
		tmpl, err := template.New(name).Funcs(funcMap).Parse(t.Template)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}
		flexTmpls[name] = FlexTemplate{
			AltText:  t.AltText,
			Template: tmpl,
		}
	}

	return nil
}

// Get retrieves a template by its name.
func getFlexTemplate(name string) (FlexTemplate, error) {
	tmpl, ok := flexTmpls[name]
	if !ok {
		return FlexTemplate{}, fmt.Errorf("template not found: %s", name)
	}
	return tmpl, nil
}

var funcMap = template.FuncMap{
	"sub": func(a, b int) int {
		return a - b
	},
}
