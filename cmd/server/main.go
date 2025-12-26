package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/thesystemicprogrammer/vimesrv/internal/utils/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/utils/logger"
)

const (
	banner = `
              ███
             ░░░
 █████ █████ ████  █████████████    ██████   █████  ████████  █████ █████
░░███ ░░███ ░░███ ░░███░░███░░███  ███░░███ ███░░  ░░███░░███░░███ ░░███ 
 ░███  ░███  ░███  ░███ ░███ ░███ ░███████ ░░█████  ░███ ░░░  ░███  ░███ 
 ░░███ ███   ░███  ░███ ░███ ░███ ░███░░░   ░░░░███ ░███      ░░███ ███  
  ░░█████    █████ █████░███ █████░░██████  ██████  █████      ░░█████   
   ░░░░░    ░░░░░ ░░░░░ ░░░ ░░░░░  ░░░░░░  ░░░░░░  ░░░░░        ░░░░░    

vimesrv - Video Media Server with DASH and HLS Transcoding


`
)

var version, commit, date = "dev", "dev", "n.a."

func main() {
	configPath := flag.String("config", "configs/default.yaml", "Path to configuration file")
	versionFlag := flag.Bool("version", false, "Flag to show the version information")
	flag.Parse()

	fmt.Print(banner)

	if *versionFlag {
		fmt.Printf("Version:      %s\n", version)
		fmt.Printf("Commit Hash:  %s\n", commit)
		fmt.Printf("Commit Date:  %s\n", date)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.New(cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.SetGlobal(log)

	logger.Info().
		Str("config", *configPath).
		Str("version", "1").
		Msg("starting vimesrv")
}
