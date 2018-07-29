package golang_test

import (
	"bytes"
	"fmt"
	"github.com/kyroy/github-app/pkg/config"
	"github.com/kyroy/github-app/pkg/golang"
	"github.com/stretchr/testify/require"
	"testing"
)

const configFile = `language: go
versions:
  - "1.11beta2"
  - "1.10"
  - "1.9"
setup:
  - go get -u github.com/golang/dep/cmd/dep
  - dep ensure -vendor-only
`

func createConfig(t *testing.T) *config.Config {
	cfg, err := config.New(bytes.NewReader([]byte(configFile)), "Kyroy", "testrepo")
	require.NoError(t, err)
	return cfg
}

func TestTestGoVersion(t *testing.T) {
	results, _, err := golang.TestGoVersion(createConfig(t), "https://github.com/Kyroy/testrepo.git", "e91d25fff08cfc19b68c6deba142caaaac448561", "golang:1.10")
	require.NoError(t, err)
	for stage, res := range results {
		fmt.Println(stage, res)
	}
}
