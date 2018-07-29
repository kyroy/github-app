package config

import "fmt"

type hiddenConfig struct {
	Language string              `yaml:"language"`
	Versions []string            `yaml:"versions"`
	Setup    []string            `yaml:"setup,omitempty"`
	Tests    map[string][]string `yaml:"tests,omitempty"`
	// go
	GoImportPath string `yaml:"go_import_path,omitempty"`
}

func (c hiddenConfig) Validate() error {
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

type Config struct {
	hidden hiddenConfig
}

func (c Config) DockerImage() string {
	switch c.hidden.Language {
	case "go":
		return "golang"
	}
	return "ubuntu"
}

func (c Config) Tags() []string {
	return c.hidden.Versions
}

func (c Config) Versions() []string {
	versions := make([]string, len(c.Tags()))
	for i, version := range c.Tags() {
		versions[i] = fmt.Sprintf("%s:%s", c.DockerImage(), version)
	}
	return versions
}

func (c Config) GoImportPath() string {
	return c.hidden.GoImportPath
}

func (c Config) SetupCommands() []string {
	if len(c.hidden.Setup) > 0 {
		return c.hidden.Setup
	}
	return nil
}

func (c Config) TestCommands() map[string][]string {
	if len(c.hidden.Tests) > 0 {
		return c.hidden.Tests
	}
	switch c.hidden.Language {
	case "go":
		return map[string][]string{
			"golint": {
				"go get -u golang.org/x/lint/golint",
				fmt.Sprintf(`golint $(go list ./...) | sed 's/'$(echo $GOPATH/src/%s/ | sed 's/\//\\\//g')'//g'`, c.GoImportPath()),
			},
			"go test": {"go test -v ./..."},
		}
	}
	return nil
}
