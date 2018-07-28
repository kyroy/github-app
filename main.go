package main

import (
	"encoding/json"
	"github.com/rjz/githubhook"
	"github.com/google/go-github/github"
	"net/http"
	"github.com/sirupsen/logrus"
)

func main() {
	http.HandleFunc("/", handler)
	logrus.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	secret := []byte("don't tell!")
	hook, err := githubhook.Parse(secret, r)
	if err != nil {
		logrus.Errorf("failed to parse request: %v", err)
		w.WriteHeader(500)
		return
	}

	var evt github.PullRequestEvent
	if err := json.Unmarshal(hook.Payload, &evt); err != nil {
		logrus.Errorf("failed to unmarshal payload: %v", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
	logrus.Infof("succeeded", evt)
}
