import { Injectable, signal, computed } from '@angular/core';
import { SortBy, SortOrder } from './api.service';

const STORAGE_KEY = 'vimesrv_filter_state';

export interface FilterState {
  sortBy: SortBy;
  sortOrder: SortOrder;
  selectedGenres: string[];
  yearFrom: number | null;
  yearTo: number | null;
  minRating: number | null;
}

const defaultState: FilterState = {
  sortBy: 'date_added',
  sortOrder: 'desc',
  selectedGenres: [],
  yearFrom: null,
  yearTo: null,
  minRating: null,
};

@Injectable({
  providedIn: 'root'
})
export class FilterStateService {
  // Signals for each filter property
  readonly sortBy = signal<SortBy>(defaultState.sortBy);
  readonly sortOrder = signal<SortOrder>(defaultState.sortOrder);
  readonly selectedGenres = signal<string[]>(defaultState.selectedGenres);
  readonly yearFrom = signal<number | null>(defaultState.yearFrom);
  readonly yearTo = signal<number | null>(defaultState.yearTo);
  readonly minRating = signal<number | null>(defaultState.minRating);

  // Computed to check if any filters are active (besides default sort)
  readonly hasActiveFilters = computed(() => {
    return (
      this.selectedGenres().length > 0 ||
      this.yearFrom() !== null ||
      this.yearTo() !== null ||
      this.minRating() !== null
    );
  });

  // Computed to get sort display label
  readonly sortLabel = computed(() => {
    const sortBy = this.sortBy();
    const order = this.sortOrder();
    const labels: Record<SortBy, string> = {
      date_added: order === 'desc' ? 'Recently Added' : 'Oldest Added',
      title: order === 'asc' ? 'Title A-Z' : 'Title Z-A',
      year: order === 'desc' ? 'Newest' : 'Oldest',
      rating: order === 'desc' ? 'Highest Rated' : 'Lowest Rated',
    };
    return labels[sortBy];
  });

  constructor() {
    this.loadFromStorage();
  }

  private loadFromStorage(): void {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        const state = JSON.parse(stored) as Partial<FilterState>;
        if (state.sortBy) this.sortBy.set(state.sortBy);
        if (state.sortOrder) this.sortOrder.set(state.sortOrder);
        if (state.selectedGenres) this.selectedGenres.set(state.selectedGenres);
        if (state.yearFrom !== undefined) this.yearFrom.set(state.yearFrom);
        if (state.yearTo !== undefined) this.yearTo.set(state.yearTo);
        if (state.minRating !== undefined) this.minRating.set(state.minRating);
      }
    } catch {
      // Ignore storage errors
    }
  }

  private saveToStorage(): void {
    try {
      const state: FilterState = {
        sortBy: this.sortBy(),
        sortOrder: this.sortOrder(),
        selectedGenres: this.selectedGenres(),
        yearFrom: this.yearFrom(),
        yearTo: this.yearTo(),
        minRating: this.minRating(),
      };
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } catch {
      // Ignore storage errors
    }
  }

  setSort(sortBy: SortBy, sortOrder: SortOrder): void {
    this.sortBy.set(sortBy);
    this.sortOrder.set(sortOrder);
    this.saveToStorage();
  }

  toggleGenre(genre: string): void {
    const current = this.selectedGenres();
    const index = current.indexOf(genre);
    if (index === -1) {
      this.selectedGenres.set([...current, genre]);
    } else {
      this.selectedGenres.set(current.filter(g => g !== genre));
    }
    this.saveToStorage();
  }

  clearGenres(): void {
    this.selectedGenres.set([]);
    this.saveToStorage();
  }

  setYearRange(from: number | null, to: number | null): void {
    this.yearFrom.set(from);
    this.yearTo.set(to);
    this.saveToStorage();
  }

  setMinRating(rating: number | null): void {
    this.minRating.set(rating);
    this.saveToStorage();
  }

  clearAllFilters(): void {
    this.sortBy.set(defaultState.sortBy);
    this.sortOrder.set(defaultState.sortOrder);
    this.selectedGenres.set(defaultState.selectedGenres);
    this.yearFrom.set(defaultState.yearFrom);
    this.yearTo.set(defaultState.yearTo);
    this.minRating.set(defaultState.minRating);
    this.saveToStorage();
  }

  // Get filter options for API call
  getApiFilterOptions() {
    return {
      sortBy: this.sortBy(),
      sortOrder: this.sortOrder(),
      genres: this.selectedGenres().length > 0 ? this.selectedGenres() : undefined,
      yearFrom: this.yearFrom() ?? undefined,
      yearTo: this.yearTo() ?? undefined,
      minRating: this.minRating() ?? undefined,
    };
  }
}
