package main

import (
	"encoding/json"
		"github.com/google/go-github/github"
	"net/http"
	"github.com/sirupsen/logrus"
	"fmt"
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
	fmt.Println("evt.Repo.Name", *evt.Repo.Name)
	fmt.Println("evt.Repo.ID", *evt.Repo.ID)
	fmt.Println("evt.Repo.CloneURL", *evt.Repo.CloneURL)
	fmt.Println("evt.Repo.FullName", *evt.Repo.FullName)
	fmt.Println("evt.Repo.GitURL", *evt.Repo.GitURL)
	fmt.Println("evt.Repo.Language", *evt.Repo.Language)

}
