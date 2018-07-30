package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"time"
)

type Conclusion *string

var (
	None           Conclusion = nil
	Success        Conclusion = github.String("success")
	Failure        Conclusion = github.String("failure")
	Neutral        Conclusion = github.String("neutral")
	Cancelled      Conclusion = github.String("cancelled")
	TimedOut       Conclusion = github.String("timed_out")
	ActionRequired Conclusion = github.String("action_required")
)

type Status *string

var (
	Queued     Status = github.String("queued")
	InProgress Status = github.String("in_progress")
	Completed  Status = github.String("completed")
)

//
func CreateCheckRun(client *github.Client, owner, repo, branch, sha, name string, status Status, conclusion Conclusion, output *github.CheckRunOutput) (int64, error) {
	var timestamp *github.Timestamp
	if status == Completed {
		timestamp = &github.Timestamp{Time: time.Now()}
	} else if output != nil || conclusion != None {
		return 0, fmt.Errorf("conclusion or output set but status not completed")
	}
	checkRun, _, err := client.Checks.CreateCheckRun(context.Background(), owner, repo, github.CreateCheckRunOptions{
		Name:        name, // *
		Status:      status,
		Conclusion:  conclusion,
		HeadBranch:  branch, // *
		HeadSHA:     sha,    // *
		StartedAt:   &github.Timestamp{Time: time.Now()},
		CompletedAt: timestamp,
		Output:      output,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create check_run: %v", err)
	}
	return checkRun.GetID(), nil
}

func UpdateCheckRun(client *github.Client, owner, repo string, checkRunID int64, name string, status Status, conclusion Conclusion, output *github.CheckRunOutput) error {
	var timestamp *github.Timestamp
	if status == Completed {
		timestamp = &github.Timestamp{Time: time.Now()}
	}
	_, _, err := client.Checks.UpdateCheckRun(context.Background(), owner, repo, checkRunID, github.UpdateCheckRunOptions{
		Name:        name,
		Status:      status,
		Conclusion:  conclusion,
		CompletedAt: timestamp,
		Output:      output,
		//&github.CheckRunOutput{
		//	Title:       &title,   // *
		//	Summary:     &summary, // *
		//	Text:        text,
		//	Annotations: annotations,
		//},
	})
	if err != nil {
		return fmt.Errorf("failed to update check_run %d: %v", checkRunID, err)
	}
	return nil
}
