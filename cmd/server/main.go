package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/media"
	metadataAdapter "github.com/thesystemicprogrammer/vimesrv/internal/adapters/metadata"
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/repository"
	"github.com/thesystemicprogrammer/vimesrv/internal/app"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/rebuild"
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

vimesrv - Video Media Server with DASH Transcoding


`
)

var version, commit, date = "dev", "dev", "n.a."

func main() {
	configPath := flag.String("config", "configs/default.yaml", "Path to configuration file")
	versionFlag := flag.Bool("version", false, "Flag to show the version information")
	prepareRebuild := flag.Bool("prepare-rebuild", false, "Export users and metadata links to rebuild.json for database rebuild")
	rebuildFromDump := flag.Bool("rebuild-from-dump", false, "Rebuild database from rebuild.json (requires rebuild.allow_rebuild: true in config)")
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

	// Handle prepare-rebuild command
	if *prepareRebuild {
		if err := runPrepareRebuild(cfg); err != nil {
			logger.Fatal().Err(err).Msg("prepare-rebuild failed")
		}
		os.Exit(0)
	}

	// Handle rebuild-from-dump command
	if *rebuildFromDump {
		if err := runRebuildFromDump(cfg); err != nil {
			logger.Fatal().Err(err).Msg("rebuild-from-dump failed")
		}
		fmt.Println("\nRebuild complete. You can now start the server normally.")
		os.Exit(0)
	}

	logger.Info().
		Str("config", *configPath).
		Str("version", version).
		Msg("starting vimesrv")

	// Create and run application
	application, err := app.NewApplication(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create application")
	}

	// Start application (blocks until shutdown signal)
	if err := application.Start(); err != nil {
		logger.Fatal().Err(err).Msg("failed to start application")
	}
}

// runPrepareRebuild exports users and metadata links to rebuild.json
func runPrepareRebuild(cfg *config.Config) error {
	logger.Info().Msg("Running prepare-rebuild: exporting data for database rebuild")

	// Initialize minimal infrastructure needed for export
	db, err := database.New(database.Config{
		Path:            cfg.Database.Path,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: 0,
		ConnMaxIdleTime: 0,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Create repositories
	userRepository := repository.NewSQLiteUserRepository(db)
	rebuildRepository := repository.NewSQLiteRebuildRepository(db)
	filesystem := media.NewOSFileSystem()

	// Create and run prepare use case
	prepareUseCase := rebuild.NewPrepareUseCase(cfg, userRepository, rebuildRepository, filesystem)

	ctx := context.Background()
	if err := prepareUseCase.Execute(ctx); err != nil {
		return fmt.Errorf("prepare-rebuild failed: %w", err)
	}

	logger.Info().Msg("Prepare-rebuild complete. You can now delete the database and run --rebuild-from-dump")
	return nil
}

// runRebuildFromDump clears database and imports data from rebuild.json
func runRebuildFromDump(cfg *config.Config) error {
	logger.Info().Msg("Running rebuild-from-dump: rebuilding database from export")

	// Verify rebuild is allowed in config
	if !cfg.Rebuild.AllowRebuild {
		return fmt.Errorf("rebuild is disabled in configuration; set rebuild.allow_rebuild: true to enable")
	}

	// Initialize minimal infrastructure needed for rebuild
	db, err := database.New(database.Config{
		Path:            cfg.Database.Path,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: 0,
		ConnMaxIdleTime: 0,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Run migrations first (in case database was deleted)
	databaseMigration := database.NewDatabaseMigration(db.DB)
	if err := databaseMigration.Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create core adapters
	filesystem := media.NewOSFileSystem()
	fileHasher := media.NewBlake2bHasher()
	ffprobeService := media.NewFFProbeAdapter(cfg.Media.FFProbeTimeoutSeconds)

	// Create repositories
	rebuildRepository := repository.NewSQLiteRebuildRepository(db)
	transcodeRepository := repository.NewTranscodeRepository(db)
	mediaRepository := repository.NewMediaRepository(db)
	jobRepository := repository.NewJobRepository(db)
	movieMetadataRepository := repository.NewSQLiteMovieMetadataRepository(db.DB)
	seriesMetadataRepository := repository.NewSQLiteSeriesMetadataRepository(db.DB)
	seasonMetadataRepository := repository.NewSQLiteSeasonMetadataRepository(db.DB)
	episodeMetadataRepository := repository.NewSQLiteEpisodeMetadataRepository(db.DB)
	movieCreditRepository := repository.NewSQLiteMovieCreditRepository(db.DB)
	movieCertificationRepository := repository.NewSQLiteMovieCertificationRepository(db.DB)
	seriesCreditRepository := repository.NewSQLiteSeriesCreditRepository(db.DB)
	searchRepository := repository.NewSearchRepository(db)

	// Create rebuild use case
	rebuildUseCase := rebuild.NewRebuildUseCase(
		cfg,
		rebuildRepository,
		transcodeRepository,
		mediaRepository,
		jobRepository,
		filesystem,
	)

	ctx := context.Background()

	// Check if TMDB is enabled for auto-linking
	var linker *rebuild.Linker
	if cfg.TMDB.Enabled {
		tmdbClient := metadataAdapter.NewTMDBHTTPClient(cfg.TMDB)
		imageDownloader := metadataAdapter.NewHTTPImageDownloader(cfg.TMDB, tmdbClient)

		linker = rebuild.NewLinker(
			cfg.TMDB,
			tmdbClient,
			imageDownloader,
			movieMetadataRepository,
			seriesMetadataRepository,
			seasonMetadataRepository,
			episodeMetadataRepository,
			movieCreditRepository,
			movieCertificationRepository,
			seriesCreditRepository,
			searchRepository,
			mediaRepository,
		)
		logger.Info().Msg("TMDB enabled - auto-linking will be performed")
	} else {
		logger.Warn().Msg("TMDB disabled - files will be imported but not auto-linked")
	}

	// We need to run Execute first to get the auto-link map, then create the scanner
	result, err := rebuildUseCase.Execute(ctx)
	if err != nil {
		return fmt.Errorf("rebuild failed: %w", err)
	}

	// Build auto-link map from the loaded media links
	autoLinkMap := make(map[string]rebuild.AutoLinkData)
	for fingerprint, link := range rebuildUseCase.GetAutoLinkMap() {
		autoLinkMap[fingerprint] = link.ToAutoLinkData()
	}

	// Create scanner with linker and auto-link map
	scanner := rebuild.NewScanner(
		cfg.Media,
		fileHasher,
		ffprobeService,
		filesystem,
		mediaRepository,
		linker,
		autoLinkMap,
	)

	// Run the full scan
	logger.Info().Msg("[rebuild] Starting media library scan...")
	scanResult, err := scanner.Scan(ctx)
	if err != nil {
		return fmt.Errorf("library scan failed: %w", err)
	}

	result.FilesScanned = scanResult.FilesScanned
	result.FilesProcessed = scanResult.FilesProcessed
	result.FilesLinked = scanResult.FilesLinked
	result.Errors = append(result.Errors, scanResult.Errors...)

	// Recover transcodes
	transcodesRecovered, err := rebuildUseCase.RecoverTranscodes(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("[rebuild] Transcode recovery failed")
	} else {
		result.TranscodesRecovered = transcodesRecovered
	}

	logger.Info().
		Int("users_imported", result.UsersImported).
		Int("media_links_loaded", result.MediaLinksLoaded).
		Int("files_scanned", result.FilesScanned).
		Int("files_processed", result.FilesProcessed).
		Int("files_linked", result.FilesLinked).
		Int("transcodes_recovered", result.TranscodesRecovered).
		Int("errors", len(result.Errors)).
		Msg("Rebuild complete")

	return nil
}
