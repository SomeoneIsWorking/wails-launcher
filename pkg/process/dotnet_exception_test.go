package process

// Tests for exception log parsing — both a unit test that feeds the exact
// ASP.NET Core format we captured from the live service, and an integration
// test that hits the real /throw endpoint.

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ── Unit test: exact format emitted by ASP.NET Core DeveloperExceptionPage ──

// TestReadOutput_ExceptionBlock verifies that the multi-line exception block
// emitted by ASP.NET Core (fail: header + indented exception + stack trace)
// is buffered into a SINGLE ERR log entry, and that the info: entry that
// immediately follows it is a separate entry with INF level.
func TestReadOutput_ExceptionBlock(t *testing.T) {
	ds := newTestService()

	// Exact format observed from .NET 10 / DeveloperExceptionPageMiddleware.
	// The fail: line starts the buffer; all following indented lines
	// (exception class + stack frames) are treated as continuations.
	// The trailing info: line is a new log-start that flushes the exception.
	feedLines(ds, "stdout", []string{
		`fail: Microsoft.AspNetCore.Diagnostics.DeveloperExceptionPageMiddleware[1]`,
		`      An unhandled exception has occurred while executing the request.`,
		`      System.InvalidOperationException: Intentional test exception from /throw endpoint`,
		`         at Program.<>c.<<Main>$>b__0_1(HttpContext ctx) in /app/Program.cs:line 8`,
		`         at Microsoft.AspNetCore.Routing.EndpointMiddleware.Invoke(HttpContext httpContext)`,
		`         at Microsoft.AspNetCore.Diagnostics.DeveloperExceptionPageMiddlewareImpl.Invoke(HttpContext context)`,
		`info: Microsoft.AspNetCore.Hosting.Diagnostics[2]`,
		`      Request finished HTTP/1.1 GET http://localhost:5199/throw - 500 - 12.34ms`,
	})

	logs := drainLogs(ds)
	if len(logs) != 2 {
		t.Fatalf("got %d log entries, want 2 (exception block + request-finished)", len(logs))
	}

	exc := logs[0]
	fin := logs[1]

	// Exception entry must be ERR level.
	if exc.Level != Err {
		t.Errorf("exception entry level = %q, want ERR", exc.Level)
	}

	// Exception entry must contain the exception class and message.
	if !strings.Contains(exc.Message, "System.InvalidOperationException") {
		t.Errorf("exception entry missing exception class:\n%s", exc.Message)
	}
	if !strings.Contains(exc.Message, "Intentional test exception from /throw endpoint") {
		t.Errorf("exception entry missing exception message:\n%s", exc.Message)
	}

	// Exception entry must contain at least one stack frame — confirming the
	// stack trace was NOT split off into a separate entry.
	if !strings.Contains(exc.Message, "at ") {
		t.Errorf("exception entry missing stack trace:\n%s", exc.Message)
	}

	// The request-finished entry must be a SEPARATE INF entry.
	if fin.Level != Inf {
		t.Errorf("request-finished entry level = %q, want INF", fin.Level)
	}
	if !strings.Contains(fin.Message, "Request finished") {
		t.Errorf("request-finished entry unexpected message:\n%s", fin.Message)
	}

	// The request-finished entry must NOT contain anything from the exception.
	if strings.Contains(fin.Message, "InvalidOperationException") {
		t.Errorf("request-finished entry has exception content blended in:\n%s", fin.Message)
	}
}

// TestReadOutput_ExceptionSurroundedByInfoLogs checks the full realistic
// sequence: several info: entries, then the exception block, then another
// info: entry — each must be its own entry.
func TestReadOutput_ExceptionSurroundedByInfoLogs(t *testing.T) {
	ds := newTestService()

	feedLines(ds, "stdout", []string{
		`info: Microsoft.AspNetCore.Hosting.Diagnostics[1]`,
		`      Request starting HTTP/1.1 GET http://localhost:5199/throw - - -`,
		`info: Microsoft.AspNetCore.Routing.EndpointMiddleware[0]`,
		`      Executing endpoint 'HTTP: GET /throw'`,
		`info: Microsoft.AspNetCore.Routing.EndpointMiddleware[1]`,
		`      Executed endpoint 'HTTP: GET /throw'`,
		`fail: Microsoft.AspNetCore.Diagnostics.DeveloperExceptionPageMiddleware[1]`,
		`      An unhandled exception has occurred while executing the request.`,
		`      System.InvalidOperationException: Intentional test exception from /throw endpoint`,
		`         at Program.<>c.<<Main>$>b__0_1(HttpContext ctx) in /app/Program.cs:line 8`,
		`         at Microsoft.AspNetCore.Routing.EndpointMiddleware.Invoke(HttpContext httpContext)`,
		`info: Microsoft.AspNetCore.Hosting.Diagnostics[2]`,
		`      Request finished HTTP/1.1 GET http://localhost:5199/throw - 500 - 12.34ms`,
	})

	logs := drainLogs(ds)
	if len(logs) != 5 {
		t.Fatalf("got %d entries, want 5 (3 info + 1 exception + 1 request-finished):\n%s",
			len(logs), formatEntries(logs))
	}

	// First three entries are all INF.
	for i, e := range logs[:3] {
		if e.Level != Inf {
			t.Errorf("entry[%d] level = %q, want INF", i, e.Level)
		}
	}

	// Fourth entry is the exception block — ERR, contains class + stack.
	exc := logs[3]
	if exc.Level != Err {
		t.Errorf("exception entry level = %q, want ERR", exc.Level)
	}
	if !strings.Contains(exc.Message, "System.InvalidOperationException") {
		t.Errorf("exception entry missing class:\n%s", exc.Message)
	}
	if !strings.Contains(exc.Message, "at ") {
		t.Errorf("exception entry missing stack frames:\n%s", exc.Message)
	}

	// The info: entries before the exception must NOT contain exception content.
	for i, e := range logs[:3] {
		if strings.Contains(e.Message, "InvalidOperationException") {
			t.Errorf("entry[%d] has exception content blended in:\n%s", i, e.Message)
		}
	}

	// Fifth entry is request-finished — INF, does not contain exception content.
	fin := logs[4]
	if fin.Level != Inf {
		t.Errorf("request-finished entry level = %q, want INF", fin.Level)
	}
	if strings.Contains(fin.Message, "InvalidOperationException") {
		t.Errorf("request-finished entry has exception content blended in:\n%s", fin.Message)
	}
}

// ── Integration test: live /throw endpoint ────────────────────────────────

// TestDotnetServiceExceptionParsing starts the real service, hits /throw, and
// verifies the exception log entry is parsed as a single ERR entry with the
// stack trace intact, while surrounding request-lifecycle entries remain separate.
func TestDotnetServiceExceptionParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping dotnet integration test in short mode")
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	svcPath := filepath.Join(filepath.Dir(thisFile), "testdata", "TestWebApi")

	svc := NewDotnetService(svcPath, ServiceEnv{
		"ASPNETCORE_URLS":        "http://localhost:5199",
		"ASPNETCORE_ENVIRONMENT": "Development",
	}, "")
	logChan, urlChan, statusChan := svc.GetChannels()

	if err := svc.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		svc.Stop() //nolint
		waitForStopped(logChan, urlChan, statusChan)
	})

	url := waitForURL(t, logChan, urlChan, statusChan)

	// Let all startup logs drain before we start asserting.
	time.Sleep(500 * time.Millisecond)
	for len(logChan) > 0 {
		<-logChan
	}

	// Hit /throw — expect a 500.
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(url + "/throw")
	if err != nil {
		t.Fatalf("GET /throw: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("GET /throw status = %d, want 500", resp.StatusCode)
	}

	// Collect log entries for up to 3 seconds.
	var logs []LogEntry
	deadline := time.After(3 * time.Second)
collect:
	for {
		select {
		case e := <-logChan:
			logs = append(logs, e)
		case <-deadline:
			break collect
		}
	}

	if len(logs) == 0 {
		t.Fatal("no log entries received after /throw")
	}

	// Find the ERR entry that contains the exception.
	excIdx := -1
	for i, e := range logs {
		if e.Level == Err && strings.Contains(e.Message, "InvalidOperationException") {
			excIdx = i
			break
		}
	}
	if excIdx == -1 {
		t.Fatalf("no ERR entry containing 'InvalidOperationException' found:\n%s", formatEntries(logs))
	}

	exc := logs[excIdx]

	// Stack trace must be inside the same entry.
	if !strings.Contains(exc.Message, "at ") {
		t.Errorf("exception entry missing stack frames (stack trace split into separate entry?):\n%s", exc.Message)
	}
	if !strings.Contains(exc.Message, "Intentional test exception from /throw endpoint") {
		t.Errorf("exception entry missing the thrown message:\n%s", exc.Message)
	}

	// Every other entry must not contain the exception class name.
	for i, e := range logs {
		if i == excIdx {
			continue
		}
		if strings.Contains(e.Message, "InvalidOperationException") {
			t.Errorf("entry[%d] (level=%s) has exception content blended in:\n%s", i, e.Level, e.Message)
		}
	}

	t.Logf("exception captured in entry[%d] (level=%s, %d chars, %d lines)",
		excIdx, exc.Level, len(exc.Message), strings.Count(exc.Message, "\n")+1)
}

// ── helpers ───────────────────────────────────────────────────────────────

func waitForURL(t *testing.T, logChan <-chan LogEntry, urlChan <-chan string, statusChan <-chan ServiceStatus) string {
	t.Helper()
	deadline := time.After(120 * time.Second)
	var lastURL string
	for {
		select {
		case s := <-statusChan:
			if s == Error {
				t.Fatal("service entered error state while waiting for URL")
			}
		case u := <-urlChan:
			lastURL = u
			if lastURL != "" {
				return lastURL
			}
		case <-logChan:
		case <-deadline:
			t.Fatalf("timeout waiting for service URL")
		}
	}
}

func waitForStopped(logChan <-chan LogEntry, urlChan <-chan string, statusChan <-chan ServiceStatus) {
	deadline := time.After(15 * time.Second)
	for {
		select {
		case s := <-statusChan:
			if s == Stopped {
				return
			}
		case <-logChan:
		case <-urlChan:
		case <-deadline:
			return
		}
	}
}

func formatEntries(entries []LogEntry) string {
	var sb strings.Builder
	for i, e := range entries {
		first := e.Message
		if idx := strings.Index(first, "\n"); idx != -1 {
			first = first[:idx] + " …"
		}
		fmt.Fprintf(&sb, "[%d] level=%-4s stream=%-6s — %s\n", i, e.Level, e.Stream, first)
	}
	return sb.String()
}
