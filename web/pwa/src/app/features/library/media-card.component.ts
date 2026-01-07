import { Component, Input, Output, EventEmitter, inject } from '@angular/core';
import { MovieSummary, SeriesSummary, RecentlyAddedItem } from '../../core/services/api.service';
import { LazyLoadDirective } from '../../shared/directives/lazy-load.directive';
import { FavoriteButtonComponent } from '../../shared/components/favorite-button.component';
import { FavoritesService } from '../../core/services/favorites.service';

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p/w342';

export type CardType = 'movie' | 'series' | 'recent';

@Component({
  selector: 'app-media-card',
  standalone: true,
  imports: [LazyLoadDirective, FavoriteButtonComponent],
  template: `
    <div
      class="relative rounded-lg overflow-hidden shadow-lg hover:ring-2 hover:ring-blue-500 transition cursor-pointer group bg-slate-800"
      (click)="onClick()"
    >
      <!-- Poster Image (2:3 aspect ratio) -->
      <div class="aspect-[2/3] bg-slate-700 relative overflow-hidden">
        @if (posterUrl) {
          <img
            appLazyLoad
            [lazySrc]="posterUrl"
            [lazyAlt]="title"
            class="w-full h-full object-cover"
          />
        } @else {
          <!-- Placeholder icon when no poster -->
          <div class="w-full h-full flex items-center justify-center">
            <svg class="w-16 h-16 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/>
            </svg>
          </div>
        }

        <!-- Hover overlay with play icon -->
        <div class="absolute inset-0 bg-black/50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition">
          <svg class="w-14 h-14 text-white" fill="currentColor" viewBox="0 0 24 24">
            <path d="M8 5v14l11-7z"/>
          </svg>
        </div>

        <!-- Rating badge -->
        @if (rating > 0) {
          <div class="absolute top-2 left-2 flex items-center gap-1 bg-black/70 px-2 py-1 rounded text-xs">
            <svg class="w-3 h-3 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
              <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
            </svg>
            <span class="text-white font-medium">{{ rating.toFixed(1) }}</span>
          </div>
        }

        <!-- Favorite button - always visible when favorited, otherwise show on hover -->
        @if (getMetadataId() > 0 && getFavoriteMediaType()) {
          <div class="absolute top-2 right-2 z-10 transition"
               [class.opacity-100]="isFavorited()"
               [class.opacity-0]="!isFavorited()"
               [class.group-hover:opacity-100]="true"
               (click)="onFavoriteClick($event)">
            <app-favorite-button
              [mediaType]="getFavoriteMediaType()!"
              [metadataId]="getMetadataId()"
              [iconClass]="'w-6 h-6'"
            />
          </div>
        }

        <!-- Series/Season episode count badge -->
        @if ((cardType === 'series' || cardType === 'recent') && episodeCount) {
          <div class="absolute bottom-2 right-2 bg-blue-600/90 px-2 py-1 rounded text-xs text-white font-medium">
            {{ episodeCount }} {{ episodeCount === 1 ? 'episode' : 'episodes' }}
          </div>
        }

        <!-- Transcode status indicator -->
        @if (transcodeStatus === 'pending') {
          <div class="absolute top-2 right-2 bg-yellow-500/90 px-2 py-1 rounded text-xs text-black font-medium">
            Processing
          </div>
        }
      </div>

      <!-- Title and info -->
      <div class="p-3">
        <h3 class="text-white font-medium truncate text-sm" [title]="title">
          {{ title }}
        </h3>
        <div class="mt-1 flex items-center gap-2 text-xs text-slate-400">
          @if (year) {
            <span>{{ year }}</span>
          }
          @if (year && duration) {
            <span>•</span>
          }
          @if (duration) {
            <span>{{ formatDuration(duration) }}</span>
          }
        </div>
      </div>
    </div>
  `
})
export class MediaCardComponent {
  private readonly favoritesService = inject(FavoritesService);
  
  @Input({ required: true }) cardType: CardType = 'movie';
  @Input() movie?: MovieSummary;
  @Input() series?: SeriesSummary;
  @Input() recentItem?: RecentlyAddedItem;
  @Output() cardClick = new EventEmitter<void>();

  get title(): string {
    if (this.cardType === 'movie' && this.movie) {
      return this.movie.title;
    }
    if (this.cardType === 'series' && this.series) {
      return this.series.name;
    }
    if (this.cardType === 'recent' && this.recentItem) {
      return this.recentItem.title;
    }
    return '';
  }

  get year(): string | undefined {
    if (this.cardType === 'movie' && this.movie) {
      return this.movie.year;
    }
    if (this.cardType === 'series' && this.series) {
      return this.series.year;
    }
    if (this.cardType === 'recent' && this.recentItem) {
      return this.recentItem.year;
    }
    return undefined;
  }

  get duration(): number {
    if (this.cardType === 'movie' && this.movie) {
      return this.movie.duration;
    }
    // Recent items and series don't show duration
    return 0;
  }

  get posterUrl(): string | null {
    let posterPath: string | undefined;

    if (this.cardType === 'movie' && this.movie) {
      posterPath = this.movie.poster_path;
    } else if (this.cardType === 'series' && this.series) {
      posterPath = this.series.poster_path;
    } else if (this.cardType === 'recent' && this.recentItem) {
      posterPath = this.recentItem.poster_path;
    }

    if (posterPath) {
      return `${TMDB_IMAGE_BASE}${posterPath}`;
    }
    return null;
  }

  get rating(): number {
    if (this.cardType === 'movie' && this.movie) {
      return this.movie.vote_average;
    }
    if (this.cardType === 'series' && this.series) {
      return this.series.vote_average;
    }
    if (this.cardType === 'recent' && this.recentItem) {
      return this.recentItem.vote_average;
    }
    return 0;
  }

  get episodeCount(): number | null {
    if (this.cardType === 'series' && this.series) {
      return this.series.available_episodes;
    }
    if (this.cardType === 'recent' && this.recentItem && this.recentItem.type === 'season') {
      return this.recentItem.episode_count || null;
    }
    return null;
  }

  get transcodeStatus(): string {
    if (this.cardType === 'movie' && this.movie) {
      return this.movie.transcode_status;
    }
    if (this.cardType === 'recent' && this.recentItem && this.recentItem.type === 'movie') {
      return this.recentItem.transcode_status || 'none';
    }
    return 'none';
  }

  onClick(): void {
    this.cardClick.emit();
  }

  onFavoriteClick(event: Event): void {
    // Prevent the card click event from firing
    event.stopPropagation();
  }

  getMetadataId(): number {
    if (this.cardType === 'movie' && this.movie) {
      return this.movie.movie_metadata_id || 0;
    }
    if (this.cardType === 'series' && this.series) {
      return this.series.series_metadata_id || 0;
    }
    if (this.cardType === 'recent' && this.recentItem) {
      // RecentlyAddedItem has both movie_metadata_id and series_metadata_id
      // depending on the type
      if (this.recentItem.type === 'movie') {
        return this.recentItem.movie_metadata_id || 0;
      } else if (this.recentItem.type === 'season') {
        return this.recentItem.series_metadata_id || 0;
      }
    }
    return 0;
  }

  getFavoriteMediaType(): 'movie' | 'series' | null {
    if (this.cardType === 'movie') {
      return 'movie';
    }
    if (this.cardType === 'series') {
      return 'series';
    }
    if (this.cardType === 'recent' && this.recentItem) {
      if (this.recentItem.type === 'movie') {
        return 'movie';
      } else if (this.recentItem.type === 'season') {
        return 'series';
      }
    }
    return null;
  }

  formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);

    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }

  isFavorited(): boolean {
    const mediaType = this.getFavoriteMediaType();
    const metadataId = this.getMetadataId();
    if (mediaType && metadataId > 0) {
      return this.favoritesService.isFavorited(mediaType, metadataId);
    }
    return false;
  }
}
