import { Component, Input, Output, EventEmitter } from '@angular/core';
import { MovieSummary, SeriesSummary, RecentlyAddedItem } from '../../core/services/api.service';
import { MediaCardComponent, CardType } from './media-card.component';

@Component({
  selector: 'app-media-row',
  standalone: true,
  imports: [MediaCardComponent],
  template: `
    <div class="mb-8">
      <!-- Row header -->
      <div class="flex justify-between items-center mb-4">
        <h2 class="text-xl font-bold text-white">{{ title }}</h2>
        @if (showSeeAll) {
          <button
            (click)="onSeeAll()"
            class="text-blue-400 hover:text-blue-300 text-sm font-medium transition"
          >
            See all
          </button>
        }
      </div>

      <!-- Horizontal scroll container -->
      <div class="relative">
        <div
          class="flex gap-3 sm:gap-4 overflow-x-auto py-2 px-1 -mx-1 snap-x snap-mandatory scrollbar-thin scrollbar-thumb-slate-600 scrollbar-track-slate-800"
          style="scroll-behavior: smooth;"
        >
          <!-- Regular items (movies/series) -->
          @if (cardType !== 'recent') {
            @for (item of items; track trackItem(item)) {
              <div class="flex-shrink-0 w-32 sm:w-36 md:w-40 snap-start">
                <app-media-card
                  [cardType]="cardType"
                  [movie]="cardType === 'movie' ? asMovie(item) : undefined"
                  [series]="cardType === 'series' ? asSeries(item) : undefined"
                  (cardClick)="onItemClick(item)"
                />
              </div>
            }
          }

          <!-- Recently added items -->
          @if (cardType === 'recent') {
            @for (item of recentItems; track trackRecentItem(item)) {
              <div class="flex-shrink-0 w-32 sm:w-36 md:w-40 snap-start">
                <app-media-card
                  [cardType]="'recent'"
                  [recentItem]="item"
                  (cardClick)="onRecentItemClick(item)"
                />
              </div>
            }
          }

          @if (items.length === 0 && recentItems.length === 0) {
            <div class="text-slate-400 text-sm py-8">
              No items to display
            </div>
          }
        </div>
      </div>
    </div>
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
export class MediaRowComponent {
  @Input({ required: true }) title!: string;
  @Input({ required: true }) cardType: CardType = 'movie';
  @Input() items: (MovieSummary | SeriesSummary)[] = [];
  @Input() recentItems: RecentlyAddedItem[] = [];
  @Input() showSeeAll = false;
  @Output() itemClick = new EventEmitter<MovieSummary | SeriesSummary>();
  @Output() recentItemClick = new EventEmitter<RecentlyAddedItem>();
  @Output() seeAllClick = new EventEmitter<void>();

  trackItem(item: MovieSummary | SeriesSummary): string | number {
    if (this.isMovie(item)) {
      return item.media_id;
    }
    return (item as SeriesSummary).series_metadata_id;
  }

  trackRecentItem(item: RecentlyAddedItem): string {
    if (item.type === 'movie') {
      return item.media_id || '';
    }
    return `season-${item.series_metadata_id}-${item.season_number}`;
  }

  isMovie(item: MovieSummary | SeriesSummary): item is MovieSummary {
    return 'media_id' in item;
  }

  asMovie(item: MovieSummary | SeriesSummary): MovieSummary {
    return item as MovieSummary;
  }

  asSeries(item: MovieSummary | SeriesSummary): SeriesSummary {
    return item as SeriesSummary;
  }

  onItemClick(item: MovieSummary | SeriesSummary): void {
    this.itemClick.emit(item);
  }

  onRecentItemClick(item: RecentlyAddedItem): void {
    this.recentItemClick.emit(item);
  }

  onSeeAll(): void {
    this.seeAllClick.emit();
  }
}
