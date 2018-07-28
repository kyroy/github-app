package github

import (
	"github.com/google/go-github/github"
	"fmt"
	"context"
	"time"
)

type Conclusion string

const (
	Success Conclusion = "success"
	Failure Conclusion = "failure"
	Neutral Conclusion = "neutral"
	Cancelled Conclusion = "cancelled"
	TimedOut Conclusion = "timed_out"
	ActionRequired Conclusion = "action_required"
)

func CreateCheckRun(client *github.Client, owner, repo, branch, sha, name string) (int64, error) {
	checkRun, _, err := client.Checks.CreateCheckRun(context.Background(), owner, repo, github.CreateCheckRunOptions{
		Name: name, // *
		HeadBranch: branch, // *
		HeadSHA: sha, // *
		Status: github.String("in_progress"),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create check_run: %v", err)
	}
	return checkRun.GetID(), nil
}

func UpdateCheckRun(client *github.Client, owner, repo string, checkRunID int64, conclusion Conclusion, title, summary string, text *string, annotations []*github.CheckRunAnnotation) error {
	_, _, err := client.Checks.UpdateCheckRun(context.Background(), owner, repo, checkRunID, github.UpdateCheckRunOptions{
		CompletedAt: &github.Timestamp{Time: time.Now()},
		Conclusion: github.String(string(conclusion)),
		Output: &github.CheckRunOutput{
			Title: &title, // *
			Summary: &summary, // *
			Text: text,
			Annotations: annotations,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update check_run %d: %v", checkRunID, err)
	}
	return nil
}
