import { Component, inject, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateModule } from '@ngx-translate/core';
import { WatchProgressService, ContinueWatchingItem } from '../../core/services/watch-progress.service';
import { LazyLoadDirective } from '../../shared/directives/lazy-load.directive';

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p/w342';

@Component({
  selector: 'app-continue-watching-row',
  standalone: true,
  imports: [LazyLoadDirective, TranslateModule],
  template: `
    <!-- Only show when there are items -->
    @if (items().length > 0) {
      <div class="mb-8">
        <!-- Row header -->
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-bold text-white">{{ 'library.continueWatching' | translate }}</h2>
        </div>

        <!-- Items -->
        <div class="relative">
          <div
            class="flex gap-3 sm:gap-4 overflow-x-auto py-2 px-1 -mx-1 snap-x snap-mandatory scrollbar-thin scrollbar-thumb-slate-600 scrollbar-track-slate-800"
            style="scroll-behavior: smooth;"
          >
            @for (item of items(); track item.id) {
              <div class="flex-shrink-0 w-48 sm:w-56 md:w-64 snap-start">
                <div
                  class="relative rounded-lg overflow-hidden shadow-lg hover:ring-2 hover:ring-blue-500 transition cursor-pointer group bg-slate-800"
                  (click)="onCardClick(item)"
                >
                  <!-- Video thumbnail (16:9 aspect ratio) -->
                  <div class="aspect-video bg-slate-700 relative overflow-hidden">
                    @if (getPosterUrl(item)) {
                      <img
                        appLazyLoad
                        [lazySrc]="getPosterUrl(item)"
                        [lazyAlt]="item.title"
                        class="w-full h-full object-cover"
                      />
                    } @else {
                      <!-- Placeholder icon when no poster -->
                      <div class="w-full h-full flex items-center justify-center">
                        <svg class="w-12 h-12 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                        </svg>
                      </div>
                    }

                    <!-- Hover overlay with play icon -->
                    <div class="absolute inset-0 bg-black/60 flex items-center justify-center opacity-0 group-hover:opacity-100 transition">
                      <svg class="w-16 h-16 text-white" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M8 5v14l11-7z"/>
                      </svg>
                    </div>

                    <!-- Remove button (X) - top-right corner -->
                    <button
                      (click)="onRemoveClick($event, item)"
                      class="absolute top-2 right-2 w-7 h-7 bg-black/70 hover:bg-red-600 text-white rounded-full flex items-center justify-center transition opacity-0 group-hover:opacity-100 z-10"
                      title="Remove from Continue Watching"
                    >
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                      </svg>
                    </button>

                    <!-- Progress bar at the bottom -->
                    <div class="absolute bottom-0 left-0 right-0 h-1 bg-slate-800/90">
                      <div
                        class="h-full bg-blue-500"
                        [style.width.%]="item.progress_percent"
                      ></div>
                    </div>
                  </div>

                  <!-- Title and info -->
                  <div class="p-3">
                    <h3 class="text-white font-medium truncate text-sm" [title]="item.title">
                      {{ item.title }}
                    </h3>
                    <div class="mt-1 text-xs text-slate-400">
                      @if (item.media_type === 'episode' && item.series_name) {
                        <div class="truncate">{{ item.series_name }}</div>
                        <div class="mt-0.5">
                          S{{ formatSeasonEpisode(item.season_number) }}E{{ formatSeasonEpisode(item.episode_number) }}
                          @if (item.episode_name) {
                            <span> • {{ item.episode_name }}</span>
                          }
                        </div>
                      } @else {
                        @if (item.year) {
                          <span>{{ item.year }}</span>
                          <span class="mx-1">•</span>
                        }
                        <span>{{ formatTimeRemaining(item) }} left</span>
                      }
                    </div>
                  </div>
                </div>
              </div>
            }
          </div>
        </div>
      </div>
    }
  `,
  styles: [`
    /* Custom scrollbar styling */
    .scrollbar-thin::-webkit-scrollbar {
      height: 6px;
    }
    .scrollbar-thin::-webkit-scrollbar-track {
      background: rgb(30 41 59);
      border-radius: 3px;
    }
    .scrollbar-thin::-webkit-scrollbar-thumb {
      background: rgb(71 85 105);
      border-radius: 3px;
    }
    .scrollbar-thin::-webkit-scrollbar-thumb:hover {
      background: rgb(100 116 139);
    }
  `]
})
export class ContinueWatchingRowComponent implements OnInit {
  readonly watchProgressService = inject(WatchProgressService);
  private readonly router = inject(Router);

  // Signal from the service
  items = this.watchProgressService.continueWatchingItems;

  ngOnInit(): void {
    // Load continue watching items on component init
    this.watchProgressService.getContinueWatching().subscribe({
      next: (response) => {
        console.log('Continue watching items loaded:', response.data?.length, response.data);
      },
      error: (err) => {
        console.error('Failed to load continue watching items:', err);
      }
    });
  }

  getPosterUrl(item: ContinueWatchingItem): string {
    // Prefer backdrop for landscape format, fallback to poster
    const path = item.backdrop_path || item.poster_path;
    if (path) {
      return `${TMDB_IMAGE_BASE}${path}`;
    }
    return '';
  }

  formatSeasonEpisode(num?: number): string {
    if (num === undefined) return '??';
    return num.toString().padStart(2, '0');
  }

  formatTimeRemaining(item: ContinueWatchingItem): string {
    const remaining = item.duration_seconds - item.position_seconds;
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

  onCardClick(item: ContinueWatchingItem): void {
    // Navigate to player or detail page
    if (item.media_type === 'movie' && item.media_id) {
      // For movies, navigate to movie detail page (which will resume playback)
      this.router.navigate(['/movie', item.media_id]);
    } else if (item.media_type === 'episode' && item.series_metadata_id) {
      // For episodes, navigate to series detail page with season
      this.router.navigate(['/series', item.series_metadata_id], {
        queryParams: { season: item.season_number }
      });
    }
  }

  onRemoveClick(event: Event, item: ContinueWatchingItem): void {
    // Prevent card click event from firing
    event.stopPropagation();

    // Remove from continue watching
    const mediaId = item.media_id || '';
    const episodeId = item.episode_metadata_id;

    this.watchProgressService.removeFromContinueWatching(mediaId, episodeId).subscribe({
      error: (err) => {
        console.error('Failed to remove from continue watching:', err);
      }
    });
  }
}
