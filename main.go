package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
	config2 "github.com/kyroy/github-app/pkg/config"
	github2 "github.com/kyroy/github-app/pkg/github"
	"github.com/kyroy/github-app/pkg/golang"
	"github.com/kyroy/github-app/pkg/ratelimit"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	//results, messages, err := golang.TestGoRepo(&config2.Config{
	//	Language: "go",
	//	Versions: []string{"golang:1.10"},
	//	GoImportPath: "github.com/Kyroy/testrepo",
	//}, "https://github.com/Kyroy/testrepo.git", "e91d25fff08cfc19b68c6deba142caaaac448561")
	//fmt.Println(results, messages, err)

	http.HandleFunc("/", handler)

	logrus.Infof("listening on 8080")
	logrus.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	//secret := []byte("don't tell!")

	logrus.Infof("header %v", r.Header)

	event := r.Header.Get("X-Github-Event")
	if event == "" {
		logrus.Errorf("failed to read event header")
		w.WriteHeader(400)
		return
	}
	switch event {
	case "check_suite":
		var evt github.CheckSuiteEvent
		if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
			logrus.Errorf("failed to unmarshal payload: %v", err)
			w.WriteHeader(500)
			return
		}

		fmt.Println("checking with", int(evt.CheckSuite.App.GetID()), int(evt.Installation.GetID()))
		itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, int(evt.CheckSuite.App.GetID()), int(evt.Installation.GetID()), "kyroy-s-testapp.2018-07-28.private-key.pem")
		if err != nil {
			logrus.Errorf("failed to read key: %v", err)
			w.WriteHeader(500)
			return
		}

		// Use installation transport with client.
		client := github.NewClient(&http.Client{Transport: itr})

		if err := handleSuite(client, evt); err != nil {
			logrus.Errorf("failed to handle suite: %v", err)
		}
	case "check_run":
		var evt github.CheckRunEvent
		if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
			logrus.Errorf("failed to unmarshal payload: %v", err)
			w.WriteHeader(500)
			return
		}

		fmt.Println("checking with", int(evt.CheckRun.App.GetID()), int(evt.Installation.GetID()))
		itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, int(evt.CheckRun.App.GetID()), int(evt.Installation.GetID()), "kyroy-s-testapp.2018-07-28.private-key.pem")
		if err != nil {
			logrus.Errorf("failed to read key: %v", err)
			w.WriteHeader(500)
			return
		}
		client := github.NewClient(&http.Client{Transport: itr})

		if err := handleRun(client, evt); err != nil {
			logrus.Errorf("failed to handle suite: %v", err)
		}
	default:
		logrus.Errorf("unknown event: %s", event)
	}
	w.WriteHeader(200)
}

func handleRun(client *github.Client, evt github.CheckRunEvent) error {
	logrus.Infof("check_run - status: %s, action: %s", evt.CheckRun.GetStatus(), evt.GetAction())
	if evt.GetAction() != "rerequested" && evt.CheckRun.GetStatus() != "queued" {
		return nil
	}

	config, err := config2.Download(client, evt.Repo.Owner.GetLogin(), evt.Repo.GetName(), evt.CheckRun.CheckSuite.GetHeadBranch())
	if err != nil {
		// TODO
		return fmt.Errorf("failed to download config: %v", err)
	}

	runID := evt.CheckRun.GetID()
	name := evt.CheckRun.GetName()

	switch evt.GetAction() {
	case "rerequested":
		if err := ratelimit.Request(evt.Repo.Owner.GetID()); err != nil {
			runID, err = github2.CreateCheckRun(client,
				evt.Repo.Owner.GetLogin(),
				evt.Repo.GetName(),
				evt.CheckRun.CheckSuite.GetHeadBranch(),
				evt.CheckRun.GetHeadSHA(),
				name,
				github2.Completed,
				github2.Failure,
				&github.CheckRunOutput{
					Title:   &name,                      // *
					Summary: github.String(err.Error()), // *
				})
			if err != nil {
				return fmt.Errorf("failed to create check_run for %s: %v", name, err)
			}
			return nil
		}
		runID, err = github2.CreateCheckRun(client,
			evt.Repo.Owner.GetLogin(),
			evt.Repo.GetName(),
			evt.CheckRun.CheckSuite.GetHeadBranch(),
			evt.CheckRun.GetHeadSHA(),
			name,
			github2.InProgress,
			github2.None,
			nil)
		if err != nil {
			return fmt.Errorf("failed to create check_run for %s: %v", name, err)
		}
	case "created":
		switch evt.CheckRun.GetStatus() {
		case "queued":
			if err := github2.UpdateCheckRun(client, evt.Repo.Owner.GetLogin(), evt.Repo.GetName(), runID, name, github2.InProgress, github2.None,
				&github.CheckRunOutput{
					Title:   &name,                    // *
					Summary: github.String("running"), // *
				}); err != nil {
				return fmt.Errorf("failed to set %d to in_progress: %v", runID, err)
			}
		default:
			return fmt.Errorf("unknown status %s for action \"created\"", evt.CheckRun.GetStatus())
		}

	default:
		return fmt.Errorf("unknown action %s", evt.GetAction())
	}

	// TODO use context!!!
	ctx, _ := context.WithTimeout(context.Background(), 15*time.Minute)
	go func(ctx context.Context) {
		results, message, err := golang.TestGoVersion(config, evt.Repo.GetCloneURL(), evt.CheckRun.GetHeadSHA(), name)
		if err != nil {
			logrus.Errorf("testGoRepo failed: %v", err)
			return
		}
		annotations, err := results.Annotations(evt.Repo.Owner.GetLogin(), evt.Repo.GetName(), evt.CheckRun.GetHeadSHA())
		if err != nil {
			logrus.Errorf("[%s] failed to create annotations: %v", name, err)
			return
		}
		conclusion := github2.Success
		if len(annotations) > 0 || message != "successful" {
			conclusion = github2.Failure
		}
		logrus.Debugf("[%d] %s: %s", runID, name, conclusion)
		err = github2.UpdateCheckRun(client, evt.Repo.Owner.GetLogin(), evt.Repo.GetName(), runID, name,
			github2.Completed,
			conclusion,
			&github.CheckRunOutput{
				Title:       &name,                                            // *
				Summary:     github.String("x succeed, x warnings, x errors"), // *
				Text:        &message,
				Annotations: annotations,
			},
		)
		if err != nil {
			logrus.Errorf("failed to update check_run %s: %v", runID, err)
		}
	}(ctx)
	return nil
}

func handleSuite(client *github.Client, evt github.CheckSuiteEvent) error {
	logrus.Infof("check_suite - status %s, action: %s", evt.CheckSuite.GetStatus(), evt.GetAction())
	switch evt.GetAction() {
	case "requested", "created":
		switch evt.CheckSuite.GetStatus() {
		case "queued":
			goto CreateSuite
		}
	case "rerequested":
		goto CreateSuite
	}
	return nil
CreateSuite:
	config, err := config2.Download(client, evt.Repo.Owner.GetLogin(), evt.Repo.GetName(), evt.CheckSuite.GetHeadBranch())
	if err != nil {
		// TODO
		return fmt.Errorf("failed to download config: %v", err)
	}

	for _, version := range config.Versions() {
		if err := ratelimit.Request(evt.Repo.Owner.GetID()); err != nil {
			_, err = github2.CreateCheckRun(client,
				evt.Repo.Owner.GetLogin(),
				evt.Repo.GetName(),
				evt.CheckSuite.GetHeadBranch(),
				evt.CheckSuite.GetHeadSHA(),
				version,
				github2.Completed,
				github2.Failure,
				&github.CheckRunOutput{
					Title:   &version,                   // *
					Summary: github.String(err.Error()), // *
				})
			if err != nil {
				logrus.Errorf("failed to create check_run for %s: %v", version, err)
			}
			continue
		}
		_, err := github2.CreateCheckRun(client,
			evt.Repo.Owner.GetLogin(),
			evt.Repo.GetName(),
			evt.CheckSuite.GetHeadBranch(),
			evt.CheckSuite.GetHeadSHA(),
			version,
			github2.Queued,
			github2.None,
			nil)
		if err != nil {
			logrus.Errorf("failed to create setup check_run for %s: %v", version, err)
		}
	}
	return nil
}
