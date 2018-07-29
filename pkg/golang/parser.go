package golang

import (
	"bytes"
	"fmt"
	"github.com/kyroy/github-app/pkg/tests"
	"github.com/sirupsen/logrus"
	"strings"
)

func parseGoTest(lines [][]byte, i int, importPath string) ([]*tests.Result, int) {
	j := i
	for ; j < len(lines); j++ {
		if bytes.HasPrefix(lines[j], []byte("### ")) {
			break
		}
	}
	return parseTestLog(lines[i:j], importPath), j - 1
}

func parseTestLog(lines [][]byte, importPath string) []*tests.Result {
	var results []*tests.Result
	var tmpResults []*tests.Result
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		switch {
		case startsWith(line, "--- FAIL"):
			if i+1 >= len(lines) || startsWith(lines[i+1], "---") {
				// inner test failed
				continue
			}
			suite := string(bytes.Split(trimStart(line, "--- FAIL:"), []byte{' '})[0])
			results := re.FindSubmatch(lines[i+1])
			res, err := newTestResult(suite, results)
			if err != nil {
				fmt.Printf("ERR failed to parse %q, got %v: %v\n", lines[i+1], res, err)
				continue
			}
			j := findNext(lines, i+1, "--- FAIL", "--- PASS", "=== RUN", "FAIL")

			var msgs [][]byte
			if res.Message != "" {
				msgs = append(msgs, []byte(res.Message))
			}
			if i+2 < len(lines) && i+1 < j {
				msgs = append(msgs, lines[i+2:j]...)
			}
			for i, m := range msgs {
				msgs[i] = bytes.TrimSpace(m)
			}
			res.Message = string(bytes.Join(msgs, []byte{'\n'}))
			tmpResults = append(tmpResults, res)

			i = j - 1
		case startsWith(line, "FAIL\t"):
			b := bytes.Split(line, []byte{'\t'})
			if len(b) < 2 {
				tmpResults = make([]*tests.Result, 0)
				logrus.Errorf("failed to get FAIL suite: %s", line)
			}
			suite := string(b[1])
			for _, res := range tmpResults {
				res.File = fmt.Sprintf("%s%s", buildFilePathPrefix(suite, importPath), res.File)
				results = append(results, res)
			}
			tmpResults = make([]*tests.Result, 0)
		//case startsWith(line, "=== RUN"), startsWith(line, "--- PASS"), startsWith(line, "PASS"), startsWith(line, "ok"):
		default:
			//fmt.Println(i, "unknown line", string(line))
		}
	}
	return results
}

func findNext(lines [][]byte, i int, matcher ...string) int {
	for ; i < len(lines); i++ {
		for _, m := range matcher {
			if startsWith(lines[i], m) {
				return i
			}
		}
	}
	return i
}

func startsWith(line []byte, prefix string) bool {
	return bytes.HasPrefix(bytes.TrimSpace(line), []byte(prefix))
}

func trimStart(line []byte, prefix string) []byte {
	return bytes.TrimSpace(bytes.TrimPrefix(bytes.TrimSpace(line), []byte(prefix)))
}

func buildFilePathPrefix(suiteName, importPath string) string {
	s := strings.TrimPrefix(strings.TrimPrefix(suiteName, importPath), "/")
	if s == "" {
		return s
	}
	return s + "/"
}
