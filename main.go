package main

import (
	"encoding/json"
		"github.com/google/go-github/github"
	"net/http"
	"github.com/sirupsen/logrus"
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

	//hook, err := githubhook.Parse(secret, r)
	//if err != nil {
	//	logrus.Errorf("failed to parse request: %v", err)
	//	w.WriteHeader(500)
	//	return
	//}
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("failed to read request: %v", err)
		w.WriteHeader(500)
		return
	}

	var evt github.PullRequestEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		logrus.Errorf("failed to unmarshal payload: %v", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
	logrus.Infof("succeeded", evt)
}
