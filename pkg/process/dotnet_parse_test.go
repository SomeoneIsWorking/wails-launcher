package process

// White-box tests for log parsing, URL extraction, and multi-line buffering.
// These exercise unexported methods directly so they live in package process
// (not package process_test).

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

// newTestService returns a DotnetService with buffered channels, ready for
// channel-based assertions. No real process is involved.
func newTestService() *DotnetService {
	return &DotnetService{
		logChan:    make(chan LogEntry, 100),
		urlChan:    make(chan string, 10),
		statusChan: make(chan ServiceStatus, 10),
	}
}

// drainLogs collects all currently-buffered log entries.
func drainLogs(ds *DotnetService) []LogEntry {
	var entries []LogEntry
	for {
		select {
		case e := <-ds.logChan:
			entries = append(entries, e)
		default:
			return entries
		}
	}
}

// drainURLs collects all currently-buffered URLs.
func drainURLs(ds *DotnetService) []string {
	var urls []string
	for {
		select {
		case u := <-ds.urlChan:
			urls = append(urls, u)
		default:
			return urls
		}
	}
}

// ── parseLog ─────────────────────────────────────────────────────────────────

func TestParseLog_Levels(t *testing.T) {
	ds := newTestService()

	cases := []struct {
		line   string
		stream string
		want   LogLevel
	}{
		// Explicit ASP.NET Core log prefixes
		{"info: Microsoft.Hosting.Lifetime[0]", "stdout", Inf},
		{"Info: something", "stdout", Inf},
		{"INFO: CAPS", "stdout", Inf},
		{"warn: slow query detected", "stdout", Warn},
		{"Warn: disk space low", "stdout", Warn},
		{"error: unhandled exception", "stdout", Err},
		{"Error: bad config", "stdout", Err},
		{"debug: entering handler", "stdout", Dbg},
		{"Debug: cache miss", "stdout", Dbg},
		{"trace: sql executed", "stdout", Dbg},
		{"Trace: verbose output", "stdout", Dbg},
		{"critical: out of memory", "stdout", Err},
		{"Critical: unrecoverable", "stdout", Err},
		{"fail: request pipeline failed", "stdout", Err},
		{"Fail: background job crashed", "stdout", Err},
		// Serilog/structured-log ERR markers
		{"[12:00:00 ERR] Database connection lost", "stdout", Err},
		{"2024-01-01 ERR something went wrong", "stdout", Err},
		// Compiler errors and warnings
		{"/app/Foo.cs(10,5): error CS0103: name not found", "stdout", Err},
		{"/app/Foo.cs(10,5): warning CS8600: null conversion", "stdout", Warn},
		// NuGet-style warnings
		{"/repo/Project.csproj : warning NU1701: package targeting", "stdout", Warn},
		// Stderr stream → always Err regardless of content
		{"Application started.", "stderr", Err},
		{"some plain text", "stderr", Err},
		// Unrecognised plain stdout → Inf
		{"Build succeeded.", "stdout", Inf},
		{"   indented continuation", "stdout", Inf},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q/%s", tc.line[:min(len(tc.line), 40)], tc.stream), func(t *testing.T) {
			entry := ds.parseLog(tc.line, tc.stream)
			if entry == nil {
				t.Fatal("parseLog returned nil, want an entry")
			}
			if entry.Level != tc.want {
				t.Errorf("level = %q, want %q", entry.Level, tc.want)
			}
			if entry.Message != tc.line {
				t.Errorf("message = %q, want %q", entry.Message, tc.line)
			}
			if entry.Stream != tc.stream {
				t.Errorf("stream = %q, want %q", entry.Stream, tc.stream)
			}
		})
	}
}

func TestParseLog_NETSDK1138Filtered(t *testing.T) {
	ds := newTestService()
	line := "warn : The 'TargetFramework' value 'net8.0' is not supported. NETSDK1138"
	if entry := ds.parseLog(line, "stdout"); entry != nil {
		t.Errorf("expected nil for NETSDK1138 line, got %+v", entry)
	}
}

// ── processLine / URL extraction ─────────────────────────────────────────────

func TestProcessLine_URLExtraction(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		wantURL string
	}{
		{
			name:    "http URL",
			line:    "info: Microsoft.Hosting.Lifetime[14] Now listening on: http://localhost:5000",
			wantURL: "http://localhost:5000",
		},
		{
			name:    "https URL",
			line:    "Now listening on: https://localhost:5001",
			wantURL: "https://localhost:5001",
		},
		{
			name:    "URL with path",
			line:    "Now listening on: http://[::]:8080",
			wantURL: "http://[::]:8080",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := newTestService()
			ds.processLine(tc.line, "stdout")

			urls := drainURLs(ds)
			if len(urls) != 1 {
				t.Fatalf("got %d URL(s), want 1: %v", len(urls), urls)
			}
			if urls[0] != tc.wantURL {
				t.Errorf("url = %q, want %q", urls[0], tc.wantURL)
			}
		})
	}
}

func TestProcessLine_NoURL(t *testing.T) {
	ds := newTestService()
	ds.processLine("info: Application started. Press Ctrl+C to shut down.", "stdout")

	if urls := drainURLs(ds); len(urls) != 0 {
		t.Errorf("expected no URLs, got %v", urls)
	}
	if logs := drainLogs(ds); len(logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(logs))
	}
}

func TestProcessLine_URLAlsoEmitsLog(t *testing.T) {
	ds := newTestService()
	ds.processLine("Now listening on: http://localhost:5000", "stdout")

	if urls := drainURLs(ds); len(urls) != 1 {
		t.Errorf("expected 1 URL, got %d", len(urls))
	}
	// The line itself should also produce a log entry
	if logs := drainLogs(ds); len(logs) != 1 {
		t.Errorf("expected 1 log entry alongside URL, got %d", len(logs))
	}
}

// ── readOutput (multi-line buffering) ────────────────────────────────────────

// feedLines writes lines to a pipe, closes the writer, then calls readOutput
// synchronously via a goroutine and waits for it to finish.
func feedLines(ds *DotnetService, stream string, lines []string) {
	pr, pw := io.Pipe()
	done := make(chan struct{})
	go func() {
		ds.readOutput(pr, stream)
		close(done)
	}()
	for _, l := range lines {
		fmt.Fprintln(pw, l)
	}
	pw.Close()
	<-done
}

func TestReadOutput_SingleLine(t *testing.T) {
	ds := newTestService()
	feedLines(ds, "stdout", []string{"info: server started"})

	logs := drainLogs(ds)
	if len(logs) != 1 {
		t.Fatalf("got %d entries, want 1", len(logs))
	}
	if logs[0].Message != "info: server started" {
		t.Errorf("message = %q", logs[0].Message)
	}
}

func TestReadOutput_MultiLineContinuation(t *testing.T) {
	// A log-start line followed by indented continuation lines should be
	// buffered into a single log entry.
	ds := newTestService()
	feedLines(ds, "stdout", []string{
		"info: Request failed",
		"      System.InvalidOperationException: bad state",
		"         at MyApp.Handler.Invoke()",
	})

	logs := drainLogs(ds)
	if len(logs) != 1 {
		t.Fatalf("got %d entries, want 1 (multi-line should be one entry)", len(logs))
	}
	want := "info: Request failed\n      System.InvalidOperationException: bad state\n         at MyApp.Handler.Invoke()"
	if logs[0].Message != want {
		t.Errorf("message =\n%q\nwant\n%q", logs[0].Message, want)
	}
}

func TestReadOutput_TwoConsecutiveLogLines(t *testing.T) {
	// Two lines that each start a new log entry must produce two separate entries.
	ds := newTestService()
	feedLines(ds, "stdout", []string{
		"info: first entry",
		"warn: second entry",
	})

	logs := drainLogs(ds)
	if len(logs) != 2 {
		t.Fatalf("got %d entries, want 2", len(logs))
	}
	if logs[0].Level != Inf {
		t.Errorf("entry 0 level = %q, want INF", logs[0].Level)
	}
	if logs[1].Level != Warn {
		t.Errorf("entry 1 level = %q, want WARN", logs[1].Level)
	}
}

func TestReadOutput_EmptyLineFlushesBuffer(t *testing.T) {
	// An empty line between two plain (non-log-start) lines forces each to be
	// flushed as its own entry.
	ds := newTestService()
	feedLines(ds, "stdout", []string{
		"Build started.",
		"",
		"Build succeeded.",
	})

	logs := drainLogs(ds)
	if len(logs) != 2 {
		t.Fatalf("got %d entries, want 2", len(logs))
	}
}

func TestReadOutput_StreamMarkerLinesNotLogStart(t *testing.T) {
	// "STDERR" and "STDOUT" marker lines are not treated as log-start lines,
	// so they don't flush the previous buffer — they get merged into it.
	// A subsequent real log-start line flushes everything accumulated so far,
	// producing one combined entry for the markers and a separate entry for
	// the real log line.
	ds := newTestService()
	feedLines(ds, "stdout", []string{
		"STDERR",
		"STDOUT",
		"info: real log line",
	})

	logs := drainLogs(ds)
	// Entry 0: the accumulated "STDERR\nSTDOUT" block (flushed when the info:
	//           line is recognised as a new log-start).
	// Entry 1: "info: real log line"
	if len(logs) != 2 {
		t.Fatalf("got %d entries, want 2", len(logs))
	}
	if !strings.Contains(logs[0].Message, "STDERR") {
		t.Errorf("entry 0 = %q, expected it to contain 'STDERR'", logs[0].Message)
	}
	if !strings.Contains(logs[1].Message, "real log line") {
		t.Errorf("entry 1 = %q, expected it to contain 'real log line'", logs[1].Message)
	}
}

func TestReadOutput_URLExtractedDuringStream(t *testing.T) {
	ds := newTestService()
	feedLines(ds, "stdout", []string{
		"info: Hosting environment: Development",
		"info: Now listening on: http://localhost:5000",
		"info: Application started.",
	})

	urls := drainURLs(ds)
	if len(urls) != 1 {
		t.Fatalf("got %d URL(s), want 1: %v", len(urls), urls)
	}
	if urls[0] != "http://localhost:5000" {
		t.Errorf("url = %q", urls[0])
	}

	logs := drainLogs(ds)
	if len(logs) != 3 {
		t.Errorf("got %d log entries, want 3", len(logs))
	}
}

func TestReadOutput_BuildOutputLines(t *testing.T) {
	ds := newTestService()
	feedLines(ds, "stdout", []string{
		"Build succeeded.",
		"  0 Warning(s)",
		"  0 Error(s)",
		"",
		"Restore complete (1.2s)",
	})

	logs := drainLogs(ds)
	// "Build succeeded." starts a buffer; its continuation lines ("  0 Warning(s)"
	// etc.) are appended until the empty line flushes. "Restore complete" is
	// a separate entry.
	if len(logs) < 1 {
		t.Fatalf("got 0 log entries, want at least 1")
	}
	first := logs[0].Message
	if !strings.HasPrefix(first, "Build succeeded.") {
		t.Errorf("first entry = %q, want it to start with 'Build succeeded.'", first)
	}
}

// min is available as a builtin in Go 1.21+; shadowed here for clarity.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
