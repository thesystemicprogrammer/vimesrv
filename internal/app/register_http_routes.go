package app

import (
	"github.com/thesystemicprogrammer/vimesrv/internal/adapters/http"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/web"
)

func registerHTTPHandlers(useCases *UseCases, adapters *Adapters, httpServer *server.HTTPServer, cfg *config.Config) {
	router := httpServer.Router()

	// Register PWA (Angular frontend)
	if err := httpServer.RegisterPWA(web.PWAFiles); err != nil {
		logger.Error().Err(err).Msg("failed to register PWA")
	}

	// === WebSocket Route (public, but with token auth in handler) ===

	// Register WebSocket handler if hub is enabled
	if adapters.WebSocketHub != nil {
		wsHandler := http.NewWebSocketHandler(adapters.WebSocketHub, &cfg.Auth)
		wsHandler.RegisterRoutes(router)
		logger.Info().Msg("WebSocket route registered")
	}

	// === Public Routes (no auth required) ===

	// Auth handler - login is public
	authHandler := http.NewAuthHandler(&cfg.Auth, adapters.UserRepository, useCases.ChangePasswordUseCase)
	authHandler.RegisterRoutes(router)

	// === Protected API Routes ===

	// Create API group with auth middleware
	apiGroup := router.Group("/api/v1")
	apiGroup.Use(server.AuthMiddleware(&cfg.Auth))
	{
		// Protected auth routes (me, stream-token, change-password)
		authHandler.RegisterProtectedRoutes(apiGroup)

		// User management routes (admin only)
		userHandler := http.NewUserHandler(
			useCases.CreateUserUseCase,
			useCases.ListUsersUseCase,
			useCases.GetUserUseCase,
			useCases.UpdateUserUseCase,
			useCases.DeleteUserUseCase,
			useCases.ResetPasswordUseCase,
		)
		userHandler.RegisterProtectedRoutes(apiGroup)

		// Scan Library API
		scanLibraryHttpHandler := http.NewScanLibraryHTTPHandler(useCases.EnqueueJobUseCase)
		scanLibraryHttpHandler.RegisterRoutes(apiGroup)

		// Media API
		mediaHandler := http.NewMediaHandler(useCases.ListMediaUseCase, useCases.GetMediaUseCase, useCases.DeleteMediaUseCase, cfg)
		mediaHandler.RegisterRoutes(apiGroup)
		mediaHandler.RegisterAdminRoutes(apiGroup)

		// Metadata Enrichment API (if TMDB is enabled)
		metadataHandler := http.NewMetadataHandler(
			useCases.GetCandidatesUseCase,
			useCases.LinkMetadataUseCase,
			useCases.SearchMetadataUseCase,
			useCases.LinkFromSearchUseCase,
			useCases.SkipEnrichmentUseCase,
			useCases.ResetEnrichmentUseCase,
			useCases.EnqueueJobUseCase,
			adapters.JobRepository,
		)
		metadataHandler.RegisterRoutes(apiGroup)

		// Library Browsing API
		libraryHandler := http.NewLibraryHandler(
			useCases.ListMoviesUseCase,
			useCases.GetMovieUseCase,
			useCases.GetMovieCreditsUseCase,
			useCases.ListSeriesUseCase,
			useCases.GetSeriesUseCase,
			useCases.GetSeriesCreditsUseCase,
			useCases.ListRecentUseCase,
			useCases.ListUnmatchedUseCase,
			useCases.ListGenresUseCase,
			useCases.SearchLibraryUseCase,
			useCases.DeleteSeasonUseCase,
			useCases.DeleteSeriesUseCase,
		)
		libraryHandler.RegisterRoutes(apiGroup)
		libraryHandler.RegisterAdminRoutes(apiGroup)

		// Job API (admin/manager only)
		jobHandler := http.NewJobHandler(useCases.ListJobsUseCase, adapters.ProgressCache)
		jobHandler.RegisterRoutes(apiGroup)

		// Watch Progress API
		watchProgressHandler := http.NewWatchProgressHandler(
			useCases.RecordWatchProgressUseCase,
			useCases.GetWatchProgressUseCase,
			useCases.GetContinueWatchingUseCase,
			useCases.RemoveFromContinueWatchingUseCase,
		)
		watchProgressHandler.RegisterRoutes(apiGroup)

		// Favorite API
		favoriteHandler := http.NewFavoriteHandler(
			useCases.ToggleFavoriteUseCase,
			useCases.GetUserFavoritesUseCase,
		)
		favoriteHandler.RegisterRoutes(apiGroup)

		// Recommendation API
		recommendationHandler := http.NewRecommendationHandler(
			useCases.BuildRecommendationModelUseCase,
			useCases.GetPersonalizedRecommendationsUseCase,
			adapters.RecommendationRepository,
		)
		recommendationHandler.RegisterRoutes(apiGroup)
		recommendationHandler.RegisterAdminRoutes(apiGroup)

		// Transcode Admin API (admin/manager only)
		transcodeAdminHandler := http.NewTranscodeAdminHandler(
			useCases.GetTranscodingDetailsUseCase,
			useCases.AddTranscodingUseCase,
			useCases.RecreateTranscodingUseCase,
			useCases.DeleteTranscodingUseCase,
			useCases.SearchMediaForTranscodingsUseCase,
			useCases.GetQualityProfilesUseCase,
		)
		transcodeAdminHandler.RegisterRoutes(apiGroup)
	}

	// === Protected Streaming Routes ===

	// DASH Streaming with stream auth middleware
	dashHandler := http.NewDASHHandler(useCases.GetMediaUseCase, cfg)

	// Direct Play handler for serving original files
	directHandler := http.NewDirectHandler(useCases.GetMediaUseCase, cfg)

	// Remux handler for on-the-fly MKV/AVI/WebM to fragmented MP4
	remuxHandler := http.NewRemuxHandler(useCases.GetMediaUseCase, cfg)

	// Probe handler for bandwidth measurement
	probeHandler := http.NewProbeHandler()

	// Stream routes require stream token auth
	streamGroup := router.Group("/stream")
	streamGroup.Use(server.StreamAuthMiddleware(&cfg.Auth))
	{
		// Manifest endpoint
		streamGroup.GET("/dash/:id/manifest.mpd", dashHandler.ServeManifest)

		// Content endpoint (video/audio segments, subtitles)
		streamGroup.Match([]string{"GET", "HEAD"}, "/dash/content/:id/*path", dashHandler.ServeContent)

		// Direct play endpoint (serves original MP4/MOV files with Range support)
		streamGroup.Match([]string{"GET", "HEAD"}, "/direct/:id", directHandler.Serve)

		// Remux endpoint (on-the-fly MKV/AVI/WebM to fragmented MP4)
		streamGroup.GET("/remux/:id", remuxHandler.Serve)

		// Bandwidth probe endpoint
		streamGroup.GET("/probe", probeHandler.Serve)
	}

	// === Worker API Routes (separate auth via worker token) ===

	// Register worker handler if worker mode is enabled
	// Note: Worker routes use their own auth middleware (not user auth)
	if cfg.Worker.Enabled && useCases.RegisterWorkerUseCase != nil {
		workerHandler := http.NewWorkerHandler(
			&cfg.Worker,
			useCases.RegisterWorkerUseCase,
			useCases.HeartbeatUseCase,
			useCases.ClaimJobForWorkerUseCase,
			useCases.CompleteWorkerJobUseCase,
			useCases.FailWorkerJobUseCase,
			useCases.ReportProgressUseCase,
		)
		// Worker routes are under /api/v1 but have their own auth middleware
		workerAPIGroup := router.Group("/api/v1")
		workerHandler.RegisterRoutes(workerAPIGroup)
		logger.Info().Msg("Worker API routes registered")
	}

	logger.Info().
		Bool("auth_enabled", cfg.Auth.Enabled).
		Msg("HTTP routes registered")
}
