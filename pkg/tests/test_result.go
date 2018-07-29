package tests

import (
	"fmt"
	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
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
type Results map[string]StageResults

func (r Results) Annotations(version, owner, repo, sha string) ([]*github.CheckRunAnnotation, error) {
	versionResults, ok := r[version]
	if !ok {
		return nil, fmt.Errorf("failed to get version %s results", version)
	}
	return versionResults.Annotations(owner, repo, sha)
}

type StageResults map[string][]*Result

func (r StageResults) Annotations(owner, repo, sha string) ([]*github.CheckRunAnnotation, error) {
	var annotations []*github.CheckRunAnnotation
	for stage, results := range r {
		for _, res := range results {
			logrus.Debugf("%s: %v", stage, res)
			annotations = append(annotations, &github.CheckRunAnnotation{
				Title:        github.String(stage),
				Message:      &res.Message,
				FileName:     &res.File,
				BlobHRef:     github.String(fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, sha, res.File)),
				StartLine:    res.Line,
				EndLine:      res.Line,
				WarningLevel: github.String("failure"),
			})
		}
	}
	logrus.Debugf("annotations: %v", annotations)
	for _, a := range annotations {
		logrus.Debugf(" - %s: %v", a.GetTitle(), a)
	}
	return annotations, nil
}
