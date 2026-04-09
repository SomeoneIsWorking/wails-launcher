package process_test

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"wails-launcher/pkg/process"
)

const (
	testPort        = "5199"
	startTimeout    = 120 * time.Second // first run needs to build
	restartTimeout  = 60 * time.Second  // subsequent runs skip restore
	stopTimeout     = 15 * time.Second
)

// waitForState drains all three channels until the condition is met or timeout.
func waitForState(
	logChan <-chan process.LogEntry,
	urlChan <-chan string,
	statusChan <-chan process.ServiceStatus,
	timeout time.Duration,
	cond func(status process.ServiceStatus, url string) bool,
) (process.ServiceStatus, string, error) {
	deadline := time.After(timeout)
	var lastStatus process.ServiceStatus
	var lastURL string

	for {
		select {
		case s := <-statusChan:
			lastStatus = s
		case u := <-urlChan:
			lastURL = u
		case <-logChan:
			// drain logs so the buffer never blocks
		case <-deadline:
			return lastStatus, lastURL, fmt.Errorf("timeout after %s (last status=%q, url=%q)", timeout, lastStatus, lastURL)
		}
		if cond(lastStatus, lastURL) {
			return lastStatus, lastURL, nil
		}
	}
}

func isRunningWithURL(status process.ServiceStatus, url string) bool {
	return url != "" && (status == process.Initializing || status == process.Running)
}

func isStopped(status process.ServiceStatus, _ string) bool {
	return status == process.Stopped
}

func testdataPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "testdata", "TestWebApi")
}

func checkHTTP(t *testing.T, baseURL string) {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health returned %d, want 200", resp.StatusCode)
	}
}

func TestDotnetServiceStartStopRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping dotnet integration test in short mode")
	}

	svcPath := testdataPath(t)
	env := process.ServiceEnv{
		"ASPNETCORE_URLS":        "http://localhost:" + testPort,
		"ASPNETCORE_ENVIRONMENT": "Development",
	}

	svc := process.NewDotnetService(svcPath, env, "")
	logChan, urlChan, statusChan := svc.GetChannels()

	// ── Start ─────────────────────────────────────────────────────────────────
	t.Log("Starting dotnet service (includes build)…")
	if err := svc.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	_, url, err := waitForState(logChan, urlChan, statusChan, startTimeout, isRunningWithURL)
	if err != nil {
		t.Fatalf("waiting for start: %v", err)
	}
	t.Logf("Service is up at %s", url)

	checkHTTP(t, url)
	t.Log("HTTP /health check passed after start")

	// ── Stop ──────────────────────────────────────────────────────────────────
	t.Log("Stopping service…")
	if err := svc.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}

	_, _, err = waitForState(logChan, urlChan, statusChan, stopTimeout, isStopped)
	if err != nil {
		t.Fatalf("waiting for stop: %v", err)
	}
	t.Log("Service stopped")

	// ── Restart ───────────────────────────────────────────────────────────────
	t.Log("Restarting service (no rebuild)…")
	if err := svc.StartWithoutBuild(); err != nil {
		t.Fatalf("StartWithoutBuild() error: %v", err)
	}

	_, url, err = waitForState(logChan, urlChan, statusChan, restartTimeout, isRunningWithURL)
	if err != nil {
		t.Fatalf("waiting for restart: %v", err)
	}
	t.Logf("Service restarted at %s", url)

	checkHTTP(t, url)
	t.Log("HTTP /health check passed after restart")

	// ── Final Stop ────────────────────────────────────────────────────────────
	t.Log("Final stop…")
	if err := svc.Stop(); err != nil {
		t.Fatalf("final Stop() error: %v", err)
	}

	_, _, err = waitForState(logChan, urlChan, statusChan, stopTimeout, isStopped)
	if err != nil {
		t.Fatalf("waiting for final stop: %v", err)
	}
	t.Log("Service stopped cleanly")
}
