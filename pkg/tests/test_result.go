package tests

import (
	"fmt"
	"github.com/google/go-github/github"
)

type Result struct {
	File    string
	Line    *int
	Message string
}

func (r *Result) Valid() bool {
	return r.File != "" && r.Line != nil && r.Message != ""
}

// version -> stage -> results
type Results map[string]map[string][]*Result

func (r Results) Annotations(version, owner, repo, sha string) ([]*github.CheckRunAnnotation, error) {
	versionResults, ok := r[version]
	if !ok {
		return nil, fmt.Errorf("failed to get version %s results", version)
	}
	var annotations []*github.CheckRunAnnotation
	for stage, results := range versionResults {
		for _, res := range results {
			annotations = append(annotations, &github.CheckRunAnnotation{
				Title:        &stage,
				Message:      &res.Message,
				FileName:     &res.File,
				BlobHRef:     github.String(fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, sha, res.File)),
				StartLine:    res.Line,
				EndLine:      res.Line,
				WarningLevel: github.String("failure"),
			})
		}
	}
	return annotations, nil
}
