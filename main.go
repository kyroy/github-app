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
	test()
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
	case "pull_request":
		pullRequest(w, r)
		return
	case "check_suite":
		check_suite(w, r)
		return
	default:
		logrus.Errorf("unknown event: %s", event)
	}
	w.WriteHeader(400)
}

func pullRequest(w http.ResponseWriter, r *http.Request) {
	var evt github.PullRequestEvent
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		logrus.Errorf("failed to unmarshal payload: %v", err)
		w.WriteHeader(500)
		return
	}
	if evt.PullRequest == nil || evt.PullRequest.Head == nil {
		logrus.Errorf("missing data in pull request: %v", evt.PullRequest)
		w.WriteHeader(500)
		return
	}
	fmt.Println("evt.PullRequest.Head.Label", *evt.PullRequest.Head.Label)
	fmt.Println("evt.PullRequest.Head.Ref", *evt.PullRequest.Head.Ref)
	fmt.Println("evt.PullRequest.Head.SHA", *evt.PullRequest.Head.SHA)
	if evt.Repo != nil {
		fmt.Println("evt.Repo.Name", *evt.Repo.Name)
		fmt.Println("evt.Repo.ID", *evt.Repo.ID)
		fmt.Println("evt.Repo.CloneURL", *evt.Repo.CloneURL)
		fmt.Println("evt.Repo.FullName", *evt.Repo.FullName)
		fmt.Println("evt.Repo.GitURL", *evt.Repo.GitURL)
	}
	w.WriteHeader(200)
}

func check_suite(w http.ResponseWriter, r *http.Request) {
	//body, err := ioutil.ReadAll(r.Body)
	//if err != nil {
	//	logrus.Errorf("failed to read body: %v", err)
	//	w.WriteHeader(500)
	//	return
	//}
	//fmt.Printf("check_suite: %s", body)
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
		checkRun, response, err := client.Checks.CreateCheckRun(context.Background(), evt.Repo.Owner.GetLogin(), evt.Repo.GetName(), github.CreateCheckRunOptions{
			Name: "first suite", // *
			HeadBranch: evt.CheckSuite.GetHeadBranch(), // *
			HeadSHA: evt.CheckSuite.GetHeadSHA(), // *
			Status: github.String("in_progress"),
		})
		fmt.Println("checkRun", checkRun)
		fmt.Println("response", response)
		fmt.Println("err", err)
	//case "in_progress":
	//case "completed":
		// output
	default:
		fmt.Printf("unhandled check suite status: %s", status)
	}



	w.WriteHeader(200)
}
