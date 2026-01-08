import { Component, inject, OnInit, OnDestroy, signal, computed, ViewChild, effect, AfterViewInit } from '@angular/core';
import { Router } from '@angular/router';
import { toObservable } from '@angular/core/rxjs-interop';
import { forkJoin, Subscription, skip } from 'rxjs';
import { TranslateModule, TranslateService } from '@ngx-translate/core';
import {
  ApiService,
  MovieSummary,
  SeriesSummary,
  UnmatchedMediaSummary,
  RecentlyAddedItem,
  SortBy,
  SortOrder
} from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';
import { FilterStateService } from '../../core/services/filter-state.service';
import { ScrollStateService } from '../../core/services/scroll-state.service';
import { MediaCardComponent } from './media-card.component';
import { MediaRowComponent } from './media-row.component';
import { MetadataMatchModalComponent } from './metadata-match-modal.component';
import { FilterBottomSheetComponent } from './filter-bottom-sheet.component';
import { ContinueWatchingRowComponent } from './continue-watching-row.component';
import { FavoritesRowComponent } from './favorites-row.component';
import { RecommendationsRowComponent } from './recommendations-row.component';

type FilterTab = 'all' | 'movies' | 'series' | 'unmatched';

interface SortOption {
  sortBy: SortBy;
  sortOrder: SortOrder;
  labelKey: string;
}

@Component({
  selector: 'app-library',
  standalone: true,
  imports: [MediaCardComponent, MediaRowComponent, MetadataMatchModalComponent, FilterBottomSheetComponent, ContinueWatchingRowComponent, FavoritesRowComponent, RecommendationsRowComponent, TranslateModule],
  template: `
    <div class="container mx-auto px-4 py-8">
      <!-- Header -->
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-3xl font-bold text-white">{{ 'library.title' | translate }}</h1>
        <button
          (click)="scanLibrary()"
          [disabled]="scanning()"
          class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md transition disabled:opacity-50"
        >
          @if (scanning()) {
            {{ 'library.scanning' | translate }}
          } @else {
            {{ 'library.scanLibrary' | translate }}
          }
        </button>
      </div>

      <!-- Filter Tabs -->
      <div class="flex gap-2 mb-6 border-b border-slate-700">
        @for (tab of tabs; track tab.id) {
          <button
            (click)="setActiveTab(tab.id)"
            [class]="getTabClass(tab.id)"
          >
            {{ tab.labelKey | translate }}
            @if (tab.count() !== null) {
              <span class="ml-1.5 px-1.5 py-0.5 text-xs rounded-full bg-slate-600">
                {{ tab.count() }}
              </span>
            }
          </button>
        }
      </div>

      <!-- Sort and Filter Controls (for movies/series tabs) -->
      @if (activeTab() === 'movies' || activeTab() === 'series') {
        <div class="flex flex-wrap items-center gap-4 mb-6">
          <!-- Mobile Filter Button -->
          <button
            (click)="openFilterSheet()"
            class="sm:hidden flex items-center gap-2 px-3 py-2 bg-slate-800 hover:bg-slate-700 text-white rounded-lg transition text-sm"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"/>
            </svg>
            <span>{{ 'filter.title' | translate }}</span>
            @if (filterState.hasActiveFilters()) {
              <span class="w-2 h-2 bg-blue-500 rounded-full"></span>
            }
          </button>

          <!-- Sort Dropdown (desktop) -->
          <div class="relative hidden sm:block">
            <button
              (click)="toggleSortDropdown()"
              class="flex items-center gap-2 px-3 py-2 bg-slate-800 hover:bg-slate-700 text-white rounded-lg transition text-sm"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12"/>
              </svg>
              <span>{{ filterState.sortLabel() }}</span>
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
              </svg>
            </button>
            @if (showSortDropdown()) {
              <div class="absolute top-full left-0 mt-1 w-48 bg-slate-800 rounded-lg shadow-lg border border-slate-700 z-10">
                @for (option of sortOptions; track option.sortBy + option.sortOrder) {
                  <button
                    (click)="setSort(option)"
                    [class]="getSortOptionClass(option)"
                    class="w-full text-left px-4 py-2 text-sm transition first:rounded-t-lg last:rounded-b-lg"
                  >
                    {{ option.labelKey | translate }}
                  </button>
                }
              </div>
            }
          </div>

          <!-- Genre Pills (desktop) -->
          @if (availableGenres().length > 0) {
            <div class="hidden sm:flex flex-wrap gap-2 flex-1">
              @for (genre of availableGenres().slice(0, showAllGenres() ? undefined : 8); track genre) {
                <button
                  (click)="toggleGenre(genre)"
                  [class]="getGenrePillClass(genre)"
                  class="px-3 py-1 rounded-full text-xs font-medium transition"
                >
                  {{ genre }}
                </button>
              }
              @if (availableGenres().length > 8) {
                <button
                  (click)="showAllGenres.set(!showAllGenres())"
                  class="px-3 py-1 text-blue-400 hover:text-blue-300 text-xs font-medium transition"
                >
                  {{ showAllGenres() ? ('library.showLess' | translate) : ('+' + (availableGenres().length - 8) + ' ' + ('library.more' | translate)) }}
                </button>
              }
            </div>
          }

          <!-- Clear Filters -->
          @if (filterState.hasActiveFilters()) {
            <button
              (click)="clearFilters()"
              class="text-sm text-slate-400 hover:text-white transition"
            >
              {{ 'library.clearFilters' | translate }}
            </button>
          }
        </div>
      }

      <!-- Continue Watching, Favorites and Recommendations Rows (only shown on All tab) -->
      @if (!loading() && !error() && activeTab() === 'all') {
        <app-continue-watching-row />
        <app-favorites-row />
        <app-recommendations-row />
      }

      @if (loading()) {
        <div class="flex justify-center items-center h-64">
          <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
        </div>
      } @else if (error()) {
        <div class="bg-red-500/10 border border-red-500 text-red-400 px-4 py-3 rounded">
          {{ error() }}
        </div>
      } @else {
        <!-- All Tab: Shows Recently Added row + Movies + Series -->
        @if (activeTab() === 'all') {
          @if (recentItems().length > 0) {
            <app-media-row
              [title]="'library.recentlyAdded' | translate"
              cardType="recent"
              [recentItems]="recentItems()"
              (recentItemClick)="onRecentItemClick($event)"
            />
          }

          @if (movies().length > 0) {
            <app-media-row
              [title]="'library.movies' | translate"
              cardType="movie"
              [items]="movies().slice(0, 10)"
              [showSeeAll]="true"
              (itemClick)="onMovieClick($event)"
              (seeAllClick)="setActiveTab('movies')"
            />
          }

          @if (series().length > 0) {
            <app-media-row
              [title]="'library.series' | translate"
              cardType="series"
              [items]="series().slice(0, 10)"
              [showSeeAll]="true"
              (itemClick)="onSeriesClick($event)"
              (seeAllClick)="setActiveTab('series')"
            />
          }

          @if (movies().length === 0 && series().length === 0) {
            <div class="text-center text-slate-400 py-16">
              <p class="text-xl mb-4">{{ 'library.noMediaFound' | translate }}</p>
              <p>{{ 'library.clickScanToDiscover' | translate }}</p>
            </div>
          }
        }

        <!-- Movies Tab -->
        @if (activeTab() === 'movies') {
          @if (movies().length === 0 && !loadingMore()) {
            <div class="text-center text-slate-400 py-16">
              <p class="text-xl">{{ filterState.hasActiveFilters() ? ('library.noMoviesMatchFilters' | translate) : ('library.noMoviesFound' | translate) }}</p>
              @if (filterState.hasActiveFilters()) {
                <button
                  (click)="clearFilters()"
                  class="mt-4 text-blue-400 hover:text-blue-300 transition"
                >
                  {{ 'library.clearFilters' | translate }}
                </button>
              }
            </div>
          } @else {
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-2 sm:gap-3 md:gap-4">
              @for (movie of movies(); track movie.media_id) {
                <app-media-card
                  cardType="movie"
                  [movie]="movie"
                  (cardClick)="navigateToMovie(movie)"
                />
              }
            </div>
            @if (hasMoreMovies()) {
              <div class="flex justify-center mt-8">
                <button
                  (click)="loadMoreMovies()"
                  [disabled]="loadingMore()"
                  class="px-6 py-3 bg-slate-700 hover:bg-slate-600 text-white rounded-lg transition disabled:opacity-50"
                >
                  @if (loadingMore()) {
                    <span class="flex items-center gap-2">
                      <div class="animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-white"></div>
                      {{ 'library.loading' | translate }}
                    </span>
                  } @else {
                    {{ 'library.loadMore' | translate }}
                  }
                </button>
              </div>
            }
          }
        }

        <!-- Series Tab -->
        @if (activeTab() === 'series') {
          @if (series().length === 0 && !loadingMore()) {
            <div class="text-center text-slate-400 py-16">
              <p class="text-xl">{{ filterState.hasActiveFilters() ? ('library.noSeriesMatchFilters' | translate) : ('library.noSeriesFound' | translate) }}</p>
              @if (filterState.hasActiveFilters()) {
                <button
                  (click)="clearFilters()"
                  class="mt-4 text-blue-400 hover:text-blue-300 transition"
                >
                  {{ 'library.clearFilters' | translate }}
                </button>
              }
            </div>
          } @else {
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-2 sm:gap-3 md:gap-4">
              @for (s of series(); track s.series_metadata_id) {
                <app-media-card
                  cardType="series"
                  [series]="s"
                  (cardClick)="navigateToSeries(s)"
                />
              }
            </div>
            @if (hasMoreSeries()) {
              <div class="flex justify-center mt-8">
                <button
                  (click)="loadMoreSeries()"
                  [disabled]="loadingMore()"
                  class="px-6 py-3 bg-slate-700 hover:bg-slate-600 text-white rounded-lg transition disabled:opacity-50"
                >
                  @if (loadingMore()) {
                    <span class="flex items-center gap-2">
                      <div class="animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-white"></div>
                      {{ 'library.loading' | translate }}
                    </span>
                  } @else {
                    {{ 'library.loadMore' | translate }}
                  }
                </button>
              </div>
            }
          }
        }

        <!-- Unmatched Tab -->
        @if (activeTab() === 'unmatched') {
          @if (unmatched().length === 0) {
            <div class="text-center text-slate-400 py-16">
              <p class="text-xl mb-2">{{ 'library.noUnmatchedMedia' | translate }}</p>
              <p class="text-sm">{{ 'library.allMediaMatched' | translate }}</p>
            </div>
          } @else {
            <div class="space-y-2">
              @for (item of unmatched(); track item.media_id) {
                <div
                  class="flex items-center justify-between p-4 bg-slate-800 rounded-lg hover:bg-slate-700 transition"
                >
                  <div class="flex-1 min-w-0">
                    <h3 class="text-white font-medium truncate">{{ item.title }}</h3>
                    <p class="text-slate-400 text-sm truncate">{{ item.filename }}</p>
                    <div class="flex gap-3 mt-1 text-xs text-slate-500">
                      <span>{{ item.resolution }}</span>
                      <span>{{ formatDuration(item.duration) }}</span>
                      <span class="text-yellow-500">{{ formatStatus(item.enrichment_status) }}</span>
                    </div>
                  </div>
                  <button
                    (click)="openMatchModal(item)"
                    class="ml-4 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-sm rounded transition"
                  >
                    {{ 'library.fixMatch' | translate }}
                  </button>
                </div>
              }
            </div>
          }
        }
      }
    </div>

    <!-- Metadata Match Modal -->
    <app-metadata-match-modal
      #matchModal
      [media]="selectedUnmatchedMedia()"
      (matched)="onMetadataMatched()"
      (skipped)="onMetadataSkipped()"
    />

    <!-- Filter Bottom Sheet (mobile) -->
    <app-filter-bottom-sheet
      #filterSheet
      [genres]="availableGenres()"
    />
  `
})
export class LibraryComponent implements OnInit, OnDestroy, AfterViewInit {
  private readonly api = inject(ApiService);
  private readonly router = inject(Router);
  private readonly auth = inject(AuthService);
  private readonly translate = inject(TranslateService);
  readonly filterState = inject(FilterStateService);
  private readonly scrollState = inject(ScrollStateService);

  private languageSubscription: Subscription | null = null;
  private pendingScrollRestore = false;

  @ViewChild('matchModal') matchModal!: MetadataMatchModalComponent;
  @ViewChild('filterSheet') filterSheet!: FilterBottomSheetComponent;

  // Data signals
  movies = signal<MovieSummary[]>([]);
  series = signal<SeriesSummary[]>([]);
  recentItems = signal<RecentlyAddedItem[]>([]);
  unmatched = signal<UnmatchedMediaSummary[]>([]);
  selectedUnmatchedMedia = signal<UnmatchedMediaSummary | null>(null);

  // Genre data
  movieGenres = signal<string[]>([]);
  seriesGenres = signal<string[]>([]);

  // UI state
  loading = signal(true);
  loadingMore = signal(false);
  error = signal<string | null>(null);
  scanning = signal(false);
  activeTab = signal<FilterTab>('all');
  showSortDropdown = signal(false);
  showAllGenres = signal(false);

  // Pagination state
  moviesPage = signal(1);
  seriesPage = signal(1);
  moviesTotalCount = signal(0);
  seriesTotalCount = signal(0);
  readonly perPage = 20;

  // Computed: available genres based on current tab
  availableGenres = computed(() => {
    return this.activeTab() === 'series' ? this.seriesGenres() : this.movieGenres();
  });

  // Computed: has more items to load
  hasMoreMovies = computed(() => this.movies().length < this.moviesTotalCount());
  hasMoreSeries = computed(() => this.series().length < this.seriesTotalCount());

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

  // Tab configuration with computed counts
  tabs = [
    { id: 'all' as const, labelKey: 'library.all', count: computed(() => null) },
    { id: 'movies' as const, labelKey: 'library.movies', count: computed(() => this.moviesTotalCount()) },
    { id: 'series' as const, labelKey: 'library.series', count: computed(() => this.seriesTotalCount()) },
    { id: 'unmatched' as const, labelKey: 'library.unmatched', count: computed(() => this.unmatched().length) }
  ];

  constructor() {
    // Re-fetch library when language changes (skip initial emission)
    this.languageSubscription = toObservable(this.auth.language)
      .pipe(skip(1))
      .subscribe(() => {
        this.loadLibrary();
      });

    // Re-fetch when filter state changes
    effect(() => {
      // Access signals to track changes
      this.filterState.sortBy();
      this.filterState.sortOrder();
      this.filterState.selectedGenres();
      this.filterState.yearFrom();
      this.filterState.yearTo();
      this.filterState.minRating();

      // Only reload if not in initial loading state
      if (!this.loading()) {
        this.resetAndReloadFiltered();
      }
    });
  }

  ngOnDestroy(): void {
    this.languageSubscription?.unsubscribe();
  }

  ngOnInit(): void {
    // Check if we're restoring from a back navigation
    const savedState = this.scrollState.getState();
    if (savedState) {
      this.pendingScrollRestore = true;
      this.activeTab.set(savedState.activeTab as FilterTab);
      this.loadLibraryWithRestoration(savedState);
    } else {
      this.loadLibrary();
    }
  }

  ngAfterViewInit(): void {
    // Scroll restoration happens after data is loaded (see loadLibraryWithRestoration)
  }

  /**
   * Load library with state restoration (for back navigation)
   */
  private loadLibraryWithRestoration(savedState: { scrollY: number; activeTab: string; moviesPage: number; seriesPage: number; moviesCount: number; seriesCount: number }): void {
    this.loading.set(true);
    this.error.set(null);

    const filterOptions = this.filterState.getApiFilterOptions();

    // Load enough items to restore the previous scroll position
    const moviesPerPage = savedState.moviesPage * this.perPage;
    const seriesPerPage = savedState.seriesPage * this.perPage;

    forkJoin({
      movies: this.api.listMovies({ page: 1, perPage: moviesPerPage, ...filterOptions }),
      series: this.api.listSeries({ page: 1, perPage: seriesPerPage, ...filterOptions }),
      recent: this.api.listRecent(),
      unmatched: this.api.listUnmatched(1, 100),
      genres: this.api.listGenres()
    }).subscribe({
      next: (responses) => {
        this.movies.set(responses.movies.data.items || []);
        this.moviesTotalCount.set(responses.movies.data.total);
        this.moviesPage.set(savedState.moviesPage);

        this.series.set(responses.series.data.items || []);
        this.seriesTotalCount.set(responses.series.data.total);
        this.seriesPage.set(savedState.seriesPage);

        this.recentItems.set(responses.recent.data.items || []);
        this.unmatched.set(responses.unmatched.data.items || []);

        this.movieGenres.set(responses.genres.data.movie_genres || []);
        this.seriesGenres.set(responses.genres.data.series_genres || []);

        this.loading.set(false);

        // Restore scroll position after a brief delay for DOM to update
        if (this.pendingScrollRestore) {
          setTimeout(() => {
            window.scrollTo({ top: savedState.scrollY, behavior: 'instant' });
            this.pendingScrollRestore = false;
            this.scrollState.clearState();
          }, 50);
        }
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Failed to load library');
        this.loading.set(false);
        this.scrollState.clearState();
      }
    });
  }

  loadLibrary(): void {
    this.loading.set(true);
    this.error.set(null);

    const filterOptions = this.filterState.getApiFilterOptions();

    forkJoin({
      movies: this.api.listMovies({ page: 1, perPage: this.perPage, ...filterOptions }),
      series: this.api.listSeries({ page: 1, perPage: this.perPage, ...filterOptions }),
      recent: this.api.listRecent(),
      unmatched: this.api.listUnmatched(1, 100),
      genres: this.api.listGenres()
    }).subscribe({
      next: (responses) => {
        this.movies.set(responses.movies.data.items || []);
        this.moviesTotalCount.set(responses.movies.data.total);
        this.moviesPage.set(1);

        this.series.set(responses.series.data.items || []);
        this.seriesTotalCount.set(responses.series.data.total);
        this.seriesPage.set(1);

        this.recentItems.set(responses.recent.data.items || []);
        this.unmatched.set(responses.unmatched.data.items || []);

        this.movieGenres.set(responses.genres.data.movie_genres || []);
        this.seriesGenres.set(responses.genres.data.series_genres || []);

        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Failed to load library');
        this.loading.set(false);
      }
    });
  }

  private resetAndReloadFiltered(): void {
    const filterOptions = this.filterState.getApiFilterOptions();

    // Reset pagination
    this.moviesPage.set(1);
    this.seriesPage.set(1);

    // Reload with new filters
    forkJoin({
      movies: this.api.listMovies({ page: 1, perPage: this.perPage, ...filterOptions }),
      series: this.api.listSeries({ page: 1, perPage: this.perPage, ...filterOptions })
    }).subscribe({
      next: (responses) => {
        this.movies.set(responses.movies.data.items || []);
        this.moviesTotalCount.set(responses.movies.data.total);
        this.series.set(responses.series.data.items || []);
        this.seriesTotalCount.set(responses.series.data.total);
      }
    });
  }

  loadMoreMovies(): void {
    if (this.loadingMore() || !this.hasMoreMovies()) return;

    this.loadingMore.set(true);
    const nextPage = this.moviesPage() + 1;
    const filterOptions = this.filterState.getApiFilterOptions();

    this.api.listMovies({ page: nextPage, perPage: this.perPage, ...filterOptions }).subscribe({
      next: (response) => {
        this.movies.update(current => [...current, ...(response.data.items || [])]);
        this.moviesPage.set(nextPage);
        this.loadingMore.set(false);
      },
      error: () => {
        this.loadingMore.set(false);
      }
    });
  }

  loadMoreSeries(): void {
    if (this.loadingMore() || !this.hasMoreSeries()) return;

    this.loadingMore.set(true);
    const nextPage = this.seriesPage() + 1;
    const filterOptions = this.filterState.getApiFilterOptions();

    this.api.listSeries({ page: nextPage, perPage: this.perPage, ...filterOptions }).subscribe({
      next: (response) => {
        this.series.update(current => [...current, ...(response.data.items || [])]);
        this.seriesPage.set(nextPage);
        this.loadingMore.set(false);
      },
      error: () => {
        this.loadingMore.set(false);
      }
    });
  }

  setActiveTab(tab: FilterTab): void {
    this.activeTab.set(tab);
    this.showSortDropdown.set(false);
  }

  openFilterSheet(): void {
    this.filterSheet.open();
  }

  getTabClass(tabId: FilterTab): string {
    const baseClass = 'px-4 py-2 font-medium text-sm transition border-b-2 -mb-px';
    if (this.activeTab() === tabId) {
      return `${baseClass} text-blue-400 border-blue-400`;
    }
    return `${baseClass} text-slate-400 border-transparent hover:text-white hover:border-slate-500`;
  }

  toggleSortDropdown(): void {
    this.showSortDropdown.update(v => !v);
  }

  setSort(option: SortOption): void {
    this.filterState.setSort(option.sortBy, option.sortOrder);
    this.showSortDropdown.set(false);
  }

  getSortOptionClass(option: SortOption): string {
    const isSelected = this.filterState.sortBy() === option.sortBy && this.filterState.sortOrder() === option.sortOrder;
    if (isSelected) {
      return 'text-blue-400 bg-slate-700';
    }
    return 'text-slate-300 hover:bg-slate-700';
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

  clearFilters(): void {
    this.filterState.clearAllFilters();
  }

  onMovieClick(item: MovieSummary | SeriesSummary): void {
    if ('media_id' in item) {
      this.navigateToMovie(item);
    }
  }

  onSeriesClick(item: MovieSummary | SeriesSummary): void {
    if ('series_metadata_id' in item) {
      this.navigateToSeries(item as SeriesSummary);
    }
  }

  onRecentItemClick(item: RecentlyAddedItem): void {
    this.saveScrollState();
    if (item.type === 'movie' && item.media_id) {
      this.router.navigate(['/movie', item.media_id]);
    } else if (item.type === 'season' && item.series_metadata_id) {
      // Navigate to series detail with season query param
      this.router.navigate(['/series', item.series_metadata_id], {
        queryParams: { season: item.season_number }
      });
    }
  }

  navigateToMovie(movie: MovieSummary): void {
    this.saveScrollState();
    this.router.navigate(['/movie', movie.media_id]);
  }

  navigateToSeries(s: SeriesSummary): void {
    this.saveScrollState();
    this.router.navigate(['/series', s.series_metadata_id]);
  }

  /**
   * Save scroll and pagination state before navigating away
   */
  private saveScrollState(): void {
    this.scrollState.saveState({
      scrollY: window.scrollY,
      activeTab: this.activeTab(),
      moviesPage: this.moviesPage(),
      seriesPage: this.seriesPage(),
      moviesCount: this.movies().length,
      seriesCount: this.series().length
    });
  }

  openMatchModal(item: UnmatchedMediaSummary): void {
    this.selectedUnmatchedMedia.set(item);
    // Use setTimeout to ensure the input is set before opening
    setTimeout(() => {
      this.matchModal.open();
    }, 0);
  }

  onMetadataMatched(): void {
    // Reload library to reflect the change
    this.loadLibrary();
  }

  onMetadataSkipped(): void {
    // Reload library to reflect the change
    this.loadLibrary();
  }

  scanLibrary(): void {
    this.scanning.set(true);
    this.api.scanLibrary().subscribe({
      next: () => {
        setTimeout(() => {
          this.scanning.set(false);
          this.loadLibrary();
        }, 2000);
      },
      error: () => {
        this.scanning.set(false);
      }
    });
  }

  formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }

  formatStatus(status: string): string {
    switch (status) {
      case 'pending': return this.translate.instant('library.pending');
      case 'not_found': return this.translate.instant('library.notFound');
      case 'failed': return this.translate.instant('library.failed');
      case 'skipped': return this.translate.instant('library.skipped');
      case 'candidates_found': return this.translate.instant('library.candidatesFound');
      case 'manual_required': return this.translate.instant('library.manualRequired');
      default: return status;
    }
  }
}
