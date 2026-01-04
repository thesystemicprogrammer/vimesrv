import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { AuthService } from './auth.service';

// API Response wrapper
export interface ApiResponse<T> {
  data: T;
}

// Auth types
export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  expires_in: number;
}

export interface StreamTokenResponse {
  token: string;
  expires_in: number;
}

export interface UserInfo {
  username: string;
}

// Media types
export interface MediaListItem {
  id: string;
  title: string;
  filename: string;
  duration: number;
  resolution: string;
  status: string;
  has_subtitles: boolean;
  audio_tracks: number;
  subtitle_tracks: number;
}

export interface MediaListResponse {
  items: MediaListItem[];
  total: number;
  page: number;
  per_page: number;
}

export interface AudioStream {
  index: number;
  language: string;
  title: string;
  channels: number;
}

export interface SubtitleStream {
  index: number;
  language: string;
  title: string;
}

export interface MediaDetail {
  id: string;
  title: string;
  filename: string;
  duration: number;
  resolution: string;
  width: number;
  height: number;
  status: string;
  dash_manifest_url: string;
  audio_streams: AudioStream[];
  subtitle_streams: SubtitleStream[];
  available_qualities: string[];
}

// Library types
export interface MovieSummary {
  media_id: string;
  duration: number;
  resolution: string;
  status: string;
  enrichment_status: string;
  created_at: string;
  transcode_status: 'none' | 'pending' | 'completed';
  movie_metadata_id?: number;
  title: string;
  year?: string;
  poster_path?: string;
  backdrop_path?: string;
  vote_average: number;
  genres?: string;
}

export interface CreditPerson {
  id: number;
  tmdb_person_id: number;
  name: string;
  character?: string;
  job?: string;
  profile_path?: string;
}

export interface MovieDetail extends MovieSummary {
  original_title?: string;
  tagline?: string;
  overview?: string;
  release_date?: string;
  runtime?: number;
  movie_status?: string;
  imdb_id?: string;
  tmdb_id?: number;
  certification?: string;
  cast?: CreditPerson[];
  directors?: CreditPerson[];
  crew?: CreditPerson[];
  similar_movies?: SimilarMovieItem[];
  collection?: MovieCollectionInfo;
  audio_languages?: string[];
  subtitle_languages?: string[];
}

export interface SeriesSummary {
  series_metadata_id: number;
  tmdb_id: number;
  name: string;
  year?: string;
  poster_path?: string;
  backdrop_path?: string;
  vote_average: number;
  genres?: string;
  number_of_seasons: number;
  number_of_episodes: number;
  available_episodes: number;
}

export interface EpisodeSummary {
  media_id?: string;
  duration: number;
  status?: string;
  transcode_status?: string;
  episode_metadata_id: number;
  season_number: number;
  episode_number: number;
  name: string;
  overview?: string;
  air_date?: string;
  still_path?: string;
  vote_average: number;
  audio_languages?: string[];
  subtitle_languages?: string[];
}

export interface SeasonSummary {
  season_metadata_id: number;
  season_number: number;
  name: string;
  overview?: string;
  poster_path?: string;
  air_date?: string;
  episode_count: number;
  episodes?: EpisodeSummary[];
}

export interface SimilarMovieItem {
  tmdb_id: number;
  title: string;
  poster_path?: string;
  release_date?: string;
  year?: string;
  vote_average: number;
  in_library: boolean;
  media_id?: string;
}

export interface CollectionMovieItem {
  tmdb_id: number;
  title: string;
  poster_path?: string;
  release_date?: string;
  year?: string;
  vote_average: number;
  in_library: boolean;
  media_id?: string;
  is_current: boolean;
}

export interface MovieCollectionInfo {
  collection_id: number;
  name: string;
  overview?: string;
  poster_path?: string;
  backdrop_path?: string;
  movies: CollectionMovieItem[];
  position: number;
  total_movies: number;
}

export interface SimilarSeriesItem {
  tmdb_id: number;
  name: string;
  poster_path?: string;
  first_air_date?: string;
  year?: string;
  vote_average: number;
  in_library: boolean;
  series_metadata_id?: number;
}

export interface SeriesDetail extends SeriesSummary {
  overview?: string;
  seasons?: SeasonSummary[];
  similar_series?: SimilarSeriesItem[];
}

// Credits response types for full cast/crew pages
export interface MovieCreditsResponse {
  cast: CreditPerson[];
  crew: CreditPerson[];
}

export interface SeriesCreditPerson {
  id: number;
  tmdb_person_id: number;
  name: string;
  roles?: string;  // JSON string of roles for cast
  jobs?: string;   // JSON string of jobs for crew
  department?: string;
  profile_path?: string;
  total_episode_count: number;
}

export interface SeriesCreditsResponse {
  cast: SeriesCreditPerson[];
  crew: SeriesCreditPerson[];
}

export interface UnmatchedMediaSummary {
  media_id: string;
  filename: string;
  title: string;
  duration: number;
  resolution: string;
  enrichment_status: string;
  created_at: string;
}

// RecentlyAddedItem represents a recently added item (movie or season)
export interface RecentlyAddedItem {
  // Type discriminator: "movie" or "season"
  type: 'movie' | 'season';

  // Common fields
  title: string;
  year?: string;
  poster_path?: string;
  backdrop_path?: string;
  vote_average: number;
  created_at: string;

  // Movie-specific fields (when type === "movie")
  media_id?: string;
  movie_metadata_id?: number;
  transcode_status?: string;

  // Season-specific fields (when type === "season")
  series_metadata_id?: number;
  season_number?: number;
  episode_count?: number;
}

export interface MoviesListResponse {
  items: MovieSummary[];
  total: number;
  page: number;
  per_page: number;
}

export interface SeriesListResponse {
  items: SeriesSummary[];
  total: number;
  page: number;
  per_page: number;
}

// Filter options for movies and series lists
export type SortBy = 'date_added' | 'title' | 'year' | 'rating';
export type SortOrder = 'asc' | 'desc';

export interface ListFilterOptions {
  page?: number;
  perPage?: number;
  sortBy?: SortBy;
  sortOrder?: SortOrder;
  genres?: string[];
  yearFrom?: number;
  yearTo?: number;
  minRating?: number;
}

export interface GenresResponse {
  movie_genres: string[];
  series_genres: string[];
}

// Library search types
export interface LibrarySearchResult {
  type: 'movie' | 'series';
  media_id?: string;
  series_metadata_id?: number;
  movie_metadata_id?: number;
  title: string;
  year?: string;
  poster_path?: string;
  vote_average: number;
}

export interface LibrarySearchResponse {
  query: string;
  results: LibrarySearchResult[];
  count: number;
}

export interface RecentListResponse {
  items: RecentlyAddedItem[];
}

export interface UnmatchedListResponse {
  items: UnmatchedMediaSummary[];
  total: number;
  page: number;
  per_page: number;
}

// Metadata types
export interface MetadataCandidate {
  id: number;
  tmdb_id: number;
  media_type: 'movie' | 'tv';
  title: string;
  release_date?: string;
  confidence: number;
  poster_url?: string;
}

export interface SearchResult {
  tmdb_id: number;
  media_type: 'movie' | 'tv';
  title: string;
  original_title?: string;
  release_date?: string;
  overview?: string;
  poster_url?: string;
}

export interface SearchRequest {
  query: string;
  year?: number;
  media_type?: 'movie' | 'tv';
  max_results?: number;
  language?: string;
}

export interface LinkSearchRequest {
  tmdb_id: number;
  media_type: 'movie' | 'tv';
  season_number?: number;
  episode_number?: number;
}

export interface SearchMetadataResponse {
  media_id: string;
  query: string;
  results: SearchResult[];
  count: number;
}

// User management types
export type UserRole = 'admin' | 'manager' | 'user';

export interface User {
  id: string;
  username: string;
  role: UserRole;
  must_change_password: boolean;
  created_at: string;
  updated_at: string;
  created_by?: string;
}

export interface CreateUserRequest {
  username: string;
  password: string;
  role: UserRole;
}

export interface UpdateUserRequest {
  role: UserRole;
}

export interface ResetPasswordRequest {
  new_password: string;
}

export interface ChangePasswordRequest {
  current_password: string;
  new_password: string;
}

// Job types
export type JobStatus = 'queued' | 'running' | 'succeeded' | 'dead';
export type JobType = 'scan_library' | 'transcode_video' | 'transcode_audio' | 'enrich_metadata' | 'fetch_translations';

export interface Job {
  id: number;
  type: JobType;
  status: JobStatus;
  payload?: Record<string, unknown>;
  priority: number;
  attempts: number;
  max_attempts: number;
  last_error?: string;
  created_at: string;
  started_at?: string;
  finished_at?: string;
  updated_at: string;
}

export interface JobListResponse {
  jobs: Job[];
  total: number;
}

export interface JobListOptions {
  status?: JobStatus[];
  type?: JobType[];
  includeOld?: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private readonly http = inject(HttpClient);
  private readonly auth = inject(AuthService);
  private readonly baseUrl = '/api/v1';

  // Auth endpoints
  login(credentials: LoginRequest): Observable<ApiResponse<LoginResponse>> {
    return this.http.post<ApiResponse<LoginResponse>>(
      `${this.baseUrl}/auth/login`,
      credentials
    ).pipe(
      tap(response => {
        this.auth.setToken(response.data.token);
      })
    );
  }

  getMe(): Observable<ApiResponse<UserInfo>> {
    return this.http.get<ApiResponse<UserInfo>>(`${this.baseUrl}/auth/me`).pipe(
      tap(response => {
        this.auth.setUsername(response.data.username);
      })
    );
  }

  getStreamToken(): Observable<ApiResponse<StreamTokenResponse>> {
    return this.http.post<ApiResponse<StreamTokenResponse>>(
      `${this.baseUrl}/auth/stream-token`,
      {}
    ).pipe(
      tap(response => {
        this.auth.setStreamToken(response.data.token);
      })
    );
  }

  // Media endpoints
  listMedia(page = 1, perPage = 20): Observable<ApiResponse<MediaListResponse>> {
    const params = new HttpParams()
      .set('page', page.toString())
      .set('per_page', perPage.toString());

    return this.http.get<ApiResponse<MediaListResponse>>(
      `${this.baseUrl}/media`,
      { params }
    );
  }

  getMedia(id: string): Observable<ApiResponse<MediaDetail>> {
    return this.http.get<ApiResponse<MediaDetail>>(`${this.baseUrl}/media/${id}`);
  }

  // Library endpoints
  scanLibrary(): Observable<ApiResponse<{ job_id: number }>> {
    return this.http.post<ApiResponse<{ job_id: number }>>(
      `${this.baseUrl}/scanlib`,
      {}
    );
  }

  // Library browsing endpoints
  listMovies(options: ListFilterOptions = {}): Observable<ApiResponse<MoviesListResponse>> {
    const lang = this.auth.language();
    let params = new HttpParams()
      .set('page', (options.page ?? 1).toString())
      .set('per_page', (options.perPage ?? 20).toString())
      .set('lang', lang);

    if (options.sortBy) {
      params = params.set('sort', options.sortBy);
    }
    if (options.sortOrder) {
      params = params.set('order', options.sortOrder);
    }
    if (options.genres && options.genres.length > 0) {
      params = params.set('genres', options.genres.join(','));
    }
    if (options.yearFrom) {
      params = params.set('year_from', options.yearFrom.toString());
    }
    if (options.yearTo) {
      params = params.set('year_to', options.yearTo.toString());
    }
    if (options.minRating) {
      params = params.set('min_rating', options.minRating.toString());
    }

    return this.http.get<ApiResponse<MoviesListResponse>>(
      `${this.baseUrl}/library/movies`,
      { params }
    );
  }

  getMovie(mediaId: string): Observable<ApiResponse<MovieDetail>> {
    const lang = this.auth.language();
    const params = new HttpParams().set('lang', lang);

    return this.http.get<ApiResponse<MovieDetail>>(
      `${this.baseUrl}/library/movies/${mediaId}`,
      { params }
    );
  }

  getMovieCredits(movieMetadataId: number): Observable<ApiResponse<MovieCreditsResponse>> {
    return this.http.get<ApiResponse<MovieCreditsResponse>>(
      `${this.baseUrl}/library/movies/${movieMetadataId}/credits`
    );
  }

  listSeries(options: ListFilterOptions & { includeEmpty?: boolean } = {}): Observable<ApiResponse<SeriesListResponse>> {
    const lang = this.auth.language();
    let params = new HttpParams()
      .set('page', (options.page ?? 1).toString())
      .set('per_page', (options.perPage ?? 20).toString())
      .set('lang', lang)
      .set('include_empty', (options.includeEmpty ?? false).toString());

    if (options.sortBy) {
      params = params.set('sort', options.sortBy);
    }
    if (options.sortOrder) {
      params = params.set('order', options.sortOrder);
    }
    if (options.genres && options.genres.length > 0) {
      params = params.set('genres', options.genres.join(','));
    }
    if (options.yearFrom) {
      params = params.set('year_from', options.yearFrom.toString());
    }
    if (options.yearTo) {
      params = params.set('year_to', options.yearTo.toString());
    }
    if (options.minRating) {
      params = params.set('min_rating', options.minRating.toString());
    }

    return this.http.get<ApiResponse<SeriesListResponse>>(
      `${this.baseUrl}/library/series`,
      { params }
    );
  }

  getSeriesDetail(seriesId: number): Observable<ApiResponse<SeriesDetail>> {
    const lang = this.auth.language();
    const params = new HttpParams().set('lang', lang);

    return this.http.get<ApiResponse<SeriesDetail>>(
      `${this.baseUrl}/library/series/${seriesId}`,
      { params }
    );
  }

  getSeriesCredits(seriesMetadataId: number): Observable<ApiResponse<SeriesCreditsResponse>> {
    return this.http.get<ApiResponse<SeriesCreditsResponse>>(
      `${this.baseUrl}/library/series/${seriesMetadataId}/credits`
    );
  }

  listRecent(): Observable<ApiResponse<RecentListResponse>> {
    const lang = this.auth.language();
    const params = new HttpParams().set('lang', lang);

    return this.http.get<ApiResponse<RecentListResponse>>(
      `${this.baseUrl}/library/recent`,
      { params }
    );
  }

  listUnmatched(page = 1, perPage = 20): Observable<ApiResponse<UnmatchedListResponse>> {
    const params = new HttpParams()
      .set('page', page.toString())
      .set('per_page', perPage.toString());

    return this.http.get<ApiResponse<UnmatchedListResponse>>(
      `${this.baseUrl}/library/unmatched`,
      { params }
    );
  }

  listGenres(): Observable<ApiResponse<GenresResponse>> {
    return this.http.get<ApiResponse<GenresResponse>>(
      `${this.baseUrl}/library/genres`
    );
  }

  searchLibrary(query: string, limit = 20): Observable<ApiResponse<LibrarySearchResponse>> {
    const lang = this.auth.language();
    const params = new HttpParams()
      .set('q', query)
      .set('lang', lang)
      .set('limit', limit.toString());

    return this.http.get<ApiResponse<LibrarySearchResponse>>(
      `${this.baseUrl}/library/search`,
      { params }
    );
  }

  // Metadata endpoints
  getCandidates(mediaId: string, pendingOnly = false): Observable<ApiResponse<MetadataCandidate[]>> {
    const params = new HttpParams().set('pending_only', pendingOnly.toString());

    return this.http.get<ApiResponse<MetadataCandidate[]>>(
      `${this.baseUrl}/media/${mediaId}/candidates`,
      { params }
    );
  }

  searchMetadata(mediaId: string, request: SearchRequest): Observable<ApiResponse<SearchMetadataResponse>> {
    return this.http.post<ApiResponse<SearchMetadataResponse>>(
      `${this.baseUrl}/media/${mediaId}/search`,
      request
    );
  }

  linkCandidate(mediaId: string, candidateId: number): Observable<ApiResponse<object>> {
    return this.http.post<ApiResponse<object>>(
      `${this.baseUrl}/media/${mediaId}/link`,
      { candidate_id: candidateId }
    );
  }

  linkSearchResult(mediaId: string, request: LinkSearchRequest): Observable<ApiResponse<object>> {
    return this.http.post<ApiResponse<object>>(
      `${this.baseUrl}/media/${mediaId}/link-search`,
      request
    );
  }

  skipEnrichment(mediaId: string): Observable<ApiResponse<object>> {
    return this.http.post<ApiResponse<object>>(
      `${this.baseUrl}/media/${mediaId}/skip`,
      {}
    );
  }

  resetEnrichment(mediaId: string): Observable<ApiResponse<object>> {
    return this.http.post<ApiResponse<object>>(
      `${this.baseUrl}/media/${mediaId}/reset`,
      {}
    );
  }

  // Translation endpoints
  fetchTranslations(language: string): Observable<ApiResponse<{ message: string; job_id?: number; queued: boolean }>> {
    return this.http.post<ApiResponse<{ message: string; job_id?: number; queued: boolean }>>(
      `${this.baseUrl}/translations/fetch`,
      { language }
    );
  }

  // User management endpoints
  listUsers(): Observable<ApiResponse<User[]>> {
    return this.http.get<ApiResponse<User[]>>(`${this.baseUrl}/users`);
  }

  getUser(id: string): Observable<ApiResponse<User>> {
    return this.http.get<ApiResponse<User>>(`${this.baseUrl}/users/${id}`);
  }

  createUser(request: CreateUserRequest): Observable<ApiResponse<User>> {
    return this.http.post<ApiResponse<User>>(`${this.baseUrl}/users`, request);
  }

  updateUser(id: string, request: UpdateUserRequest): Observable<ApiResponse<User>> {
    return this.http.put<ApiResponse<User>>(`${this.baseUrl}/users/${id}`, request);
  }

  deleteUser(id: string): Observable<ApiResponse<{ message: string }>> {
    return this.http.delete<ApiResponse<{ message: string }>>(`${this.baseUrl}/users/${id}`);
  }

  resetUserPassword(id: string, request: ResetPasswordRequest): Observable<ApiResponse<{ message: string }>> {
    return this.http.post<ApiResponse<{ message: string }>>(
      `${this.baseUrl}/users/${id}/reset-password`,
      request
    );
  }

  changePassword(request: ChangePasswordRequest): Observable<ApiResponse<{ message: string; token: string }>> {
    return this.http.post<ApiResponse<{ message: string; token: string }>>(
      `${this.baseUrl}/auth/change-password`,
      request
    ).pipe(
      tap(response => {
        // Update the token with the new one (must_change_password will be false)
        this.auth.setToken(response.data.token);
      })
    );
  }

  // Job management endpoints
  listJobs(options: JobListOptions = {}): Observable<ApiResponse<JobListResponse>> {
    let params = new HttpParams();

    if (options.status && options.status.length > 0) {
      params = params.set('status', options.status.join(','));
    }
    if (options.type && options.type.length > 0) {
      params = params.set('type', options.type.join(','));
    }
    if (options.includeOld) {
      params = params.set('include_old', 'true');
    }

    return this.http.get<ApiResponse<JobListResponse>>(
      `${this.baseUrl}/jobs`,
      { params }
    );
  }

  // Helper to build stream URL with token
  getStreamUrl(manifestUrl: string): string {
    const token = this.auth.streamToken();
    if (!token) {
      return manifestUrl;
    }
    const separator = manifestUrl.includes('?') ? '&' : '?';
    return `${manifestUrl}${separator}token=${token}`;
  }
}
