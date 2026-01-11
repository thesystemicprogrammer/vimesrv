import { Component, inject, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateModule } from '@ngx-translate/core';
import {
  WatchProgressService,
  ContinueWatchingItem,
} from '../../core/services/watch-progress.service';
import { LazyLoadDirective } from '../../shared/directives/lazy-load.directive';

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p/w342';

@Component({
  selector: 'app-continue-watching-row',
  standalone: true,
  imports: [LazyLoadDirective, TranslateModule],
  templateUrl: './continue-watching-row.component.html',
  styles: [
    `
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
    `,
  ],
})
export class ContinueWatchingRowComponent implements OnInit {
  readonly watchProgressService = inject(WatchProgressService);
  private readonly router = inject(Router);

  // Signal from the service
  items = this.watchProgressService.continueWatchingItems;

  ngOnInit(): void {
    // Load continue watching items on component init
    this.watchProgressService.getContinueWatching().subscribe();
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
    if (item.media_id) {
      this.router.navigate(['/play', item.media_id]);
    }
  }

  onRemoveClick(event: Event, item: ContinueWatchingItem): void {
    // Prevent card click event from firing
    event.stopPropagation();

    // Remove from continue watching
    const mediaId = item.media_id || '';
    const episodeId = item.episode_metadata_id;

    this.watchProgressService
      .removeFromContinueWatching(mediaId, episodeId)
      .subscribe({
        error: (err) => {
          console.error('Failed to remove from continue watching:', err);
        },
      });
  }
}
