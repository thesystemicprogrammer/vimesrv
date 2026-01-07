import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { AuthService } from './auth.service';

// API Response wrapper
export interface ApiResponse<T> {
  data: T;
}

// Watch Progress Types
export interface WatchProgress {
  id: string;
  user_id: string;
  media_id?: string;
  episode_metadata_id?: number;
  position_seconds: number;
  duration_seconds: number;
  progress_percent: number;
  completed: boolean;
  manually_removed: boolean;
  last_watched_at: string;
  updated_at: string;
}

export interface ContinueWatchingItem {
  // Progress info
  id: string;
  user_id: string;
  media_id?: string;
  episode_metadata_id?: number;
  position_seconds: number;
  duration_seconds: number;
  progress_percent: number;
  completed: boolean;
  last_watched_at: string;
  
  // Enriched metadata
  title: string;
  poster_path?: string;
  backdrop_path?: string;
  media_type: 'movie' | 'episode';
  year?: number;
  
  // Episode-specific fields (when media_type = 'episode')
  series_name?: string;
  series_metadata_id?: number;
  season_number?: number;
  episode_number?: number;
  episode_title?: string;
}

export interface RecordProgressRequest {
  media_id?: string;
  episode_metadata_id?: number;
  position_seconds: number;
  duration_seconds: number;
}

@Injectable({
  providedIn: 'root'
})
export class WatchProgressService {
  private http = inject(HttpClient);
  private auth = inject(AuthService);
  private baseUrl = '/api/v1';

  // Signal to track continue watching items (cached)
  continueWatchingItems = signal<ContinueWatchingItem[]>([]);
  continueWatchingLoading = signal<boolean>(false);

  /**
   * Record watch progress for a movie or episode
   * This should be called periodically (e.g., every 10 seconds) during playback
   */
  recordProgress(request: RecordProgressRequest): Observable<ApiResponse<void>> {
    return this.http.post<ApiResponse<void>>(
      `${this.baseUrl}/playback/progress`,
      request
    ).pipe(
      tap(() => {
        // Invalidate cache when progress is recorded
        this.invalidateContinueWatchingCache();
      })
    );
  }

  /**
   * Get watch progress for a specific media item
   */
  getProgress(mediaId: string, episodeId?: number): Observable<ApiResponse<WatchProgress>> {
    let params = new HttpParams();
    if (episodeId !== undefined) {
      params = params.set('episode_id', episodeId.toString());
    }

    return this.http.get<ApiResponse<WatchProgress>>(
      `${this.baseUrl}/playback/progress/${mediaId}`,
      { params }
    );
  }

  /**
   * Get continue watching items (in-progress content)
   * Results are cached in a signal for UI reactivity
   */
  getContinueWatching(limit: number = 20): Observable<ApiResponse<ContinueWatchingItem[]>> {
    this.continueWatchingLoading.set(true);
    
    const params = new HttpParams().set('limit', limit.toString());

    return this.http.get<ApiResponse<ContinueWatchingItem[]>>(
      `${this.baseUrl}/library/continue-watching`,
      { params }
    ).pipe(
      tap(response => {
        this.continueWatchingItems.set(response.data);
        this.continueWatchingLoading.set(false);
      })
    );
  }

  /**
   * Remove an item from continue watching
   * This soft-deletes the progress (marks as manually_removed)
   */
  removeFromContinueWatching(mediaId: string, episodeId?: number): Observable<void> {
    let params = new HttpParams();
    if (episodeId !== undefined) {
      params = params.set('episode_id', episodeId.toString());
    }

    return this.http.delete<void>(
      `${this.baseUrl}/playback/progress/${mediaId}`,
      { params }
    ).pipe(
      tap(() => {
        // Update local cache by removing the item
        const currentItems = this.continueWatchingItems();
        const filteredItems = currentItems.filter(item => {
          if (episodeId !== undefined) {
            return !(item.media_id === mediaId && item.episode_metadata_id === episodeId);
          }
          return item.media_id !== mediaId;
        });
        this.continueWatchingItems.set(filteredItems);
      })
    );
  }

  /**
   * Invalidate the continue watching cache
   * Call this after recording progress to force a refresh
   */
  invalidateContinueWatchingCache(): void {
    // Don't clear the signal immediately to avoid UI flicker
    // The next call to getContinueWatching() will refresh the data
    // Alternatively, you could auto-refresh here:
    // this.getContinueWatching().subscribe();
  }

  /**
   * Refresh continue watching items
   * Convenience method to reload the cache
   */
  refreshContinueWatching(): Observable<ApiResponse<ContinueWatchingItem[]>> {
    return this.getContinueWatching();
  }

  /**
   * Calculate progress percentage
   * Utility function for UI
   */
  calculateProgressPercent(position: number, duration: number): number {
    if (duration <= 0) return 0;
    return Math.min(100, (position / duration) * 100);
  }

  /**
   * Format remaining time
   * Utility function for UI (e.g., "23 min left")
   */
  formatTimeRemaining(position: number, duration: number): string {
    const remaining = duration - position;
    if (remaining <= 0) return '0 min';
    
    const minutes = Math.ceil(remaining / 60);
    if (minutes < 60) {
      return `${minutes} min`;
    }
    
    const hours = Math.floor(minutes / 60);
    const remainingMinutes = minutes % 60;
    if (remainingMinutes === 0) {
      return `${hours}h`;
    }
    return `${hours}h ${remainingMinutes}m`;
  }
}
