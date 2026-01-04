package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/thesystemicprogrammer/vimesrv/internal/worker"
	"github.com/thesystemicprogrammer/vimesrv/internal/worker/client"
	"github.com/thesystemicprogrammer/vimesrv/internal/worker/config"
	"github.com/thesystemicprogrammer/vimesrv/pkg/transcoding"
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

vimesrv-worker - Distributed Transcoding Worker


`
)

var version, commit, date = "dev", "dev", "n.a."

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	versionFlag := flag.Bool("version", false, "Show version information")
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
	logger := initLogger(cfg.Logging)

	// Generate worker ID if not set
	workerID := cfg.Worker.ID
	if workerID == "" {
		workerID = uuid.New().String()
		logger.Info().Str("worker_id", workerID).Msg("Generated worker ID")
	}

	// Validate FFmpeg is available
	transcoder := transcoding.NewFFmpegTranscoder(transcoding.FFmpegConfig{
		FFmpegPath:     cfg.Transcoding.FFmpegPath,
		FFprobePath:    cfg.Transcoding.FFprobePath,
		TimeoutSeconds: cfg.Transcoding.TimeoutSeconds,
	})

	if err := transcoder.IsAvailable(); err != nil {
		logger.Fatal().Err(err).Msg("FFmpeg not available")
	}
	logger.Info().Str("ffmpeg", cfg.Transcoding.FFmpegPath).Msg("FFmpeg available")

	// Create server client
	serverClient := client.NewServerClient(cfg.Server.URL, cfg.Server.AuthToken)

	// Create worker
	w := worker.New(workerID, cfg, serverClient, transcoder, logger)

	// Setup signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger.Info().
		Str("server_url", cfg.Server.URL).
		Str("worker_name", cfg.Worker.Name).
		Int("concurrency", cfg.Worker.Concurrency).
		Msg("Starting worker")

	// Start worker (blocks until shutdown)
	if err := w.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Worker failed")
	}

	logger.Info().Msg("Worker shutdown complete")
}

// initLogger initializes the zerolog logger based on configuration
func initLogger(cfg config.LoggingConfig) zerolog.Logger {
	// Set log level
	level := zerolog.InfoLevel
	switch cfg.Level {
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	}

	// Set output format
	var logger zerolog.Logger
	if cfg.Format == "json" {
		logger = zerolog.New(os.Stdout).
			Level(level).
			With().
			Timestamp().
			Logger()
	} else {
		// Console format
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		logger = zerolog.New(output).
			Level(level).
			With().
			Timestamp().
			Logger()
	}

	return logger
}
