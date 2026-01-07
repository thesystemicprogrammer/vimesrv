import { Injectable, inject, signal } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, tap } from 'rxjs';

// API Response wrapper
export interface ApiResponse<T> {
  data: T;
}

// Recommendation item from API
export interface RecommendationItem {
  item_id: number;
  item_type: 'movie' | 'series';
  media_id?: string;
  title: string;
  year?: string;
  poster_path?: string;
  backdrop_path?: string;
  vote_average: number;
  score: number;
}

// Similar item response from API
export interface SimilarItem {
  metadata_id: number;
  similarity_score: number;
  rank: number;
}

// Model status response from API
export interface ModelStatus {
  model_type: string;
  total_items: number;
  feature_count: number;
  last_built_at: string;
  build_duration_ms: number;
}

// Rebuild response from API
export interface RebuildResponse {
  movie_model_built: boolean;
  series_model_built: boolean;
  movie_items: number;
  series_items: number;
  movie_duration_ms: number;
  series_duration_ms: number;
}

@Injectable({
  providedIn: 'root'
})
export class RecommendationsService {
  private http = inject(HttpClient);
  private baseUrl = '/api/v1';

  // Signal to track user's recommendations (cached)
  recommendationItems = signal<RecommendationItem[]>([]);
  recommendationsLoading = signal<boolean>(false);
  recommendationsError = signal<string | null>(null);

  /**
   * Get personalized recommendations for the current user
   * Results are cached in a signal for UI reactivity
   */
  getRecommendations(limit: number = 20, type?: 'movie' | 'series'): Observable<ApiResponse<RecommendationItem[]>> {
    this.recommendationsLoading.set(true);
    this.recommendationsError.set(null);
    
    let params = new HttpParams().set('limit', limit.toString());
    if (type) {
      params = params.set('type', type);
    }

    return this.http.get<ApiResponse<RecommendationItem[]>>(
      `${this.baseUrl}/recommendations`,
      { params }
    ).pipe(
      tap({
        next: (response) => {
          this.recommendationItems.set(response.data || []);
          this.recommendationsLoading.set(false);
        },
        error: (error) => {
          console.error('Failed to fetch recommendations:', error);
          this.recommendationsError.set('Failed to load recommendations');
          this.recommendationsLoading.set(false);
          this.recommendationItems.set([]);
        }
      })
    );
  }

  /**
   * Get similar movies to a given movie
   */
  getSimilarMovies(movieId: number, limit: number = 10): Observable<ApiResponse<SimilarItem[]>> {
    const params = new HttpParams().set('limit', limit.toString());
    
    return this.http.get<ApiResponse<SimilarItem[]>>(
      `${this.baseUrl}/movies/${movieId}/similar`,
      { params }
    );
  }

  /**
   * Get similar series to a given series
   */
  getSimilarSeries(seriesId: number, limit: number = 10): Observable<ApiResponse<SimilarItem[]>> {
    const params = new HttpParams().set('limit', limit.toString());
    
    return this.http.get<ApiResponse<SimilarItem[]>>(
      `${this.baseUrl}/series/${seriesId}/similar`,
      { params }
    );
  }

  /**
   * Trigger a rebuild of recommendation models (admin only)
   */
  rebuildModels(type?: 'movie' | 'series'): Observable<ApiResponse<RebuildResponse>> {
    let params = new HttpParams();
    if (type) {
      params = params.set('type', type);
    }

    return this.http.post<ApiResponse<RebuildResponse>>(
      `${this.baseUrl}/admin/recommendations/rebuild`,
      null,
      { params }
    );
  }

  /**
   * Get the status of recommendation models (admin only)
   */
  getModelStatus(): Observable<ApiResponse<ModelStatus[]>> {
    return this.http.get<ApiResponse<ModelStatus[]>>(
      `${this.baseUrl}/admin/recommendations/status`
    );
  }

  /**
   * Refresh recommendations list
   * Convenience method to reload the cache
   */
  refreshRecommendations(): Observable<ApiResponse<RecommendationItem[]>> {
    return this.getRecommendations();
  }

  /**
   * Check if recommendations are available
   */
  hasRecommendations(): boolean {
    return this.recommendationItems().length > 0;
  }
}
