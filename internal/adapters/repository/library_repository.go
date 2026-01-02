package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/database"
	"github.com/thesystemicprogrammer/vimesrv/internal/usecase/ports"
)

// LibraryRepository implements library-focused queries
type LibraryRepository struct {
	db *sql.DB
}

// NewLibraryRepository creates a new LibraryRepository instance
func NewLibraryRepository(db *database.DB) ports.LibraryRepository {
	return &LibraryRepository{
		db: db.DB,
	}
}

// ListMovies returns movies with their metadata, sorted by most recently added
func (r *LibraryRepository) ListMovies(ctx context.Context, language string, limit, offset int) ([]ports.MovieSummary, int, error) {
	exactLang, baseLang := languageParams(language)

	// Count total movies
	countQuery := `
		SELECT COUNT(*) 
		FROM media_files mf
		WHERE mf.metadata_type = 'movie' AND mf.movie_metadata_id IS NOT NULL
	`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count movies: %w", err)
	}

	// Query movies with metadata
	// Translation join uses subquery with priority: exact lang > base lang > English > English variant
	query := `
		SELECT 
			mf.id,
			mf.duration,
			mf.resolution,
			mf.status,
			mf.enrichment_status,
			mf.created_at,
			mf.movie_metadata_id,
			COALESCE(t.transcode_status, 'none') as transcode_status,
			mm.original_title,
			COALESCE(mt.title, mm.original_title) as title,
			COALESCE(SUBSTR(mm.release_date, 1, 4), '') as year,
			COALESCE(mm.poster_path, '') as poster_path,
			COALESCE(mm.backdrop_path, '') as backdrop_path,
			mm.vote_average,
			COALESCE(mm.genres, '[]') as genres
		FROM media_files mf
		LEFT JOIN movie_metadata mm ON mf.movie_metadata_id = mm.id
		LEFT JOIN (
			SELECT movie_metadata_id, title, tagline, overview,
				ROW_NUMBER() OVER (PARTITION BY movie_metadata_id ORDER BY 
					CASE 
						WHEN language = ? THEN 0 
						WHEN language LIKE ? || '%' THEN 1 
						WHEN language = 'en' THEN 2
						WHEN language LIKE 'en%' THEN 3
						ELSE 4 
					END
				) as rn
			FROM movie_metadata_translations
			WHERE language = ? OR language LIKE ? || '%' OR language = 'en' OR language LIKE 'en%'
		) mt ON mm.id = mt.movie_metadata_id AND mt.rn = 1
		LEFT JOIN (
			SELECT media_id, 
				CASE WHEN SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) > 0 THEN 'completed'
					 WHEN SUM(CASE WHEN status IN ('pending', 'processing') THEN 1 ELSE 0 END) > 0 THEN 'pending'
					 ELSE 'none' END as transcode_status
			FROM transcodes
			GROUP BY media_id
		) t ON mf.id = t.media_id
		WHERE mf.metadata_type = 'movie' AND mf.movie_metadata_id IS NOT NULL
		ORDER BY mf.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, exactLang, baseLang, exactLang, baseLang, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query movies: %w", err)
	}
	defer rows.Close()

	var movies []ports.MovieSummary
	for rows.Next() {
		var m ports.MovieSummary
		var movieMetadataID sql.NullInt64
		var originalTitle, genresJSON string

		err := rows.Scan(
			&m.MediaID,
			&m.Duration,
			&m.Resolution,
			&m.Status,
			&m.EnrichmentStatus,
			&m.CreatedAt,
			&movieMetadataID,
			&m.TranscodeStatus,
			&originalTitle,
			&m.Title,
			&m.Year,
			&m.PosterPath,
			&m.BackdropPath,
			&m.VoteAverage,
			&genresJSON,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan movie: %w", err)
		}

		if movieMetadataID.Valid {
			m.MovieMetadataID = &movieMetadataID.Int64
		}
		m.Genres = parseGenresJSON(genresJSON)

		movies = append(movies, m)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating movies: %w", err)
	}

	return movies, total, nil
}

// GetMovie returns a single movie with full details
func (r *LibraryRepository) GetMovie(ctx context.Context, mediaID string, language string) (*ports.MovieSummary, error) {
	exactLang, baseLang := languageParams(language)

	query := `
		SELECT 
			mf.id,
			mf.duration,
			mf.resolution,
			mf.status,
			mf.enrichment_status,
			mf.created_at,
			mf.movie_metadata_id,
			COALESCE(t.transcode_status, 'none') as transcode_status,
			mm.original_title,
			COALESCE(mt.title, mm.original_title) as title,
			COALESCE(SUBSTR(mm.release_date, 1, 4), '') as year,
			COALESCE(mm.poster_path, '') as poster_path,
			COALESCE(mm.backdrop_path, '') as backdrop_path,
			mm.vote_average,
			COALESCE(mm.genres, '[]') as genres
		FROM media_files mf
		LEFT JOIN movie_metadata mm ON mf.movie_metadata_id = mm.id
		LEFT JOIN (
			SELECT movie_metadata_id, title, tagline, overview,
				ROW_NUMBER() OVER (PARTITION BY movie_metadata_id ORDER BY 
					CASE 
						WHEN language = ? THEN 0 
						WHEN language LIKE ? || '%' THEN 1 
						WHEN language = 'en' THEN 2
						WHEN language LIKE 'en%' THEN 3
						ELSE 4 
					END
				) as rn
			FROM movie_metadata_translations
			WHERE language = ? OR language LIKE ? || '%' OR language = 'en' OR language LIKE 'en%'
		) mt ON mm.id = mt.movie_metadata_id AND mt.rn = 1
		LEFT JOIN (
			SELECT media_id, 
				CASE WHEN SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) > 0 THEN 'completed'
					 WHEN SUM(CASE WHEN status IN ('pending', 'processing') THEN 1 ELSE 0 END) > 0 THEN 'pending'
					 ELSE 'none' END as transcode_status
			FROM transcodes
			GROUP BY media_id
		) t ON mf.id = t.media_id
		WHERE mf.id = ?
	`

	var m ports.MovieSummary
	var movieMetadataID sql.NullInt64
	var originalTitle, genresJSON string

	err := r.db.QueryRowContext(ctx, query, exactLang, baseLang, exactLang, baseLang, mediaID).Scan(
		&m.MediaID,
		&m.Duration,
		&m.Resolution,
		&m.Status,
		&m.EnrichmentStatus,
		&m.CreatedAt,
		&movieMetadataID,
		&m.TranscodeStatus,
		&originalTitle,
		&m.Title,
		&m.Year,
		&m.PosterPath,
		&m.BackdropPath,
		&m.VoteAverage,
		&genresJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	if movieMetadataID.Valid {
		m.MovieMetadataID = &movieMetadataID.Int64
	}
	m.Genres = parseGenresJSON(genresJSON)

	return &m, nil
}

// GetMovieDetail returns a movie with full details including credits and certification
func (r *LibraryRepository) GetMovieDetail(ctx context.Context, mediaID string, language string, maxCast int) (*ports.MovieDetail, error) {
	exactLang, baseLang := languageParams(language)

	// Query movie with extended metadata
	// Translation join uses subquery with priority: exact lang > base lang > English > English variant
	query := `
		SELECT 
			mf.id,
			mf.duration,
			mf.resolution,
			mf.status,
			mf.enrichment_status,
			mf.created_at,
			mf.movie_metadata_id,
			COALESCE(t.transcode_status, 'none') as transcode_status,
			mm.original_title,
			COALESCE(mt.title, mm.original_title) as title,
			COALESCE(SUBSTR(mm.release_date, 1, 4), '') as year,
			COALESCE(mm.poster_path, '') as poster_path,
			COALESCE(mm.backdrop_path, '') as backdrop_path,
			mm.vote_average,
			COALESCE(mm.genres, '[]') as genres,
			COALESCE(mt.tagline, '') as tagline,
			COALESCE(mt.overview, '') as overview,
			COALESCE(mm.release_date, '') as release_date,
			mm.runtime,
			COALESCE(mm.status, '') as movie_status,
			COALESCE(mm.imdb_id, '') as imdb_id,
			mm.tmdb_id
		FROM media_files mf
		LEFT JOIN movie_metadata mm ON mf.movie_metadata_id = mm.id
		LEFT JOIN (
			SELECT movie_metadata_id, title, tagline, overview,
				ROW_NUMBER() OVER (PARTITION BY movie_metadata_id ORDER BY 
					CASE 
						WHEN language = ? THEN 0 
						WHEN language LIKE ? || '%' THEN 1 
						WHEN language = 'en' THEN 2
						WHEN language LIKE 'en%' THEN 3
						ELSE 4 
					END
				) as rn
			FROM movie_metadata_translations
			WHERE language = ? OR language LIKE ? || '%' OR language = 'en' OR language LIKE 'en%'
		) mt ON mm.id = mt.movie_metadata_id AND mt.rn = 1
		LEFT JOIN (
			SELECT media_id, 
				CASE WHEN SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) > 0 THEN 'completed'
					 WHEN SUM(CASE WHEN status IN ('pending', 'processing') THEN 1 ELSE 0 END) > 0 THEN 'pending'
					 ELSE 'none' END as transcode_status
			FROM transcodes
			GROUP BY media_id
		) t ON mf.id = t.media_id
		WHERE mf.id = ?
	`

	var detail ports.MovieDetail
	var movieMetadataID sql.NullInt64
	var genresJSON string
	var runtime sql.NullInt64
	var tmdbID sql.NullInt64
	var mediaStatus, movieStatus string

	err := r.db.QueryRowContext(ctx, query, exactLang, baseLang, exactLang, baseLang, mediaID).Scan(
		&detail.MediaID,
		&detail.Duration,
		&detail.Resolution,
		&mediaStatus,
		&detail.EnrichmentStatus,
		&detail.CreatedAt,
		&movieMetadataID,
		&detail.TranscodeStatus,
		&detail.OriginalTitle,
		&detail.Title,
		&detail.Year,
		&detail.PosterPath,
		&detail.BackdropPath,
		&detail.VoteAverage,
		&genresJSON,
		&detail.Tagline,
		&detail.Overview,
		&detail.ReleaseDate,
		&runtime,
		&movieStatus,
		&detail.IMDbID,
		&tmdbID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get movie detail: %w", err)
	}

	if movieMetadataID.Valid {
		detail.MovieMetadataID = &movieMetadataID.Int64
	}
	if runtime.Valid {
		detail.Runtime = int(runtime.Int64)
	}
	if tmdbID.Valid {
		detail.TMDBID = int(tmdbID.Int64)
	}
	detail.MovieSummary.Status = mediaStatus
	detail.MovieStatus = movieStatus
	detail.Genres = parseGenresJSON(genresJSON)

	// If no metadata linked, return early
	if detail.MovieMetadataID == nil {
		return &detail, nil
	}

	metadataID := *detail.MovieMetadataID

	// Fetch cast (limited by maxCast)
	castQuery := `
		SELECT id, tmdb_person_id, name, character, profile_path
		FROM movie_credits
		WHERE movie_metadata_id = ? AND credit_type = 'cast'
		ORDER BY display_order
		LIMIT ?
	`
	castRows, err := r.db.QueryContext(ctx, castQuery, metadataID, maxCast)
	if err != nil {
		return nil, fmt.Errorf("failed to query cast: %w", err)
	}
	defer castRows.Close()

	for castRows.Next() {
		var c ports.CreditPerson
		var profilePath sql.NullString
		var character sql.NullString
		err := castRows.Scan(&c.ID, &c.TMDBPersonID, &c.Name, &character, &profilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cast: %w", err)
		}
		if character.Valid {
			c.Character = character.String
		}
		if profilePath.Valid {
			c.ProfilePath = profilePath.String
		}
		detail.Cast = append(detail.Cast, c)
	}
	if err := castRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating cast: %w", err)
	}

	// Fetch directors
	directorQuery := `
		SELECT id, tmdb_person_id, name, job, profile_path
		FROM movie_credits
		WHERE movie_metadata_id = ? AND credit_type = 'crew' AND job = 'Director'
		ORDER BY display_order
	`
	directorRows, err := r.db.QueryContext(ctx, directorQuery, metadataID)
	if err != nil {
		return nil, fmt.Errorf("failed to query directors: %w", err)
	}
	defer directorRows.Close()

	for directorRows.Next() {
		var c ports.CreditPerson
		var profilePath sql.NullString
		var job sql.NullString
		err := directorRows.Scan(&c.ID, &c.TMDBPersonID, &c.Name, &job, &profilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to scan director: %w", err)
		}
		if job.Valid {
			c.Job = job.String
		}
		if profilePath.Valid {
			c.ProfilePath = profilePath.String
		}
		detail.Directors = append(detail.Directors, c)
	}
	if err := directorRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating directors: %w", err)
	}

	// Fetch other crew (non-directors)
	crewQuery := `
		SELECT id, tmdb_person_id, name, job, profile_path
		FROM movie_credits
		WHERE movie_metadata_id = ? AND credit_type = 'crew' AND job != 'Director'
		ORDER BY display_order
	`
	crewRows, err := r.db.QueryContext(ctx, crewQuery, metadataID)
	if err != nil {
		return nil, fmt.Errorf("failed to query crew: %w", err)
	}
	defer crewRows.Close()

	for crewRows.Next() {
		var c ports.CreditPerson
		var profilePath sql.NullString
		var job sql.NullString
		err := crewRows.Scan(&c.ID, &c.TMDBPersonID, &c.Name, &job, &profilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to scan crew: %w", err)
		}
		if job.Valid {
			c.Job = job.String
		}
		if profilePath.Valid {
			c.ProfilePath = profilePath.String
		}
		detail.Crew = append(detail.Crew, c)
	}
	if err := crewRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating crew: %w", err)
	}

	// Fetch certifications and select appropriate one
	certQuery := `
		SELECT country, certification
		FROM movie_certifications
		WHERE movie_metadata_id = ?
	`
	certRows, err := r.db.QueryContext(ctx, certQuery, metadataID)
	if err != nil {
		return nil, fmt.Errorf("failed to query certifications: %w", err)
	}
	defer certRows.Close()

	var certifications []domain.MovieCertification
	for certRows.Next() {
		var cert domain.MovieCertification
		err := certRows.Scan(&cert.Country, &cert.Certification)
		if err != nil {
			return nil, fmt.Errorf("failed to scan certification: %w", err)
		}
		certifications = append(certifications, cert)
	}
	if err := certRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating certifications: %w", err)
	}

	// Use domain function to get certification with fallback
	detail.Certification = domain.GetCertificationWithFallback(certifications, baseLang)

	return &detail, nil
}

// ListSeries returns series with summary info, sorted by name
func (r *LibraryRepository) ListSeries(ctx context.Context, language string, includeEmpty bool) ([]ports.SeriesSummary, error) {
	exactLang, baseLang := languageParams(language)

	// Query series with episode counts
	// Translation join uses subquery with priority: exact lang > base lang > English > English variant
	query := `
		SELECT 
			sm.id,
			sm.tmdb_id,
			COALESCE(st.name, sm.original_name) as name,
			COALESCE(SUBSTR(sm.first_air_date, 1, 4), '') as year,
			COALESCE(sm.poster_path, '') as poster_path,
			COALESCE(sm.backdrop_path, '') as backdrop_path,
			sm.vote_average,
			COALESCE(sm.genres, '[]') as genres,
			sm.number_of_seasons,
			sm.number_of_episodes,
			COALESCE(episode_count.available, 0) as available_episodes
		FROM series_metadata sm
		LEFT JOIN (
			SELECT series_metadata_id, name, overview,
				ROW_NUMBER() OVER (PARTITION BY series_metadata_id ORDER BY 
					CASE 
						WHEN language = ? THEN 0 
						WHEN language LIKE ? || '%' THEN 1 
						WHEN language = 'en' THEN 2
						WHEN language LIKE 'en%' THEN 3
						ELSE 4 
					END
				) as rn
			FROM series_metadata_translations
			WHERE language = ? OR language LIKE ? || '%' OR language = 'en' OR language LIKE 'en%'
		) st ON sm.id = st.series_metadata_id AND st.rn = 1
		LEFT JOIN (
			SELECT 
				ssm.series_id,
				COUNT(DISTINCT mf.id) as available
			FROM season_metadata ssm
			JOIN episode_metadata em ON ssm.id = em.season_id
			JOIN media_files mf ON mf.episode_metadata_id = em.id
			GROUP BY ssm.series_id
		) episode_count ON sm.id = episode_count.series_id
	`

	if !includeEmpty {
		query += ` WHERE episode_count.available > 0`
	}

	query += ` ORDER BY COALESCE(st.name, sm.original_name)`

	rows, err := r.db.QueryContext(ctx, query, exactLang, baseLang, exactLang, baseLang)
	if err != nil {
		return nil, fmt.Errorf("failed to query series: %w", err)
	}
	defer rows.Close()

	var series []ports.SeriesSummary
	for rows.Next() {
		var s ports.SeriesSummary
		var genresJSON string

		err := rows.Scan(
			&s.SeriesMetadataID,
			&s.TMDBID,
			&s.Name,
			&s.Year,
			&s.PosterPath,
			&s.BackdropPath,
			&s.VoteAverage,
			&genresJSON,
			&s.NumberOfSeasons,
			&s.NumberOfEpisodes,
			&s.AvailableEpisodes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan series: %w", err)
		}

		s.Genres = parseGenresJSON(genresJSON)
		series = append(series, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating series: %w", err)
	}

	return series, nil
}

// GetSeriesDetail returns a series with all seasons and episodes
func (r *LibraryRepository) GetSeriesDetail(ctx context.Context, seriesID int64, language string) (*ports.SeriesDetail, error) {
	exactLang, baseLang := languageParams(language)

	// Get series info with translation fallback (priority: exact lang > base lang > English)
	seriesQuery := `
		SELECT 
			sm.id,
			sm.tmdb_id,
			COALESCE(st.name, sm.original_name) as name,
			COALESCE(SUBSTR(sm.first_air_date, 1, 4), '') as year,
			COALESCE(sm.poster_path, '') as poster_path,
			COALESCE(sm.backdrop_path, '') as backdrop_path,
			sm.vote_average,
			COALESCE(sm.genres, '[]') as genres,
			sm.number_of_seasons,
			sm.number_of_episodes,
			COALESCE(st.overview, '') as overview
		FROM series_metadata sm
		LEFT JOIN (
			SELECT series_metadata_id, name, overview,
				ROW_NUMBER() OVER (PARTITION BY series_metadata_id ORDER BY 
					CASE 
						WHEN language = ? THEN 0 
						WHEN language LIKE ? || '%' THEN 1 
						WHEN language = 'en' THEN 2
						WHEN language LIKE 'en%' THEN 3
						ELSE 4 
					END
				) as rn
			FROM series_metadata_translations
			WHERE language = ? OR language LIKE ? || '%' OR language = 'en' OR language LIKE 'en%'
		) st ON sm.id = st.series_metadata_id AND st.rn = 1
		WHERE sm.id = ?
	`

	var detail ports.SeriesDetail
	var genresJSON string

	err := r.db.QueryRowContext(ctx, seriesQuery, exactLang, baseLang, exactLang, baseLang, seriesID).Scan(
		&detail.SeriesMetadataID,
		&detail.TMDBID,
		&detail.Name,
		&detail.Year,
		&detail.PosterPath,
		&detail.BackdropPath,
		&detail.VoteAverage,
		&genresJSON,
		&detail.NumberOfSeasons,
		&detail.NumberOfEpisodes,
		&detail.Overview,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	detail.Genres = parseGenresJSON(genresJSON)

	// Get seasons with translation fallback (priority: exact lang > base lang > English)
	seasonQuery := `
		SELECT 
			ssm.id,
			ssm.season_number,
			COALESCE(sst.name, '') as name,
			COALESCE(sst.overview, '') as overview,
			COALESCE(ssm.poster_path, '') as poster_path,
			COALESCE(ssm.air_date, '') as air_date,
			ssm.episode_count
		FROM season_metadata ssm
		LEFT JOIN (
			SELECT season_metadata_id, name, overview,
				ROW_NUMBER() OVER (PARTITION BY season_metadata_id ORDER BY 
					CASE 
						WHEN language = ? THEN 0 
						WHEN language LIKE ? || '%' THEN 1 
						WHEN language = 'en' THEN 2
						WHEN language LIKE 'en%' THEN 3
						ELSE 4 
					END
				) as rn
			FROM season_metadata_translations
			WHERE language = ? OR language LIKE ? || '%' OR language = 'en' OR language LIKE 'en%'
		) sst ON ssm.id = sst.season_metadata_id AND sst.rn = 1
		WHERE ssm.series_id = ?
		ORDER BY ssm.season_number
	`

	seasonRows, err := r.db.QueryContext(ctx, seasonQuery, exactLang, baseLang, exactLang, baseLang, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to query seasons: %w", err)
	}
	defer seasonRows.Close()

	seasonMap := make(map[int64]*ports.SeasonSummary)
	var seasonIDs []int64

	for seasonRows.Next() {
		var s ports.SeasonSummary
		err := seasonRows.Scan(
			&s.SeasonMetadataID,
			&s.SeasonNumber,
			&s.Name,
			&s.Overview,
			&s.PosterPath,
			&s.AirDate,
			&s.EpisodeCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan season: %w", err)
		}

		seasonMap[s.SeasonMetadataID] = &s
		seasonIDs = append(seasonIDs, s.SeasonMetadataID)
		detail.Seasons = append(detail.Seasons, s)
	}

	if err := seasonRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating seasons: %w", err)
	}

	if len(seasonIDs) == 0 {
		detail.AvailableEpisodes = 0
		return &detail, nil
	}

	// Get episodes for all seasons with translation fallback (priority: exact lang > base lang > English)
	placeholders := make([]string, len(seasonIDs))
	args := make([]interface{}, 0, len(seasonIDs)+4)
	// First 4 args are for the translation subquery
	args = append(args, exactLang, baseLang, exactLang, baseLang)
	for i, id := range seasonIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	episodeQuery := fmt.Sprintf(`
		SELECT 
			em.id,
			em.season_id,
			ssm.season_number,
			em.episode_number,
			COALESCE(et.name, '') as name,
			COALESCE(et.overview, '') as overview,
			COALESCE(em.air_date, '') as air_date,
			COALESCE(em.still_path, '') as still_path,
			em.vote_average,
			em.runtime,
			mf.id as media_id,
			mf.status as media_status,
			COALESCE(t.transcode_status, 'none') as transcode_status
		FROM episode_metadata em
		JOIN season_metadata ssm ON em.season_id = ssm.id
		LEFT JOIN (
			SELECT episode_metadata_id, name, overview,
				ROW_NUMBER() OVER (PARTITION BY episode_metadata_id ORDER BY 
					CASE 
						WHEN language = ? THEN 0 
						WHEN language LIKE ? || '%%' THEN 1 
						WHEN language = 'en' THEN 2
						WHEN language LIKE 'en%%' THEN 3
						ELSE 4 
					END
				) as rn
			FROM episode_metadata_translations
			WHERE language = ? OR language LIKE ? || '%%' OR language = 'en' OR language LIKE 'en%%'
		) et ON em.id = et.episode_metadata_id AND et.rn = 1
		LEFT JOIN media_files mf ON mf.episode_metadata_id = em.id
		LEFT JOIN (
			SELECT media_id, 
				CASE WHEN SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) > 0 THEN 'completed'
					 WHEN SUM(CASE WHEN status IN ('pending', 'processing') THEN 1 ELSE 0 END) > 0 THEN 'pending'
					 ELSE 'none' END as transcode_status
			FROM transcodes
			GROUP BY media_id
		) t ON mf.id = t.media_id
		WHERE em.season_id IN (%s)
		ORDER BY ssm.season_number, em.episode_number
	`, strings.Join(placeholders, ","))

	episodeRows, err := r.db.QueryContext(ctx, episodeQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query episodes: %w", err)
	}
	defer episodeRows.Close()

	availableCount := 0
	for episodeRows.Next() {
		var e ports.EpisodeSummary
		var seasonID int64
		var mediaID sql.NullString
		var mediaStatus, transcodeStatus sql.NullString

		err := episodeRows.Scan(
			&e.EpisodeMetadataID,
			&seasonID,
			&e.SeasonNumber,
			&e.EpisodeNumber,
			&e.Name,
			&e.Overview,
			&e.AirDate,
			&e.StillPath,
			&e.VoteAverage,
			&e.Duration,
			&mediaID,
			&mediaStatus,
			&transcodeStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan episode: %w", err)
		}

		if mediaID.Valid {
			e.MediaID = &mediaID.String
			e.Status = mediaStatus.String
			e.TranscodeStatus = transcodeStatus.String
			availableCount++
		}

		// Find season and append episode
		for i := range detail.Seasons {
			if detail.Seasons[i].SeasonMetadataID == seasonID {
				detail.Seasons[i].Episodes = append(detail.Seasons[i].Episodes, e)
				break
			}
		}
	}

	if err := episodeRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating episodes: %w", err)
	}

	detail.AvailableEpisodes = availableCount

	return &detail, nil
}

// ListRecentlyAdded returns the most recently added media (movies and seasons)
// Movies are returned as individual items, episodes are grouped by season
func (r *LibraryRepository) ListRecentlyAdded(ctx context.Context, language string, limit int) ([]ports.RecentlyAddedItem, error) {
	exactLang, baseLang := languageParams(language)

	// UNION query: movies as individual items + episodes grouped by season
	// Both parts return the same columns for UNION compatibility
	// Translation subqueries use priority: exact lang > base lang > English > English variant
	query := `
		SELECT * FROM (
			-- Movies: one row per movie
			SELECT 
				'movie' as item_type,
				COALESCE(mt.title, mm.original_title, mf.filename) as title,
				COALESCE(SUBSTR(mm.release_date, 1, 4), '') as year,
				COALESCE(mm.poster_path, '') as poster_path,
				COALESCE(mm.backdrop_path, '') as backdrop_path,
				COALESCE(mm.vote_average, 0) as vote_average,
				mf.created_at,
				mf.id as media_id,
				mm.id as movie_metadata_id,
				COALESCE(t.transcode_status, 'none') as transcode_status,
				NULL as series_metadata_id,
				NULL as season_number,
				1 as episode_count
			FROM media_files mf
			JOIN movie_metadata mm ON mf.movie_metadata_id = mm.id
			LEFT JOIN (
				SELECT movie_metadata_id, title,
					ROW_NUMBER() OVER (PARTITION BY movie_metadata_id ORDER BY 
						CASE 
							WHEN language = ? THEN 0 
							WHEN language LIKE ? || '%' THEN 1 
							WHEN language = 'en' THEN 2
							WHEN language LIKE 'en%' THEN 3
							ELSE 4 
						END
					) as rn
				FROM movie_metadata_translations
				WHERE language = ? OR language LIKE ? || '%' OR language = 'en' OR language LIKE 'en%'
			) mt ON mm.id = mt.movie_metadata_id AND mt.rn = 1
			LEFT JOIN (
				SELECT media_id, 
					CASE WHEN SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) > 0 THEN 'completed'
						 WHEN SUM(CASE WHEN status IN ('pending', 'processing') THEN 1 ELSE 0 END) > 0 THEN 'pending'
						 ELSE 'none' END as transcode_status
				FROM transcodes
				GROUP BY media_id
			) t ON mf.id = t.media_id
			WHERE mf.metadata_type = 'movie' AND mf.enrichment_status IN ('linked', 'auto_linked')

			UNION ALL

			-- Seasons: one row per (series, season) with episode count
			SELECT 
				'season' as item_type,
				COALESCE(st.name, sm.original_name, '') || ' - Season ' || ssm.season_number as title,
				COALESCE(SUBSTR(sm.first_air_date, 1, 4), '') as year,
				COALESCE(ssm.poster_path, sm.poster_path, '') as poster_path,
				COALESCE(sm.backdrop_path, '') as backdrop_path,
				COALESCE(sm.vote_average, 0) as vote_average,
				MAX(mf.created_at) as created_at,
				NULL as media_id,
				NULL as movie_metadata_id,
				'' as transcode_status,
				sm.id as series_metadata_id,
				ssm.season_number,
				COUNT(*) as episode_count
			FROM media_files mf
			JOIN episode_metadata em ON mf.episode_metadata_id = em.id
			JOIN season_metadata ssm ON em.season_id = ssm.id
			JOIN series_metadata sm ON ssm.series_id = sm.id
			LEFT JOIN (
				SELECT series_metadata_id, name,
					ROW_NUMBER() OVER (PARTITION BY series_metadata_id ORDER BY 
						CASE 
							WHEN language = ? THEN 0 
							WHEN language LIKE ? || '%' THEN 1 
							WHEN language = 'en' THEN 2
							WHEN language LIKE 'en%' THEN 3
							ELSE 4 
						END
					) as rn
				FROM series_metadata_translations
				WHERE language = ? OR language LIKE ? || '%' OR language = 'en' OR language LIKE 'en%'
			) st ON sm.id = st.series_metadata_id AND st.rn = 1
			WHERE mf.metadata_type = 'episode' AND mf.enrichment_status IN ('linked', 'auto_linked')
			GROUP BY sm.id, ssm.season_number
		)
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query,
		exactLang, baseLang, exactLang, baseLang, // movie translations
		exactLang, baseLang, exactLang, baseLang, // series translations
		limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recently added: %w", err)
	}
	defer rows.Close()

	var items []ports.RecentlyAddedItem
	for rows.Next() {
		var item ports.RecentlyAddedItem
		var mediaID, transcodeStatus sql.NullString
		var movieMetadataID, seriesMetadataID sql.NullInt64
		var seasonNumber sql.NullInt64

		err := rows.Scan(
			&item.Type,
			&item.Title,
			&item.Year,
			&item.PosterPath,
			&item.BackdropPath,
			&item.VoteAverage,
			&item.CreatedAt,
			&mediaID,
			&movieMetadataID,
			&transcodeStatus,
			&seriesMetadataID,
			&seasonNumber,
			&item.EpisodeCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recently added: %w", err)
		}

		if mediaID.Valid {
			item.MediaID = mediaID.String
		}
		if movieMetadataID.Valid {
			item.MovieMetadataID = &movieMetadataID.Int64
		}
		if transcodeStatus.Valid {
			item.TranscodeStatus = transcodeStatus.String
		}
		if seriesMetadataID.Valid {
			item.SeriesMetadataID = &seriesMetadataID.Int64
		}
		if seasonNumber.Valid {
			sn := int(seasonNumber.Int64)
			item.SeasonNumber = &sn
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating recently added: %w", err)
	}

	return items, nil
}

// ListUnmatched returns media files that don't have metadata linked
func (r *LibraryRepository) ListUnmatched(ctx context.Context, limit, offset int) ([]ports.UnmatchedMediaSummary, int, error) {
	// Count total unmatched
	countQuery := `
		SELECT COUNT(*) 
		FROM media_files mf
		WHERE mf.enrichment_status NOT IN ('linked', 'auto_linked')
	`
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count unmatched: %w", err)
	}

	query := `
		SELECT 
			mf.id,
			mf.filename,
			COALESCE(mf.title, mf.filename) as title,
			mf.duration,
			mf.resolution,
			mf.enrichment_status,
			mf.created_at
		FROM media_files mf
		WHERE mf.enrichment_status NOT IN ('linked', 'auto_linked')
		ORDER BY mf.created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query unmatched: %w", err)
	}
	defer rows.Close()

	var items []ports.UnmatchedMediaSummary
	for rows.Next() {
		var m ports.UnmatchedMediaSummary
		err := rows.Scan(
			&m.MediaID,
			&m.Filename,
			&m.Title,
			&m.Duration,
			&m.Resolution,
			&m.EnrichmentStatus,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan unmatched: %w", err)
		}
		items = append(items, m)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating unmatched: %w", err)
	}

	return items, total, nil
}

// languageParams returns the exact language and base language for fallback queries.
// E.g., "en-US" returns ("en-US", "en"), "de" returns ("de", "de")
func languageParams(lang string) (exact string, base string) {
	if lang == "" {
		return "en", "en"
	}
	if idx := strings.IndexAny(lang, "-_"); idx > 0 {
		return lang, lang[:idx]
	}
	return lang, lang
}

// parseGenresJSON parses a JSON array of genres into a comma-separated string
func parseGenresJSON(json string) string {
	// Simple parsing: remove brackets and quotes
	json = strings.TrimPrefix(json, "[")
	json = strings.TrimSuffix(json, "]")
	json = strings.ReplaceAll(json, "\"", "")
	return json
}
