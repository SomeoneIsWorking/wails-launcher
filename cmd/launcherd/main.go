// launcherd runs the launcher with no window: the same services, the same control API
// on 127.0.0.1:9901, nothing on the screen and nothing in the Dock. It is what the
// `oasis service` commands drive, so bringing the dev stack up never opens an app.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"wails-launcher/pkg/launcher"
)

func main() {
	startAll := flag.Bool("start-all", false, "start every configured service once the API is up")
	flag.Parse()

	app := launcher.NewApp()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app.Startup(ctx)
	fmt.Println("launcherd listening on", launcher.HTTPListenAddr)

	if *startAll {
		for id := range app.GetServices() {
			if err := app.StartServiceWithoutBuild(id); err != nil {
				fmt.Fprintf(os.Stderr, "start %s: %v\n", id, err)
			}
		}
	}

	<-ctx.Done()
	fmt.Println("launcherd stopping")

	for id, info := range app.GetServices() {
		if info.Status == "running" || info.Status == "starting" {
			app.StopService(id) //nolint:errcheck
		}
	}
	app.Shutdown(context.Background())
}
