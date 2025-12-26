package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/thesystemicprogrammer/vimesrv/internal/utils/config"
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

func main() {
	configPath := flag.String("config", "configs/default.yaml", "Path to configuration file")
	flag.Parse()

	fmt.Print(banner)

	// Load configuration
	_, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}
}
