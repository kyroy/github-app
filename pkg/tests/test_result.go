package tests

import (
	"github.com/google/go-github/github"
	"fmt"
)

type Result struct {
	File string
	Line *int
	Message string
}

func (r *Result) Valid() bool {
	return r.File != "" && r.Line != nil && r.Message != ""
}

// version -> stage -> results
type Results map[string]map[string][]*Result

func (r Results) Annotations(version, stage, owner, repo, sha string) ([]*github.CheckRunAnnotation, error) {
	versionResults, ok := r[version]
	if !ok {
		return nil, fmt.Errorf("failed to get version %s results", version)
	}
	stageResults, ok := versionResults[stage]
	if !ok {
		return nil, fmt.Errorf("failed to get version %s stage %s results", version, stage)
	}
	var annotations []*github.CheckRunAnnotation
	for _, res := range stageResults {
		annotations = append(annotations, &github.CheckRunAnnotation{
			FileName: &res.File, // *
			BlobHRef: github.String(fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, sha, res.File)), // *
			StartLine: res.Line, // *
			EndLine: res.Line, // *
			WarningLevel: github.String("failure"),// * notice, warning, failure
			Message: &res.Message, // *
		})
	}
	return annotations, nil
}
