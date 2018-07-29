package golang

import (
	"bytes"
	"fmt"
	"github.com/ahmetalpbalkan/dexec"
	"github.com/fsouza/go-dockerclient"
	"github.com/kyroy/github-app/pkg/config"
	"github.com/kyroy/github-app/pkg/tests"
	"github.com/sirupsen/logrus"
	"github.com/tebeka/go2xunit/lib"
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
//func TestGoRepo(config *config.Config, URL, commit string) (tests.Results, map[string]string, error) {
//	commands := []string{
//		"go version",
//		fmt.Sprintf("mkdir -p $GOPATH/src/%s", config.GoImportPath()),
//		fmt.Sprintf("cd $GOPATH/src/%s", config.GoImportPath()),
//		fmt.Sprintf("git clone -q %s .", URL),
//		fmt.Sprintf("git checkout -q %s", commit),
//		"echo '### setup'",
//	}
//	commands = append(commands, config.SetupCommands()...)
//	for stage, cmds := range config.TestCommands() {
//		commands = append(commands, fmt.Sprintf("echo '### %s'", stage))
//		commands = append(commands, cmds...)
//	}
//
//	cl, err := docker.NewClient("unix:///var/run/docker.sock")
//	if err != nil {
//		logrus.Errorf("failed to create docker client: %v", err)
//		return nil, nil, fmt.Errorf("internal server error")
//	}
//	d := dexec.Docker{Client: cl}
//
//	result := make(tests.Results)
//	messages := make(map[string]string)
//	for i, version := range config.Versions() {
//		if err := cl.PullImage(docker.PullImageOptions{Repository: config.DockerImage(), Tag: config.Tags()[i]}, docker.AuthConfiguration{}); err != nil {
//			logrus.Errorf("failed to pull image %s: %v", version, err)
//		}
//		result[version], messages[version] = testGoVersion(&d, version, commands)
//	}
//	return result, messages, nil
//}

func TestGoVersion(config *config.Config, URL, commit, image string) (tests.StageResults, string, error) {
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
		return nil, "", fmt.Errorf("internal server error")
	}
	d := dexec.Docker{Client: cl}

	s := strings.Split(image, ":")
	if len(s) != 2 {
		return nil, "", fmt.Errorf("failed to parse image tag for %s", image)
	}
	tag := s[1]
	if err := cl.PullImage(docker.PullImageOptions{Repository: config.DockerImage(), Tag: tag}, docker.AuthConfiguration{}); err != nil {
		logrus.Errorf("failed to pull image %s:%s: %v", config.DockerImage(), tag, err)
	}
	stageResults, message := testGoVersion(&d, image, config.GoImportPath(), commands)

	return stageResults, message, nil
}

//
// stage -> results, message
func testGoVersion(d *dexec.Docker, image, importPath string, commands []string) (map[string][]*tests.Result, string) {
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
	return parseTestResults(b, importPath), msg
}

func parseTestResults(testLog []byte, importPath string) map[string][]*tests.Result {
	findings := make(map[string][]*tests.Result)
	lines := bytes.Split(testLog, []byte{'\n'})
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if bytes.HasPrefix(line, []byte("### go test")) {
			findings["go test"], i = parseGoTest(lines, i+1, importPath)
		} else if bytes.HasPrefix(line, []byte("### ")) {
			stage := string(bytes.TrimPrefix(line, []byte("### ")))
			findings[stage], i = parseStage(lines, i+1)
		}
	}
	return findings
}

func parseGoTest(lines [][]byte, i int, importPath string) ([]*tests.Result, int) {
	j := i
	for ; j < len(lines); j++ {
		if bytes.HasPrefix(lines[j], []byte("### ")) {
			break
		}
	}
	var results []*tests.Result
	suites, err := lib.ParseGotest(bytes.NewReader(bytes.Join(lines[i:j], []byte{'\n'})), "")
	if err == nil {
		for a, suite := range suites {
			fmt.Printf("%d suite %s, %s, %s\n", a, suite.Name, suite.Status, suite.Time)
			for b, test := range suite.Tests {
				fmt.Printf("  %d test %s, %v, %s, %s\n", b, test.Name, test.Status, test.Time, test.Message)

				reResults := re.FindSubmatch([]byte(fmt.Sprintf("%s/%s", strings.TrimPrefix(suite.Name, importPath+"/"), strings.TrimSpace(test.Message))))
				res, err := newTestResult(reResults)
				if err != nil {
					continue
				}
				res.Message = fmt.Sprintf("%s %s: %s", suite.Name, test.Name, res.Message)
				results = append(results, res)
			}
		}
	}
	return results, j - 1
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
