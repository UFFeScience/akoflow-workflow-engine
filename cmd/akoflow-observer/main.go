package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/UFFeScience/akoflow/internal/provider/observation"
)

const observationPrefix = "AKOFLOW_ARTIFACT_MANIFEST="

func main() {
	if len(os.Args) < 2 {
		fatal("expected run")
	}
	switch os.Args[1] {
	case "run":
		run()
	default:
		fatal("unknown command %q", os.Args[1])
	}
}

func run() {
	flags := flag.NewFlagSet("run", flag.ExitOnError)
	config := observation.Config{}
	flags.StringVar(&config.RunID, "run-id", "", "execution run identifier")
	flags.StringVar(&config.ActivityID, "activity-id", "", "activity identifier")
	flags.StringVar(&config.Runtime, "runtime", "", "runtime identifier")
	flags.StringVar(&config.Root, "root", ".", "directory to observe")
	flags.StringVar(&config.ManifestPath, "manifest", "", "manifest destination")
	flags.IntVar(&config.Attempt, "attempt", 1, "activity attempt")
	_ = flags.Parse(os.Args[2:])
	command := flags.Args()
	if len(command) > 0 && command[0] == "--" {
		command = command[1:]
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	manifest, err := observation.Run(ctx, config, command)
	payload, marshalErr := json.Marshal(manifest)
	if marshalErr == nil {
		// Start on a fresh line even when the activity did not terminate its last
		// stdout record. The adapter can then recover the lifecycle envelope.
		fmt.Println("\n" + observationPrefix + base64.StdEncoding.EncodeToString(payload))
	}
	if marshalErr != nil && err == nil {
		err = marshalErr
	}
	if err != nil {
		if manifest.ExitCode > 0 {
			os.Exit(manifest.ExitCode)
		}
		fatal("observe activity: %v", err)
	}
}

func fatal(format string, values ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", values...)
	os.Exit(1)
}
