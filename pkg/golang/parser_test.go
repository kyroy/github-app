package golang

import (
	"bytes"
	"testing"
)

const goTestLog = `=== RUN   TestA
--- FAIL: TestA (0.00s)
	x_test.go:10: failing cause I can
=== RUN   TestB
succeeding
--- PASS: TestB (0.00s)
=== RUN   TestC
=== RUN   TestC/hello
=== RUN   TestC/world
=== RUN   TestC/foo
=== RUN   TestC/bar
--- FAIL: TestC (0.00s)
    --- PASS: TestC/hello (0.00s)
    --- PASS: TestC/world (0.00s)
    --- PASS: TestC/foo (0.00s)
    --- FAIL: TestC/bar (0.00s)
    	x_test.go:29:
    			Error Trace:	x_test.go:29
    			Error:      	"bar" does not contain "o"
    			Test:       	TestC/bar
=== RUN   TestD
--- FAIL: TestD (0.00s)
	x_test.go:35:
			Error Trace:	x_test.go:35
			Error:      	Not equal:
			            	expected: "blasdasd"
			            	actual  : "adsnivrwpi"

			            	Diff:
			            	--- Expected
			            	+++ Actual
			            	@@ -1 +1 @@
			            	-blasdasd
			            	+adsnivrwpi
			Test:       	TestD
FAIL
FAIL	github.com/kyroy/testrepo	0.017s
=== RUN   TestC
=== RUN   TestC/hello
=== RUN   TestC/world
=== RUN   TestC/foo
=== RUN   TestC/bar
--- FAIL: TestC (0.00s)
    --- PASS: TestC/hello (0.00s)
    --- PASS: TestC/world (0.00s)
    --- PASS: TestC/foo (0.00s)
    --- FAIL: TestC/bar (0.00s)
    	x_test.go:20:
    			Error Trace:	x_test.go:20
    			Error:      	"bar" does not contain "o"
    			Test:       	TestC/bar
FAIL
FAIL	github.com/kyroy/testrepo/package	0.021s
=== RUN   TestS
=== RUN   TestS/a
=== RUN   TestS/b
=== RUN   TestS/c
=== RUN   TestS/d
--- PASS: TestS (288.01s)
    --- PASS: TestS/a (72.01s)
    --- PASS: TestS/b (72.00s)
    --- PASS: TestS/c (72.00s)
    --- PASS: TestS/d (72.00s)
PASS
ok  	github.com/kyroy/testrepo/success	288.031s
?   	github.com/kyroy/testrepo/vendor	[no test files]`

func TestParseTestLog(t *testing.T) {
	parseTestLog(bytes.Split([]byte(goTestLog), []byte{'\n'}), "github.com/kyroy/testrepo")
}
