import { Component, inject, signal, output, input, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TranslateModule } from '@ngx-translate/core';
import { FilterStateService } from '../../core/services/filter-state.service';
import { SortBy, SortOrder } from '../../core/services/api.service';

interface SortOption {
  sortBy: SortBy;
  sortOrder: SortOrder;
  labelKey: string;
}

@Component({
  selector: 'app-filter-bottom-sheet',
  standalone: true,
  imports: [CommonModule, FormsModule, TranslateModule],
  template: `
    <!-- Backdrop -->
    @if (isOpen()) {
      <div
        class="fixed inset-0 bg-black/60 z-40 transition-opacity"
        [class.opacity-100]="isOpen()"
        (click)="close()"
      ></div>
    }

    <!-- Bottom Sheet -->
    <div
      class="fixed inset-x-0 bottom-0 z-50 transform transition-transform duration-300 ease-out"
      [class.translate-y-full]="!isOpen()"
      [class.translate-y-0]="isOpen()"
    >
      <div class="bg-slate-800 rounded-t-2xl max-h-[85vh] overflow-hidden">
        <!-- Handle -->
        <div class="flex justify-center pt-3 pb-2">
          <div class="w-10 h-1 bg-slate-600 rounded-full"></div>
        </div>

        <!-- Header -->
        <div class="flex items-center justify-between px-4 pb-4 border-b border-slate-700">
          <h2 class="text-lg font-semibold text-white">{{ 'filter.title' | translate }}</h2>
          <div class="flex items-center gap-4">
            @if (filterState.hasActiveFilters()) {
              <button
                (click)="clearAll()"
                class="text-sm text-blue-400 hover:text-blue-300"
              >
                {{ 'filter.clearAll' | translate }}
              </button>
            }
            <button
              (click)="close()"
              class="text-slate-400 hover:text-white"
            >
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>
        </div>

        <!-- Content -->
        <div class="overflow-y-auto max-h-[calc(85vh-120px)] px-4 py-4">
          <!-- Sort Section -->
          <div class="mb-6">
            <h3 class="text-sm font-medium text-slate-400 mb-3">{{ 'filter.sortBy' | translate }}</h3>
            <div class="grid grid-cols-2 gap-2">
              @for (option of sortOptions; track option.sortBy + option.sortOrder) {
                <button
                  (click)="setSort(option)"
                  [class]="getSortOptionClass(option)"
                  class="px-3 py-2 text-sm rounded-lg transition"
                >
                  {{ option.labelKey | translate }}
                </button>
              }
            </div>
          </div>

          <!-- Genres Section -->
          @if (genres().length > 0) {
            <div class="mb-6">
              <h3 class="text-sm font-medium text-slate-400 mb-3">{{ 'filter.genres' | translate }}</h3>
              <div class="flex flex-wrap gap-2">
                @for (genre of genres(); track genre) {
                  <button
                    (click)="toggleGenre(genre)"
                    [class]="getGenrePillClass(genre)"
                    class="px-3 py-1.5 rounded-full text-sm transition"
                  >
                    {{ genre }}
                  </button>
                }
              </div>
            </div>
          }

          <!-- Year Range Section -->
          <div class="mb-6">
            <h3 class="text-sm font-medium text-slate-400 mb-3">{{ 'filter.yearRange' | translate }}</h3>
            <div class="flex items-center gap-3">
              <input
                type="number"
                [(ngModel)]="yearFrom"
                [placeholder]="'filter.from' | translate"
                (change)="updateYearRange()"
                min="1900"
                max="2030"
                class="flex-1 px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-blue-500 text-sm"
              />
              <span class="text-slate-400">-</span>
              <input
                type="number"
                [(ngModel)]="yearTo"
                [placeholder]="'filter.to' | translate"
                (change)="updateYearRange()"
                min="1900"
                max="2030"
                class="flex-1 px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-blue-500 text-sm"
              />
            </div>
          </div>

          <!-- Rating Section -->
          <div class="mb-6">
            <h3 class="text-sm font-medium text-slate-400 mb-3">{{ 'filter.minimumRating' | translate }}</h3>
            <div class="flex items-center gap-2">
              @for (rating of ratingOptions; track rating) {
                <button
                  (click)="setMinRating(rating)"
                  [class]="getRatingClass(rating)"
                  class="flex items-center gap-1 px-3 py-2 rounded-lg transition"
                >
                  <svg class="w-4 h-4 text-yellow-500" fill="currentColor" viewBox="0 0 20 20">
                    <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                  </svg>
                  <span class="text-sm text-white">{{ rating }}+</span>
                </button>
              }
              @if (filterState.minRating()) {
                <button
                  (click)="setMinRating(null)"
                  class="px-3 py-2 text-sm text-slate-400 hover:text-white transition"
                >
                  {{ 'filter.any' | translate }}
                </button>
              }
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="px-4 py-4 border-t border-slate-700">
          <button
            (click)="applyAndClose()"
            class="w-full py-3 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition"
          >
            {{ 'filter.showResults' | translate }}
          </button>
        </div>
      </div>
    </div>
  `
})
export class FilterBottomSheetComponent {
  readonly filterState = inject(FilterStateService);

  // Inputs
  genres = input<string[]>([]);

  // Outputs
  closed = output<void>();

  // State
  isOpen = signal(false);
  yearFrom: number | null = null;
  yearTo: number | null = null;

  // Sort options
  sortOptions: SortOption[] = [
    { sortBy: 'date_added', sortOrder: 'desc', labelKey: 'library.sort.recentlyAdded' },
    { sortBy: 'date_added', sortOrder: 'asc', labelKey: 'library.sort.oldestAdded' },
    { sortBy: 'title', sortOrder: 'asc', labelKey: 'library.sort.titleAZ' },
    { sortBy: 'title', sortOrder: 'desc', labelKey: 'library.sort.titleZA' },
    { sortBy: 'year', sortOrder: 'desc', labelKey: 'library.sort.newest' },
    { sortBy: 'year', sortOrder: 'asc', labelKey: 'library.sort.oldest' },
    { sortBy: 'rating', sortOrder: 'desc', labelKey: 'library.sort.highestRated' },
    { sortBy: 'rating', sortOrder: 'asc', labelKey: 'library.sort.lowestRated' },
  ];

  ratingOptions = [5, 6, 7, 8];

  open(): void {
    // Load current values
    this.yearFrom = this.filterState.yearFrom();
    this.yearTo = this.filterState.yearTo();
    this.isOpen.set(true);
    // Prevent body scroll
    document.body.style.overflow = 'hidden';
  }

  close(): void {
    this.isOpen.set(false);
    document.body.style.overflow = '';
    this.closed.emit();
  }

  applyAndClose(): void {
    this.close();
  }

  clearAll(): void {
    this.filterState.clearAllFilters();
    this.yearFrom = null;
    this.yearTo = null;
  }

  setSort(option: SortOption): void {
    this.filterState.setSort(option.sortBy, option.sortOrder);
  }

  getSortOptionClass(option: SortOption): string {
    const isSelected = this.filterState.sortBy() === option.sortBy && this.filterState.sortOrder() === option.sortOrder;
    if (isSelected) {
      return 'bg-blue-600 text-white';
    }
    return 'bg-slate-700 text-slate-300 hover:bg-slate-600';
  }

  toggleGenre(genre: string): void {
    this.filterState.toggleGenre(genre);
  }

  getGenrePillClass(genre: string): string {
    const isSelected = this.filterState.selectedGenres().includes(genre);
    if (isSelected) {
      return 'bg-blue-600 text-white';
    }
    return 'bg-slate-700 text-slate-300 hover:bg-slate-600';
  }

  updateYearRange(): void {
    this.filterState.setYearRange(this.yearFrom, this.yearTo);
  }

  setMinRating(rating: number | null): void {
    this.filterState.setMinRating(rating);
  }

  getRatingClass(rating: number): string {
    const isSelected = this.filterState.minRating() === rating;
    if (isSelected) {
      return 'bg-blue-600';
    }
    return 'bg-slate-700 hover:bg-slate-600';
  }
}
