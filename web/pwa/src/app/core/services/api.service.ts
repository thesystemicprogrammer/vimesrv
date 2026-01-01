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
