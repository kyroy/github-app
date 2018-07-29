package config

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"gopkg.in/yaml.v2"
	"io"
)

const name = ".kyroy.yaml"

func Download(client *github.Client, owner, repo, ref string) (*Config, error) {
	f, err := client.Repositories.DownloadContents(context.Background(), owner, repo, name, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return nil, fmt.Errorf("could not download config file: %v", err)
	}
	return New(f, owner, repo)
}

func New(r io.Reader, owner, repo string) (*Config, error) {
	var cfg hiddenConfig
	if err := yaml.NewDecoder(r).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("could not decode config file: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config file: %v", err)
	}
	if cfg.GoImportPath == "" {
		cfg.GoImportPath = fmt.Sprintf("github.com/%s/%s", owner, repo)
	}
	return &Config{
		hidden: cfg,
	}, nil
}
