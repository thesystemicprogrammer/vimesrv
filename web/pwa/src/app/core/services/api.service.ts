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

export interface SeriesDetail extends SeriesSummary {
  overview?: string;
  seasons?: SeasonSummary[];
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
}

export interface LinkSearchRequest {
  tmdb_id: number;
  media_type: 'movie' | 'tv';
  season_number?: number;
  episode_number?: number;
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
  listMovies(page = 1, perPage = 20): Observable<ApiResponse<MoviesListResponse>> {
    const lang = this.auth.language();
    const params = new HttpParams()
      .set('page', page.toString())
      .set('per_page', perPage.toString())
      .set('lang', lang);

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

  listSeries(includeEmpty = false): Observable<ApiResponse<SeriesListResponse>> {
    const lang = this.auth.language();
    const params = new HttpParams()
      .set('lang', lang)
      .set('include_empty', includeEmpty.toString());

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

  // Metadata endpoints
  getCandidates(mediaId: string, pendingOnly = false): Observable<ApiResponse<MetadataCandidate[]>> {
    const params = new HttpParams().set('pending_only', pendingOnly.toString());

    return this.http.get<ApiResponse<MetadataCandidate[]>>(
      `${this.baseUrl}/media/${mediaId}/candidates`,
      { params }
    );
  }

  searchMetadata(mediaId: string, request: SearchRequest): Observable<ApiResponse<SearchResult[]>> {
    return this.http.post<ApiResponse<SearchResult[]>>(
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
