package rebuild

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// PrepareUseCase exports users and metadata links to a JSON file
type PrepareUseCase struct {
	config            *config.Config
	userRepository    ports.UserRepository
	rebuildRepository ports.RebuildRepository
	filesystem        ports.FileSystemService
}

// NewPrepareUseCase creates a new prepare use case
func NewPrepareUseCase(
	cfg *config.Config,
	userRepository ports.UserRepository,
	rebuildRepository ports.RebuildRepository,
	filesystem ports.FileSystemService,
) *PrepareUseCase {
	return &PrepareUseCase{
		config:            cfg,
		userRepository:    userRepository,
		rebuildRepository: rebuildRepository,
		filesystem:        filesystem,
	}
}

// Execute exports all users and metadata links to rebuild.json in the library root
func (uc *PrepareUseCase) Execute(ctx context.Context) error {
	logger.Info().Msg("[rebuild] Starting prepare-rebuild export")

	// Build the rebuild data structure
	data := RebuildData{
		Version:   RebuildDataVersion,
		CreatedAt: time.Now().UTC(),
	}

	// Export users
	logger.Info().Msg("[rebuild] Exporting users...")
	users, err := uc.userRepository.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	for _, u := range users {
		userData := UserData{
			ID:                 u.ID,
			Username:           u.Username,
			PasswordHash:       u.PasswordHash,
			Role:               string(u.Role),
			MustChangePassword: u.MustChangePassword,
			CreatedAt:          u.CreatedAt,
			UpdatedAt:          u.UpdatedAt,
		}
		if u.CreatedBy.Valid {
			userData.CreatedBy = &u.CreatedBy.String
		}
		data.Users = append(data.Users, userData)
	}
	logger.Info().Int("count", len(data.Users)).Msg("[rebuild] Exported users")

	// Export media links
	logger.Info().Msg("[rebuild] Exporting media links...")
	exportedLinks, err := uc.rebuildRepository.ExportMediaLinks(ctx)
	if err != nil {
		return fmt.Errorf("failed to export media links: %w", err)
	}

	// Convert from ports.ExportedMediaLink to rebuild.MediaLink
	for _, link := range exportedLinks {
		data.MediaLinks = append(data.MediaLinks, MediaLink{
			Fingerprint:   link.Fingerprint,
			MetadataType:  link.MetadataType,
			TMDBID:        link.TMDBID,
			SeriesTMDBID:  link.SeriesTMDBID,
			SeasonNumber:  link.SeasonNumber,
			EpisodeNumber: link.EpisodeNumber,
			Edition:       link.Edition,
		})
	}
	logger.Info().Int("count", len(data.MediaLinks)).Msg("[rebuild] Exported media links")

	// Write to JSON file
	outputPath := filepath.Join(uc.config.Media.LibraryPath, RebuildFileName)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal rebuild data: %w", err)
	}

	if err := uc.filesystem.WriteFile(outputPath, jsonData); err != nil {
		return fmt.Errorf("failed to write rebuild.json: %w", err)
	}

	logger.Info().
		Str("path", outputPath).
		Int("users", len(data.Users)).
		Int("media_links", len(data.MediaLinks)).
		Msg("[rebuild] Prepare complete. Export saved to rebuild.json")

	return nil
}
