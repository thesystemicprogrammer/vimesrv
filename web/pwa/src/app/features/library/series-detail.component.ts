import { Component, inject, OnInit, OnDestroy, signal, computed } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { toObservable } from '@angular/core/rxjs-interop';
import { Subscription, skip } from 'rxjs';
import {
  ApiService,
  SeriesDetail,
  SeasonSummary,
  EpisodeSummary
} from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p';

@Component({
  selector: 'app-series-detail',
  standalone: true,
  template: `
    @if (loading()) {
      <div class="flex justify-center items-center h-screen">
        <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    } @else if (error()) {
      <div class="container mx-auto px-4 py-8">
        <div class="bg-red-500/10 border border-red-500 text-red-400 px-4 py-3 rounded">
          {{ error() }}
        </div>
        <button
          (click)="goBack()"
          class="mt-4 text-blue-400 hover:text-blue-300 transition"
        >
          &larr; Back to Library
        </button>
      </div>
    } @else if (series()) {
      <!-- Backdrop hero section -->
      <div class="relative min-h-[40vh] md:min-h-[50vh]">
        <!-- Backdrop image -->
        @if (backdropUrl()) {
          <div
            class="absolute inset-0 bg-cover bg-center"
            [style.background-image]="'url(' + backdropUrl() + ')'"
          ></div>
        }
        <!-- Gradient overlay -->
        <div class="absolute inset-0 bg-gradient-to-t from-slate-900 via-slate-900/80 to-slate-900/30"></div>

        <!-- Back button -->
        <div class="absolute top-4 left-4 z-10">
          <button
            (click)="goBack()"
            class="flex items-center gap-2 px-3 py-2 bg-black/50 hover:bg-black/70 text-white rounded-lg transition"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
            </svg>
            <span class="text-sm">Back</span>
          </button>
        </div>

        <!-- Content overlay -->
        <div class="absolute bottom-0 left-0 right-0 p-4 md:p-8">
          <div class="container mx-auto flex gap-6 items-end">
            <!-- Poster -->
            <div class="hidden md:block flex-shrink-0 w-40 lg:w-48">
              <div class="aspect-[2/3] rounded-lg overflow-hidden shadow-2xl bg-slate-800">
                @if (posterUrl()) {
                  <img
                    [src]="posterUrl()"
                    [alt]="series()!.name"
                    class="w-full h-full object-cover"
                  />
                } @else {
                  <div class="w-full h-full flex items-center justify-center">
                    <svg class="w-16 h-16 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                    </svg>
                  </div>
                }
              </div>
            </div>

            <!-- Info -->
            <div class="flex-1 min-w-0">
              <h1 class="text-2xl md:text-3xl lg:text-4xl font-bold text-white mb-2">
                {{ series()!.name }}
              </h1>

              <!-- Meta info row -->
              <div class="flex flex-wrap items-center gap-3 text-sm text-slate-300 mb-3">
                @if (series()!.year) {
                  <span>{{ series()!.year }}</span>
                }
                <span class="text-slate-500">•</span>
                <span>{{ series()!.number_of_seasons }} seasons</span>
                <span class="text-slate-500">•</span>
                <span>{{ series()!.available_episodes }} / {{ series()!.number_of_episodes }} episodes</span>
                @if (series()!.vote_average > 0) {
                  <span class="text-slate-500">•</span>
                  <div class="flex items-center gap-1">
                    <svg class="w-4 h-4 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                      <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                    </svg>
                    <span class="font-medium">{{ series()!.vote_average.toFixed(1) }}</span>
                  </div>
                }
              </div>

              <!-- Genres -->
              @if (series()!.genres) {
                <div class="flex flex-wrap gap-2">
                  @for (genre of parseGenres(series()!.genres); track genre) {
                    <span class="px-3 py-1 bg-slate-700/80 text-slate-300 rounded-full text-sm">
                      {{ genre }}
                    </span>
                  }
                </div>
              }
            </div>
          </div>
        </div>
      </div>

      <!-- Overview section -->
      @if (series()!.overview) {
        <div class="container mx-auto px-4 py-6">
          <p class="text-slate-300 leading-relaxed max-w-4xl">
            {{ series()!.overview }}
          </p>
        </div>
      }

      <!-- Season selector and episodes -->
      <div class="container mx-auto px-4 py-6">
        <!-- Season dropdown -->
        @if (seasons().length > 0) {
          <div class="flex items-center gap-4 mb-6">
            <label class="text-white font-medium">Season:</label>
            <select
              [value]="selectedSeasonNumber()"
              (change)="onSeasonChange($event)"
              class="bg-slate-800 text-white border border-slate-600 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              @for (season of seasons(); track season.season_metadata_id) {
                <option [value]="season.season_number">
                  {{ season.name || 'Season ' + season.season_number }}
                  ({{ season.episode_count }} episodes)
                </option>
              }
            </select>
          </div>

          <!-- Episode list -->
          @if (selectedSeason()) {
            <div class="space-y-3">
              @for (episode of selectedSeason()!.episodes || []; track episode.episode_metadata_id) {
                <div
                  class="flex items-center gap-4 p-4 bg-slate-800 rounded-lg hover:bg-slate-700 transition"
                  [class.opacity-50]="!episode.media_id"
                >
                  <!-- Episode still image -->
                  <div class="flex-shrink-0 w-32 md:w-40">
                    <div class="aspect-video rounded overflow-hidden bg-slate-700">
                      @if (episode.still_path) {
                        <img
                          [src]="getStillUrl(episode.still_path)"
                          [alt]="episode.name"
                          class="w-full h-full object-cover"
                          loading="lazy"
                        />
                      } @else {
                        <div class="w-full h-full flex items-center justify-center">
                          <svg class="w-8 h-8 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                          </svg>
                        </div>
                      }
                    </div>
                  </div>

                  <!-- Episode info -->
                  <div class="flex-1 min-w-0">
                    <div class="flex items-start justify-between gap-4">
                      <div>
                        <h3 class="text-white font-medium">
                          {{ episode.episode_number }}. {{ episode.name }}
                        </h3>
                        <div class="flex items-center gap-2 mt-1 text-xs text-slate-400">
                          @if (episode.air_date) {
                            <span>{{ formatDate(episode.air_date) }}</span>
                          }
                          @if (episode.duration) {
                            <span class="text-slate-500">•</span>
                            <span>{{ formatDuration(episode.duration) }}</span>
                          }
                          @if (episode.vote_average > 0) {
                            <span class="text-slate-500">•</span>
                            <div class="flex items-center gap-1">
                              <svg class="w-3 h-3 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                                <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                              </svg>
                              <span>{{ episode.vote_average.toFixed(1) }}</span>
                            </div>
                          }
                        </div>
                        @if (episode.overview) {
                          <p class="mt-2 text-sm text-slate-400 line-clamp-2">
                            {{ episode.overview }}
                          </p>
                        }
                      </div>

                      <!-- Play button -->
                      @if (episode.media_id && episode.transcode_status === 'completed') {
                        <button
                          (click)="playEpisode(episode)"
                          class="flex-shrink-0 flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition"
                        >
                          <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                            <path d="M8 5v14l11-7z"/>
                          </svg>
                          <span class="hidden sm:inline">Play</span>
                        </button>
                      } @else if (episode.media_id && episode.transcode_status === 'pending') {
                        <div class="flex-shrink-0 px-3 py-2 text-yellow-500 text-sm">
                          <svg class="w-5 h-5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                          </svg>
                        </div>
                      } @else if (!episode.media_id) {
                        <div class="flex-shrink-0 px-3 py-2 text-slate-500 text-sm">
                          Not available
                        </div>
                      }
                    </div>
                  </div>
                </div>
              }

              @if (!selectedSeason()!.episodes || selectedSeason()!.episodes!.length === 0) {
                <div class="text-center text-slate-400 py-8">
                  No episodes found for this season
                </div>
              }
            </div>
          }
        } @else {
          <div class="text-center text-slate-400 py-8">
            No seasons available
          </div>
        }
      </div>
    }
  `,
  styles: [`
    .line-clamp-2 {
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
    }
  `]
})
export class SeriesDetailComponent implements OnInit, OnDestroy {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);

  private seriesId: number | null = null;
  private languageSubscription: Subscription | null = null;

  series = signal<SeriesDetail | null>(null);
  loading = signal(true);
  error = signal<string | null>(null);
  selectedSeasonNumber = signal<number>(1);

  posterUrl = signal<string | null>(null);
  backdropUrl = signal<string | null>(null);

  seasons = computed(() => {
    const s = this.series();
    return s?.seasons || [];
  });

  selectedSeason = computed(() => {
    const seasonNum = this.selectedSeasonNumber();
    return this.seasons().find(s => s.season_number === seasonNum) || null;
  });

  constructor() {
    // Re-fetch series details when language changes (skip initial emission)
    this.languageSubscription = toObservable(this.auth.language)
      .pipe(skip(1))
      .subscribe(() => {
        if (this.seriesId) {
          // Preserve current season selection when re-fetching
          const currentSeason = this.selectedSeasonNumber();
          this.loadSeries(this.seriesId, currentSeason);
        }
      });
  }

  ngOnDestroy(): void {
    this.languageSubscription?.unsubscribe();
  }

  ngOnInit(): void {
    const seriesIdParam = this.route.snapshot.paramMap.get('id');
    if (!seriesIdParam || isNaN(Number(seriesIdParam))) {
      this.error.set('Invalid series ID');
      this.loading.set(false);
      return;
    }
    this.seriesId = Number(seriesIdParam);

    // Read optional season query param (e.g., /series/1?season=2)
    const seasonParam = this.route.snapshot.queryParamMap.get('season');
    const requestedSeason = seasonParam && !isNaN(Number(seasonParam)) ? Number(seasonParam) : null;

    this.loadSeries(this.seriesId, requestedSeason);
  }

  loadSeries(seriesId: number, requestedSeason: number | null = null): void {
    this.loading.set(true);
    this.error.set(null);

    this.api.getSeriesDetail(seriesId).subscribe({
      next: (response) => {
        const seriesData = response.data;
        this.series.set(seriesData);

        if (seriesData.poster_path) {
          this.posterUrl.set(`${TMDB_IMAGE_BASE}/w500${seriesData.poster_path}`);
        }
        if (seriesData.backdrop_path) {
          this.backdropUrl.set(`${TMDB_IMAGE_BASE}/w1280${seriesData.backdrop_path}`);
        }

        // Select requested season from query param, or fall back to first available
        if (seriesData.seasons && seriesData.seasons.length > 0) {
          const seasonExists = requestedSeason !== null &&
            seriesData.seasons.some(s => s.season_number === requestedSeason);
          
          if (seasonExists) {
            this.selectedSeasonNumber.set(requestedSeason!);
          } else {
            this.selectedSeasonNumber.set(seriesData.seasons[0].season_number);
          }
        }

        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Failed to load series');
        this.loading.set(false);
      }
    });
  }

  onSeasonChange(event: Event): void {
    const target = event.target as HTMLSelectElement;
    this.selectedSeasonNumber.set(Number(target.value));
  }

  playEpisode(episode: EpisodeSummary): void {
    if (episode.media_id) {
      this.router.navigate(['/play', episode.media_id]);
    }
  }

  goBack(): void {
    this.router.navigate(['/library']);
  }

  getStillUrl(stillPath: string): string {
    return `${TMDB_IMAGE_BASE}/w300${stillPath}`;
  }

  formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }

  formatDate(dateStr: string): string {
    try {
      const date = new Date(dateStr);
      return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
    } catch {
      return dateStr;
    }
  }

  parseGenres(genres: string | undefined): string[] {
    if (!genres) return [];
    return genres.split(',').map(g => g.trim()).filter(g => g.length > 0);
  }
}
