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
		"github.com/ahmetalpbalkan/dexec"
	"github.com/fsouza/go-dockerclient"
	"strings"
	"bytes"
	"time"
	"regexp"
	"strconv"
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

func testGoRepo(coURL, projPath, commit string) ([]*TestResult, error) {
	//coURL := "https://github.com/Kyroy/testrepo.git"
	//commit := "11621bbd1f7ef7ab05156563fc3ab9d663b8a0c"
	//projPath := "github.com/Kyroy/testrepo"
	commands := []string{"go version"}
	commands = append(commands,
		"go get -u golang.org/x/lint/golint",
		fmt.Sprintf("mkdir -p $GOPATH/src/%s", projPath),
		fmt.Sprintf("cd $GOPATH/src/%s", projPath),
		fmt.Sprintf("git clone -q %s .", coURL),
		fmt.Sprintf("git checkout -q %s", commit),
		"echo '### go fmt'",
		"go fmt ./...",
		"echo '### golint'",
		fmt.Sprintf(`golint $(go list ./...) | sed 's/'$(echo $GOPATH/src/%s/ | sed 's/\//\\\//g')'//g'`, projPath),
		"echo '### go test'",
		"go test -v ./...")

	cl, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		logrus.Errorf("failed to create docker client: %v", err)
		return nil, fmt.Errorf("failed to create docker client")
	}
	d := dexec.Docker{Client: cl}

	m, err := dexec.ByCreatingContainer(docker.CreateContainerOptions{
		Config: &docker.Config{Image: "golang:1.10"},
	})
	if err != nil {
		logrus.Errorf("failed to dexec.ByCreatingContainer: %v", err)
		return nil, fmt.Errorf("failed to dexec.ByCreatingContainer")
	}

	cmd := d.Command(m, "/bin/bash", "-c", strings.Join(commands, " && "))
	var ba bytes.Buffer
	cmd.Stderr = &ba
	b, err := cmd.Output()
	if err != nil {
		msg := ba.String()
		if msg != "" {
			msg = " - " + msg
		}
		logrus.Errorf("command failed: %s%s", strings.TrimPrefix(err.Error(), "dexec: "), msg)
	}
	fmt.Printf("%s\n", b)

	results := parseTestResults(projPath, b)
	for _, res := range results {
		fmt.Printf("- %v\n", res)
	}
	return results, nil
}

type TestResult struct {
	File string
	Line *int
	Message string
}

func (r *TestResult) Valid() bool {
	return r.File != "" && r.Line != nil && r.Message != ""
}

func NewTestResult(results [][]byte, names []string) (*TestResult, error) {
	found := &TestResult{}
	if len(results) > 0 {
		for i, res := range results {
			switch names[i] {
			case "file":
				found.File = string(res)
			case "line":
				l, _ := strconv.Atoi(string(res))
				found.Line = &l
			case "message":
				found.Message = string(res)
			}
		}
	}
	if !found.Valid() {
		return nil, fmt.Errorf("TestResult invalid: %v", found)
	}
	return found, nil
}

func parseTestResults(repo string, res []byte) []*TestResult {
	var findings []*TestResult
	lines := bytes.Split(res, []byte{'\n'})
	goLintRe := regexp.MustCompile(`(?P<file>.+\.go):(?P<line>\d+):(?P<col>\d+): (?P<message>.+)`)
	goLintNames := goLintRe.SubexpNames()
	goTestRe := regexp.MustCompile(`\s*(?P<file>.+\.go):(?P<line>\d+): (?P<message>.+)`)
	goTestNames := goTestRe.SubexpNames()
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		switch {
		case bytes.HasPrefix(line, []byte("### golint")):
			for i++; i < len(lines); i++ {
				line = lines[i]
				if bytes.HasPrefix(line, []byte("### ")) {
					i--
					break
				}
				results := goLintRe.FindSubmatch(line)
				if res, err := NewTestResult(results, goLintNames); err == nil {
					findings = append(findings, res)
				}
			}
		case bytes.HasPrefix(line, []byte("### go test")):
			for i++; i < len(lines); i++ {
				line = lines[i]
				if bytes.HasPrefix(line, []byte("### ")) {
					i--
					break
				}
				if bytes.HasPrefix(line, []byte("=== ")) || bytes.HasPrefix(line, []byte("--- ")) {
					continue
				}
				results := goTestRe.FindSubmatch(line)
				if res, err := NewTestResult(results, goTestNames); err == nil {
					findings = append(findings, res)
				}
			}
		}
	}
	return findings
}

func main() {
	//setSuiteProgress("Kyroy", "testrepo", "Kyroy-patch-1", "11621bbd1f7ef7ab05156563fc3ab9d663b8a0c", 7719827)
	//setRunCompleted("Kyroy", "testrepo", "Kyroy-patch-1", "11621bbd1f7ef7ab05156563fc3ab9d663b8a0c", 9669759)
	//test()
	//fmt.Printf("\n\n\n")
	//testGoRepo()
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
		handleSuite(w, r)
		return
	default:
		logrus.Errorf("unknown event: %s", event)
	}
	w.WriteHeader(400)
}

func setSuiteProgress(client *github.Client, owner, repo, branch, sha string) (int64, error) {
	//itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, 15308, 262390, "kyroy-s-testapp.2018-07-28.private-key.pem")
	//if err != nil {
	//	logrus.Errorf("failed to read key: %v", err)
	//	return
	//}
	//
	//// Use installation transport with client.
	//client := github.NewClient(&http.Client{Transport: itr})
	checkRun, response, err := client.Checks.CreateCheckRun(context.Background(), owner, repo, github.CreateCheckRunOptions{
		Name: "first suite", // *
		HeadBranch: branch, // *
		HeadSHA: sha, // *
		Status: github.String("in_progress"),
	})
	fmt.Println("setSuiteProgress", checkRun, response, err)
	if err != nil {
		return 0, fmt.Errorf("CreateCheckRun failed: %v", err)
	}
	if checkRun.ID == nil {
		return 0, fmt.Errorf("checkRun.ID not set")
	}
	return checkRun.GetID(), nil
}

func setRunCompleted(client *github.Client, owner, repo, branch, sha string, checkRunID int64, results []*TestResult) {
	//itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, 15308, 262390, "kyroy-s-testapp.2018-07-28.private-key.pem")
	//if err != nil {
	//	logrus.Errorf("failed to read key: %v", err)
	//	return
	//}
	//
	//// Use installation transport with client.
	//client := github.NewClient(&http.Client{Transport: itr})
	if len(results) == 0 {
		checkRun, response, err := client.Checks.UpdateCheckRun(context.Background(), owner, repo, checkRunID, github.UpdateCheckRunOptions{
			CompletedAt: &github.Timestamp{Time: time.Now()},
			Conclusion:  github.String("success"), // success, failure, neutral, cancelled, timed_out, or action_required
		})
		fmt.Println("[success] setSuiteProgress", checkRun, response, err)
		return
	}

	var annotations []*github.CheckRunAnnotation
	for _, res := range results {
		annotations = append(annotations, &github.CheckRunAnnotation{
			FileName: &res.File, // *
			BlobHRef: github.String(fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, sha, res.File)), // *
			StartLine: res.Line, // *
			EndLine: res.Line, // *
			WarningLevel: github.String("failure"),// * notice, warning, failure
			Message: &res.Message, // *
		})
	}

	checkRun, response, err := client.Checks.UpdateCheckRun(context.Background(), owner, repo, checkRunID, github.UpdateCheckRunOptions{
		CompletedAt: &github.Timestamp{Time: time.Now()},
		Conclusion: github.String("failure"), // success, failure, neutral, cancelled, timed_out, or action_required
		Output: &github.CheckRunOutput{
			Title: github.String("Test Output Title"), // *
			Summary: github.String("test Output Summary"), // *
			Annotations: annotations,
			//Annotations: []*github.CheckRunAnnotation{
			//	{
			//		FileName:github.String("x_test.go"), // *
			//		BlobHRef: github.String(fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, sha, "x_test.go")), // *
			//		StartLine: github.Int(9), // *
			//		EndLine: github.Int(9), // *
			//		WarningLevel: github.String("warning"),// * notice, warning, failure
			//		Message: github.String("don't use underscores in Go names; const x_y_x should be xYX"), // *
			//	},
			//},
		},
	})
	fmt.Println("[failure] setSuiteProgress", checkRun, response, err)
}

func handleSuite(w http.ResponseWriter, r *http.Request) {
	var evt github.CheckSuiteEvent
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		logrus.Errorf("failed to unmarshal payload: %v", err)
		w.WriteHeader(500)
		return
	}
	fmt.Printf("evt.CheckSuite.ID %v\n", *evt.CheckSuite.ID)
	fmt.Printf("evt.CheckSuite.App %v\n", *evt.CheckSuite.App)
	fmt.Printf("evt.CheckSuite.HeadBranch %v\n", *evt.CheckSuite.HeadBranch)
	fmt.Printf("evt.CheckSuite.HeadSHA %v\n", *evt.CheckSuite.HeadSHA)
	fmt.Printf("evt.CheckSuite.PullRequests %v\n", evt.CheckSuite.PullRequests)
	fmt.Printf("evt.CheckSuite.Status %v\n", *evt.CheckSuite.Status)
	fmt.Printf("evt.CheckSuite.URL %v\n", *evt.CheckSuite.URL)
	if evt.CheckSuite.Conclusion != nil {
		fmt.Printf("evt.CheckSuite.Conclusion %v\n", *evt.CheckSuite.Conclusion)
	}
	if evt.CheckSuite.Repository != nil {
		fmt.Printf("evt.CheckSuite.Repository %v\n", *evt.CheckSuite.Repository)
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

	token, response, err := client.Apps.CreateInstallationToken(context.Background(), evt.Installation.GetID())
	fmt.Println("CreateInstallationToken", token, response, err)

	installations, response, err := client.Apps.ListInstallations(context.Background(), &github.ListOptions{})
	fmt.Println("ListInstallations", installations, response, err)

	switch status := evt.CheckSuite.GetStatus(); status {
	case "queued":
		runID, err := setSuiteProgress(client, evt.Repo.Owner.GetLogin(), evt.Repo.GetName(), evt.CheckSuite.GetHeadBranch(), evt.CheckSuite.GetHeadSHA())
		if err != nil {
			w.WriteHeader(500)
			return
		}

		go func() {
			results, err := testGoRepo(evt.Repo.GetCloneURL(), strings.TrimPrefix(evt.Repo.GetHTMLURL(), "https://"), evt.CheckSuite.GetHeadSHA())
			if err != nil {
				logrus.Errorf("testGoRepo failed: %v", err)
				return
			}
			setRunCompleted(client, evt.Repo.Owner.GetLogin(), evt.Repo.GetName(), evt.CheckSuite.GetHeadBranch(), evt.CheckSuite.GetHeadSHA(), runID, results)
		}()


	//case "in_progress":
	//case "completed":
		// output
	default:
		fmt.Printf("unhandled check suite status: %s", status)
	}

	w.WriteHeader(200)
}
