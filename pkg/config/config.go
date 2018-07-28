package config

import "fmt"

type Config struct {
	Language string `yaml:"language"`
	Versions []string `yaml:"versions"`
	Setup []string `yaml:"setup,omitempty"`
	Tests map[string][]string `yaml:"tests,omitempty"`
	// go
	GoImportPath string `yaml:"go_import_path,omitempty"`
}

func (c Config) Validate() error {
	switch c.Language {
	case "go":
	default:
		return fmt.Errorf("language %q not supported", c.Language)
	}
	if len(c.Versions) == 0 {
		return fmt.Errorf("at least 1 version required")
	}
	if len(c.Versions) > 5 {
		return fmt.Errorf("maximum 5 versions allowed")
	}
	return nil
}

func (c Config) dockerImage() string {
	switch c.Language {
	case "go":
		return "golang"
	}
	return "ubuntu"
}

func (c Config) GetSetup() []string {
	if len(c.Setup) > 0 {
		return c.Setup
	}
	return nil
}

func (c Config) GetTests() map[string][]string {
	if len(c.Tests) > 0 {
		return c.Tests
	}
	switch c.Language {
	case "go":
		return map[string][]string{
			"golint":  {
				"go get -u golang.org/x/lint/golint",
				fmt.Sprintf(`golint $(go list ./...) | sed 's/'$(echo $GOPATH/src/%s/ | sed 's/\//\\\//g')'//g'`, c.GoImportPath),
			},
			"go test": {"go test -v ./..."},
		}
	}
	return nil
}

