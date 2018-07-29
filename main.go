package main

import (
	"encoding/json"
	"github.com/google/go-github/github"
	"net/http"
	"github.com/sirupsen/logrus"
	"fmt"
	"context"
	"github.com/bradleyfalzon/ghinstallation"
	"io/ioutil"
	"time"
	"github.com/kyroy/github-app/pkg/golang"
	config2 "github.com/kyroy/github-app/pkg/config"
	github2 "github.com/kyroy/github-app/pkg/github"
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
	default:
		logrus.Errorf("unknown event: %s", event)
	}
	w.WriteHeader(400)
}

func handleSuite(client *github.Client, evt github.CheckSuiteEvent) error {
	if evt.CheckSuite.GetStatus() != "queued" {
		logrus.Infof("unhandled check suite status: %s", evt.CheckSuite.GetStatus())
		return nil
	}

	config, err := config2.Download(client, evt.Repo.Owner.GetLogin(), evt.Repo.GetName(), evt.CheckSuite.GetHeadBranch())
	if err != nil {
		// TODO
		return fmt.Errorf("failed to download config: %v", err)
	}

	runIDs := make(map[string]map[string]int64)
	for _, version := range config.Versions() {
		runIDs[version] = make(map[string]int64)
		d, err := github2.CreateCheckRun(client,
			evt.Repo.Owner.GetLogin(),
			evt.Repo.GetName(),
			evt.CheckSuite.GetHeadBranch(),
			evt.CheckSuite.GetHeadSHA(),
			fmt.Sprintf("%s: setup", version))
		if err != nil {
			logrus.Errorf("failed to create setup check_run: %v", err)
			continue
		}
		runIDs[version]["setup"] = d
		for stage := range config.TestCommands() {
			d, err = github2.CreateCheckRun(client,
				evt.Repo.Owner.GetLogin(),
				evt.Repo.GetName(),
				evt.CheckSuite.GetHeadBranch(),
				evt.CheckSuite.GetHeadSHA(),
				fmt.Sprintf("%s: %s", version, stage))
			if err != nil {
				logrus.Errorf("failed to create %s check_run: %v", stage, err)
				continue
			}
			runIDs[version][stage] = d
		}
	}

	// TODO use context!!!
	ctx, _ := context.WithTimeout(context.Background(), 15 * time.Minute)
	go func(ctx context.Context) {
		results, messages, err := golang.TestGoRepo(config, evt.Repo.GetCloneURL(), evt.CheckSuite.GetHeadSHA())
		if err != nil {
			logrus.Errorf("testGoRepo failed: %v", err)
			return
		}
		logrus.Infof("results", results)
		logrus.Infof("messages", messages)
		for version, runStageIDs := range runIDs {
			for stage, runID := range runStageIDs {
				annotations, err := results.Annotations(version, stage, evt.Repo.Owner.GetLogin(), evt.Repo.GetName(), evt.CheckSuite.GetHeadSHA())
				if err != nil {
					logrus.Errorf("failed to create annotations: %v", err)
					continue
				}
				conclusion := github2.Success
				if len(annotations) > 0 {
					conclusion = github2.Failure
				}
				logrus.Infof("%s %s: %s", version, stage, conclusion)
				err = github2.UpdateCheckRun(client,  evt.Repo.Owner.GetLogin(), evt.Repo.GetName(), runID, conclusion,
					fmt.Sprintf("%s: %s", version, stage), // title,
					messages[version], // summary
					github.String("beautiful test"), // text
					annotations)
				if err != nil {
					logrus.Errorf("failed to update check_run %s: %v", runID, err)
				}
			}
		}
	}(ctx)
	return nil
}
