package golang

import (
	"bytes"
	"fmt"
	"github.com/ahmetalpbalkan/dexec"
	"github.com/fsouza/go-dockerclient"
	"github.com/kyroy/github-app/pkg/config"
	"github.com/kyroy/github-app/pkg/tests"
	"github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
)

var (
	re      = regexp.MustCompile(`^\s*(?P<file>.+\.go):(?P<line>\d+):(?P<col>\d+)?:? (?P<message>.+)$`)
	reNames = re.SubexpNames()
)

//
// version -> stage -> results, version -> message, error
func TestGoRepo(config *config.Config, URL, commit string) (tests.Results, map[string]string, error) {
	commands := []string{
		"go version",
		fmt.Sprintf("mkdir -p $GOPATH/src/%s", config.GoImportPath()),
		fmt.Sprintf("cd $GOPATH/src/%s", config.GoImportPath()),
		fmt.Sprintf("git clone -q %s .", URL),
		fmt.Sprintf("git checkout -q %s", commit),
		"echo '### setup'",
	}
	commands = append(commands, config.SetupCommands()...)
	for stage, cmds := range config.TestCommands() {
		commands = append(commands, fmt.Sprintf("echo '### %s'", stage))
		commands = append(commands, cmds...)
	}

	cl, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		logrus.Errorf("failed to create docker client: %v", err)
		return nil, nil, fmt.Errorf("internal server error")
	}
	d := dexec.Docker{Client: cl}

	result := make(tests.Results)
	messages := make(map[string]string)
	for i, version := range config.Versions() {
		if err := cl.PullImage(docker.PullImageOptions{Repository: config.DockerImage(), Tag: config.Tags()[i]}, docker.AuthConfiguration{}); err != nil {
			logrus.Errorf("failed to pull image %s: %v", version, err)
		}
		result[version], messages[version] = testGoVersion(&d, version, commands)
	}
	return result, messages, nil
}

//
// stage -> results, message
func testGoVersion(d *dexec.Docker, image string, commands []string) (map[string][]*tests.Result, string) {
	m, err := dexec.ByCreatingContainer(docker.CreateContainerOptions{
		Config: &docker.Config{Image: image},
	})
	if err != nil {
		logrus.Errorf("failed to dexec.ByCreatingContainer: %v", err)
		return nil, "internal server error"
	}

	cmd := d.Command(m, "/bin/bash", "-c", strings.Join(commands, " && "))
	var ba bytes.Buffer
	cmd.Stderr = &ba
	b, err := cmd.Output()
	msg := "successful"
	if err != nil {
		msg = ba.String()
		if msg != "" {
			msg = " - " + msg
		}
		msg = fmt.Sprintf("execution failed with: %s%s", strings.TrimPrefix(err.Error(), "dexec: "), msg)
	}
	logrus.Infof("[%s] %s", image, msg)
	if logrus.GetLevel() == logrus.DebugLevel {
		fmt.Printf("[%s] testLog ----------------\n%s\n----------------\n", image, b)
	}
	return parseTestResults(b), msg
}

func parseTestResults(testLog []byte) map[string][]*tests.Result {
	findings := make(map[string][]*tests.Result)
	lines := bytes.Split(testLog, []byte{'\n'})
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if bytes.HasPrefix(line, []byte("### ")) {
			stage := string(bytes.TrimPrefix(line, []byte("### ")))
			findings[stage], i = parseStage(lines, i+1)
		}
	}
	return findings
}

func parseStage(lines [][]byte, i int) ([]*tests.Result, int) {
	var findings []*tests.Result
	for ; i < len(lines); i++ {
		line := lines[i]
		if bytes.HasPrefix(line, []byte("### ")) {
			i--
			break
		}
		results := re.FindSubmatch(line)
		res, err := newTestResult(results)
		if err != nil {
			continue
		}
		findings = append(findings, res)
	}
	return findings, i
}

func newTestResult(results [][]byte) (*tests.Result, error) {
	found := &tests.Result{}
	if len(results) > 0 {
		for i, res := range results {
			switch reNames[i] {
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
		return found, fmt.Errorf("invalid")
	}
	return found, nil
}
