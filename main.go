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
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

func test() {
	fmt.Println("checking with", 15308, 262390)
	//itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, 15308, 262390, "kyroy-s-testapp.2018-07-28.private-key.pem")
	privateKey, err := ioutil.ReadFile("kyroy-s-testapp.2018-07-28.private-key.pem")
	if err != nil {
		logrus.Errorf("could not read private key: %s", err)
		return
	}
	fmt.Printf("privatekey: %s", privateKey)
	itr, err := ghinstallation.New(http.DefaultTransport, 15308, 262390, privateKey)
	if err != nil {
		logrus.Errorf("failed to read key: %v", err)
		return
	}
	client := github.NewClient(&http.Client{Transport: itr})

	a, b, c := client.Checks.GetCheckSuite(context.Background(), "Kyroy", "testrepo", 7719827)
	fmt.Println("aaa", a, b, c)
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	//setSuiteProgress("Kyroy", "testrepo", "Kyroy-patch-1", "11621bbd1f7ef7ab05156563fc3ab9d663b8a0c", 7719827)
	//setRunCompleted("Kyroy", "testrepo", "Kyroy-patch-1", "11621bbd1f7ef7ab05156563fc3ab9d663b8a0c", 9669759)
	//test()
	//fmt.Printf("\n\n\n")
	//testGoRepo()

	//results, messages, err := golang.TestGoRepo(&config2.Config{
	//	Language: "go",
	//	Versions: []string{"golang:1.10"},
	//	GoImportPath: "github.com/Kyroy/testrepo",
	//}, "https://github.com/Kyroy/testrepo.git", "eb041cb31ee1df478bba2194a48e0ce19b42e4e9")
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

	case "rerequested":
		_, err := github2.CreateCheckRun(client,
			evt.Repo.Owner.GetLogin(),
			evt.Repo.GetName(),
			evt.CheckRun.CheckSuite.GetHeadBranch(),
			evt.CheckRun.GetHeadSHA(),
			name,
			github2.InProgress)
		if err != nil {
			return fmt.Errorf("failed to create check_run for %s: %v", name, err)
		}
	default:
		return fmt.Errorf("unknown action %s", evt.GetAction())
	}

	// TODO use context!!!
	ctx, _ := context.WithTimeout(context.Background(), 15*time.Minute)
	go func(ctx context.Context) {
		time.Sleep(30 * time.Second)
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
	case "created":
		switch evt.CheckSuite.GetStatus() {
		case "queued":
		default:
			return nil
		}
	case "rerequested":
		//switch evt.CheckSuite.GetStatus() {
		//
		//}
	default:
		return nil
	}
	//if evt.CheckSuite.GetStatus() != "queued" || evt.GetAction() != "created" {
	//	return nil
	//}

	config, err := config2.Download(client, evt.Repo.Owner.GetLogin(), evt.Repo.GetName(), evt.CheckSuite.GetHeadBranch())
	if err != nil {
		// TODO
		return fmt.Errorf("failed to download config: %v", err)
	}

	for _, version := range config.Versions() {
		_, err := github2.CreateCheckRun(client,
			evt.Repo.Owner.GetLogin(),
			evt.Repo.GetName(),
			evt.CheckSuite.GetHeadBranch(),
			evt.CheckSuite.GetHeadSHA(),
			version,
			github2.Queued)
		if err != nil {
			logrus.Errorf("failed to create setup check_run for %s: %v", version, err)
			continue
		}
	}
	return nil
}
