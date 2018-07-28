package main

import (
	"encoding/json"
		"github.com/google/go-github/github"
	"net/http"
	"github.com/sirupsen/logrus"
	"fmt"
	"io/ioutil"
)

func main() {
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
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("failed to read body: %v", err)
		w.WriteHeader(500)
		return
	}
	fmt.Printf("check_suite: %s", body)
	w.WriteHeader(200)
}
