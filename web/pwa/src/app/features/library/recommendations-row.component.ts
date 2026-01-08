import { Component, inject, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateModule } from '@ngx-translate/core';
import { RecommendationsService, RecommendationItem } from '../../core/services/recommendations.service';
import { LazyLoadDirective } from '../../shared/directives/lazy-load.directive';
import { FavoriteButtonComponent } from '../../shared/components/favorite-button.component';

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p/w342';

@Component({
  selector: 'app-recommendations-row',
  standalone: true,
  imports: [LazyLoadDirective, TranslateModule, FavoriteButtonComponent],
  template: `
    <!-- Only show when there are items (hide on empty or error) -->
    @if (items().length > 0 && !recommendationsService.recommendationsError()) {
      <div class="mb-8">
        <!-- Row header -->
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-bold text-white">{{ 'library.recommendations' | translate }}</h2>
        </div>

        <!-- Items -->
        <div class="relative">
          <div
            class="flex gap-4 overflow-x-auto py-2 px-1 -mx-1 snap-x snap-mandatory scrollbar-thin scrollbar-thumb-slate-600 scrollbar-track-slate-800"
            style="scroll-behavior: smooth;"
          >
            @for (item of items(); track trackByItem(item)) {
              <div class="flex-shrink-0 w-40 snap-start">
                <div
                  class="relative rounded-lg overflow-hidden shadow-lg hover:ring-2 hover:ring-blue-500 transition cursor-pointer group bg-slate-800"
                  (click)="onCardClick(item)"
                >
                  <!-- Poster Image (2:3 aspect ratio) -->
                  <div class="aspect-[2/3] bg-slate-700 relative overflow-hidden">
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
                    <div class="absolute inset-0 bg-black/50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition">
                      <svg class="w-14 h-14 text-white" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M8 5v14l11-7z"/>
                      </svg>
                    </div>

                    <!-- Favorite heart button - always visible, top-right -->
                    <div class="absolute top-2 right-2 z-10">
                      <app-favorite-button
                        [mediaType]="item.item_type"
                        [metadataId]="item.item_id"
                        [iconClass]="'w-6 h-6'"
                        (click)="$event.stopPropagation()"
                      />
                    </div>

                    <!-- Rating badge -->
                    @if (item.vote_average && item.vote_average > 0) {
                      <div class="absolute top-2 left-2 flex items-center gap-1 bg-black/70 px-2 py-1 rounded text-xs">
                        <svg class="w-3 h-3 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                          <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                        </svg>
                        <span class="text-white font-medium">{{ item.vote_average.toFixed(1) }}</span>
                      </div>
                    }

                    <!-- Type badge (movie/series indicator) -->
                    <div class="absolute bottom-2 left-2 px-2 py-0.5 rounded text-xs font-medium"
                         [class]="item.item_type === 'movie' ? 'bg-blue-600 text-white' : 'bg-purple-600 text-white'">
                      {{ item.item_type === 'movie' ? 'Movie' : 'Series' }}
                    </div>
                  </div>

                  <!-- Title and info -->
                  <div class="p-3">
                    <h3 class="text-white font-medium truncate text-sm" [title]="item.title">
                      {{ item.title }}
                    </h3>
                    <div class="mt-1 text-xs text-slate-400">
                      @if (item.year) {
                        <div>{{ item.year }}</div>
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
export class RecommendationsRowComponent implements OnInit {
  readonly recommendationsService = inject(RecommendationsService);
  private readonly router = inject(Router);

  // Signal from the service
  items = this.recommendationsService.recommendationItems;

  ngOnInit(): void {
    // Load recommendations on component init
    this.recommendationsService.getRecommendations().subscribe();
  }

  getPosterUrl(item: RecommendationItem): string {
    const path = item.poster_path;
    if (path) {
      return `${TMDB_IMAGE_BASE}${path}`;
    }
    return '';
  }

  trackByItem(item: RecommendationItem): string {
    return `${item.item_type}-${item.item_id}`;
  }

  onCardClick(item: RecommendationItem): void {
    // Navigate to detail page
    if (item.item_type === 'movie' && item.media_id) {
      this.router.navigate(['/movie', item.media_id]);
    } else if (item.item_type === 'series') {
      this.router.navigate(['/series', item.item_id]);
    }
  }
}
