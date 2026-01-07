import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, tap } from 'rxjs';
import { AuthService } from './auth.service';

// API Response wrapper
export interface ApiResponse<T> {
  data: T;
}

// Favorite Types
export interface Favorite {
  id: string;
  user_id: string;
  media_type: 'movie' | 'series';
  movie_metadata_id?: number;
  series_metadata_id?: number;
  added_at: string;
}

export interface FavoriteItem {
  // Favorite info
  id: string;
  user_id: string;
  media_type: 'movie' | 'series';
  metadata_id: number;  // Unified metadata ID from API
  movie_metadata_id?: number;  // Legacy/alternative field
  series_metadata_id?: number; // Legacy/alternative field
  added_at: string;
  
  // Enriched metadata
  title: string;
  poster_path?: string;
  backdrop_path?: string;
  year?: number;
  rating?: number;
  genres?: string;
  media_id?: string; // For movies: the media_id to navigate to movie detail
}

export interface ToggleFavoriteRequest {
  media_type: 'movie' | 'series';
  metadata_id: number;
}

export interface ToggleFavoriteResponse {
  favorited: boolean; // true = added, false = removed
}

@Injectable({
  providedIn: 'root'
})
export class FavoritesService {
  private http = inject(HttpClient);
  private auth = inject(AuthService);
  private baseUrl = '/api/v1';

  // Signal to track user's favorites (cached)
  favoriteItems = signal<FavoriteItem[]>([]);
  favoritesLoading = signal<boolean>(false);

  // Map to track favorited status by media type and ID for quick lookup
  private favoritedMap = signal<Map<string, boolean>>(new Map());

  /**
   * Toggle favorite status for a movie or series
   * @returns Observable with favorited status (true = now favorited, false = now unfavorited)
   */
  toggleFavorite(mediaType: 'movie' | 'series', metadataId: number): Observable<ApiResponse<ToggleFavoriteResponse>> {
    const request: ToggleFavoriteRequest = {
      media_type: mediaType,
      metadata_id: metadataId
    };

    return this.http.post<ApiResponse<ToggleFavoriteResponse>>(
      `${this.baseUrl}/favorites`,
      request
    ).pipe(
      tap(response => {
        const key = this.getFavoriteKey(mediaType, metadataId);
        const isFavorited = response.data.favorited;
        
        // Update local map
        const map = this.favoritedMap();
        if (isFavorited) {
          map.set(key, true);
        } else {
          map.delete(key);
        }
        this.favoritedMap.set(new Map(map));

        // Invalidate favorites cache
        this.invalidateFavoritesCache();
      })
    );
  }

  /**
   * Get all user favorites
   * Results are cached in a signal for UI reactivity
   */
  getFavorites(limit: number = 50): Observable<ApiResponse<FavoriteItem[]>> {
    this.favoritesLoading.set(true);
    
    const params = new HttpParams().set('limit', limit.toString());

    return this.http.get<ApiResponse<FavoriteItem[]>>(
      `${this.baseUrl}/favorites`,
      { params }
    ).pipe(
      tap(response => {
        this.favoriteItems.set(response.data);
        this.favoritesLoading.set(false);
        
        // Update favoritedMap for quick lookups
        const map = new Map<string, boolean>();
        response.data.forEach(item => {
          // Use unified metadata_id from API, fallback to type-specific fields
          const metadataId = item.metadata_id 
            || (item.media_type === 'movie' ? item.movie_metadata_id : item.series_metadata_id);
          if (metadataId) {
            const key = this.getFavoriteKey(item.media_type, metadataId);
            map.set(key, true);
          }
        });
        this.favoritedMap.set(map);
      })
    );
  }

  /**
   * Check if an item is favorited (client-side check using cached map)
   * For most accurate results, ensure getFavorites() has been called first
   */
  isFavorited(mediaType: 'movie' | 'series', metadataId: number): boolean {
    const key = this.getFavoriteKey(mediaType, metadataId);
    return this.favoritedMap().has(key);
  }

  /**
   * Get favorite signal for a specific item (reactive)
   * Returns a computed signal that updates when favorites change
   */
  getFavoriteSignal(mediaType: 'movie' | 'series', metadataId: number) {
    return signal(this.isFavorited(mediaType, metadataId));
  }

  /**
   * Invalidate the favorites cache
   * Call this after toggling a favorite to force a refresh
   */
  invalidateFavoritesCache(): void {
    // Refresh the favorites list to update the UI
    this.getFavorites().subscribe();
  }

  /**
   * Refresh favorites list
   * Convenience method to reload the cache
   */
  refreshFavorites(): Observable<ApiResponse<FavoriteItem[]>> {
    return this.getFavorites();
  }

  /**
   * Generate a unique key for favorite lookups
   * Format: "movie:123" or "series:456"
   */
  private getFavoriteKey(mediaType: 'movie' | 'series', metadataId: number): string {
    return `${mediaType}:${metadataId}`;
  }

  /**
   * Parse genres string to array
   * Utility function for UI
   */
  parseGenres(genresString?: string): string[] {
    if (!genresString) return [];
    
    // Try parsing as JSON first
    try {
      return JSON.parse(genresString);
    } catch {
      // Fallback to comma-separated
      return genresString.split(',').map(g => g.trim()).filter(g => g);
    }
  }

  /**
   * Get first N genres
   * Utility function for UI (e.g., show only 3 genres)
   */
  getTopGenres(genresString?: string, count: number = 3): string[] {
    const genres = this.parseGenres(genresString);
    return genres.slice(0, count);
  }
}
