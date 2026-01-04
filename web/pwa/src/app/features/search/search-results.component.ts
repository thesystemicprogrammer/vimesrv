import { Component, inject, OnInit, signal, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { TranslateModule } from '@ngx-translate/core';
import { ApiService, LibrarySearchResult } from '../../core/services/api.service';
import { LazyLoadDirective } from '../../shared/directives/lazy-load.directive';

@Component({
  selector: 'app-search-results',
  standalone: true,
  imports: [CommonModule, FormsModule, TranslateModule, LazyLoadDirective],
  template: `
    <div class="container mx-auto px-4 py-8">
      <!-- Search Header -->
      <div class="mb-8">
        <h1 class="text-2xl font-bold text-white mb-4">{{ 'search.results' | translate }}</h1>
        
        <!-- Search Input -->
        <div class="relative max-w-xl">
          <input
            type="text"
            [(ngModel)]="searchQuery"
            (ngModelChange)="onSearchChange()"
            (keydown.enter)="search()"
            [placeholder]="'search.placeholder' | translate"
            class="w-full pl-12 pr-4 py-3 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-blue-500"
          />
          <svg class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
          </svg>
          @if (searchQuery) {
            <button
              (click)="clearSearch()"
              class="absolute right-4 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          }
        </div>

        @if (lastSearchedQuery()) {
          <p class="mt-3 text-slate-400">
            {{ 'search.showingResultsFor' | translate }} "<span class="text-white">{{ lastSearchedQuery() }}</span>"
            @if (!loading()) {
              <span class="ml-1">({{ results().length }} {{ 'search.resultsCount' | translate }})</span>
            }
          </p>
        }
      </div>

      <!-- Results Grid -->
      @if (loading()) {
        <div class="flex justify-center items-center h-64">
          <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
        </div>
      } @else if (results().length === 0 && lastSearchedQuery()) {
        <div class="text-center py-16">
          <svg class="w-16 h-16 mx-auto text-slate-600 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
          </svg>
          <p class="text-xl text-slate-400 mb-2">{{ 'search.noResults' | translate }}</p>
          <p class="text-slate-500">{{ 'search.tryDifferentTerms' | translate }}</p>
        </div>
      } @else if (results().length > 0) {
        <!-- Filter Tabs -->
        <div class="flex gap-2 mb-6 border-b border-slate-700">
          <button
            (click)="setFilter('all')"
            [class]="getTabClass('all')"
          >
            {{ 'search.all' | translate }}
            <span class="ml-1.5 px-1.5 py-0.5 text-xs rounded-full bg-slate-600">{{ results().length }}</span>
          </button>
          <button
            (click)="setFilter('movie')"
            [class]="getTabClass('movie')"
          >
            {{ 'library.movies' | translate }}
            <span class="ml-1.5 px-1.5 py-0.5 text-xs rounded-full bg-slate-600">{{ movieCount() }}</span>
          </button>
          <button
            (click)="setFilter('series')"
            [class]="getTabClass('series')"
          >
            {{ 'library.series' | translate }}
            <span class="ml-1.5 px-1.5 py-0.5 text-xs rounded-full bg-slate-600">{{ seriesCount() }}</span>
          </button>
        </div>

        <!-- Results -->
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
          @for (result of filteredResults(); track result.media_id || result.series_metadata_id) {
            <button
              (click)="navigateToResult(result)"
              class="group text-left"
            >
              <!-- Poster -->
              <div class="relative aspect-[2/3] bg-slate-800 rounded-lg overflow-hidden mb-2">
                @if (result.poster_path) {
                  <img
                    appLazyLoad
                    [lazySrc]="'https://image.tmdb.org/t/p/w342' + result.poster_path"
                    [lazyAlt]="result.title"
                    class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                  />
                } @else {
                  <div class="w-full h-full flex items-center justify-center text-slate-600">
                    <svg class="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z"/>
                    </svg>
                  </div>
                }
                
                <!-- Type Badge -->
                <div class="absolute top-2 left-2">
                  <span class="px-2 py-0.5 text-xs font-medium rounded bg-black/70 text-white">
                    {{ result.type === 'movie' ? ('metadataMatch.movie' | translate) : ('metadataMatch.tvSeries' | translate) }}
                  </span>
                </div>

                <!-- Rating Badge -->
                @if (result.vote_average) {
                  <div class="absolute top-2 right-2 flex items-center gap-1 px-1.5 py-0.5 rounded bg-black/70">
                    <svg class="w-3 h-3 text-yellow-500" fill="currentColor" viewBox="0 0 20 20">
                      <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                    </svg>
                    <span class="text-xs text-white">{{ result.vote_average.toFixed(1) }}</span>
                  </div>
                }

                <!-- Hover Overlay -->
                <div class="absolute inset-0 bg-blue-600/0 group-hover:bg-blue-600/20 transition-colors"></div>
              </div>

              <!-- Title & Year -->
              <h3 class="text-white text-sm font-medium truncate group-hover:text-blue-400 transition-colors">
                {{ result.title }}
              </h3>
              @if (result.year) {
                <p class="text-slate-400 text-xs">{{ result.year }}</p>
              }
            </button>
          }
        </div>
      } @else {
        <!-- Initial State -->
        <div class="text-center py-16">
          <svg class="w-16 h-16 mx-auto text-slate-600 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
          </svg>
          <p class="text-xl text-slate-400 mb-2">{{ 'search.searchLibrary' | translate }}</p>
          <p class="text-slate-500">{{ 'search.enterQueryToSearch' | translate }}</p>
        </div>
      }
    </div>
  `
})
export class SearchResultsComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  searchQuery = '';
  lastSearchedQuery = signal('');
  loading = signal(false);
  results = signal<LibrarySearchResult[]>([]);
  activeFilter = signal<'all' | 'movie' | 'series'>('all');

  // Computed counts
  movieCount = computed(() => this.results().filter(r => r.type === 'movie').length);
  seriesCount = computed(() => this.results().filter(r => r.type === 'series').length);

  // Filtered results
  filteredResults = computed(() => {
    const filter = this.activeFilter();
    if (filter === 'all') return this.results();
    return this.results().filter(r => r.type === filter);
  });

  ngOnInit(): void {
    // Get query from URL params
    this.route.queryParams.subscribe(params => {
      const query = params['q'];
      if (query) {
        this.searchQuery = query;
        this.search();
      }
    });
  }

  onSearchChange(): void {
    // Update URL without navigating
    if (this.searchQuery.length >= 2) {
      this.router.navigate([], {
        relativeTo: this.route,
        queryParams: { q: this.searchQuery },
        queryParamsHandling: 'merge',
        replaceUrl: true
      });
    }
  }

  search(): void {
    if (this.searchQuery.length < 2) return;

    this.loading.set(true);
    this.lastSearchedQuery.set(this.searchQuery);
    this.activeFilter.set('all');

    this.api.searchLibrary(this.searchQuery, 100).subscribe({
      next: (response) => {
        this.results.set(response.data.results || []);
        this.loading.set(false);
      },
      error: () => {
        this.results.set([]);
        this.loading.set(false);
      }
    });
  }

  clearSearch(): void {
    this.searchQuery = '';
    this.results.set([]);
    this.lastSearchedQuery.set('');
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: { q: null },
      queryParamsHandling: 'merge'
    });
  }

  setFilter(filter: 'all' | 'movie' | 'series'): void {
    this.activeFilter.set(filter);
  }

  getTabClass(filter: 'all' | 'movie' | 'series'): string {
    const baseClass = 'px-4 py-2 font-medium text-sm transition border-b-2 -mb-px';
    if (this.activeFilter() === filter) {
      return `${baseClass} text-blue-400 border-blue-400`;
    }
    return `${baseClass} text-slate-400 border-transparent hover:text-white hover:border-slate-500`;
  }

  navigateToResult(result: LibrarySearchResult): void {
    if (result.type === 'movie' && result.media_id) {
      this.router.navigate(['/movie', result.media_id]);
    } else if (result.type === 'series' && result.series_metadata_id) {
      this.router.navigate(['/series', result.series_metadata_id]);
    }
  }
}
