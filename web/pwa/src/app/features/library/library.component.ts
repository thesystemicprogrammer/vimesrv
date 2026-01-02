import { Component, inject, OnInit, OnDestroy, signal, computed, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { toObservable } from '@angular/core/rxjs-interop';
import { forkJoin, Subscription, skip } from 'rxjs';
import {
  ApiService,
  MovieSummary,
  SeriesSummary,
  UnmatchedMediaSummary,
  RecentlyAddedItem
} from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';
import { MediaCardComponent } from './media-card.component';
import { MediaRowComponent } from './media-row.component';
import { MetadataMatchModalComponent } from './metadata-match-modal.component';

type FilterTab = 'all' | 'movies' | 'series' | 'unmatched';

@Component({
  selector: 'app-library',
  standalone: true,
  imports: [MediaCardComponent, MediaRowComponent, MetadataMatchModalComponent],
  template: `
    <div class="container mx-auto px-4 py-8">
      <!-- Header -->
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-3xl font-bold text-white">Library</h1>
        <button
          (click)="scanLibrary()"
          [disabled]="scanning()"
          class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md transition disabled:opacity-50"
        >
          @if (scanning()) {
            Scanning...
          } @else {
            Scan Library
          }
        </button>
      </div>

      <!-- Filter Tabs -->
      <div class="flex gap-2 mb-8 border-b border-slate-700">
        @for (tab of tabs; track tab.id) {
          <button
            (click)="setActiveTab(tab.id)"
            [class]="getTabClass(tab.id)"
          >
            {{ tab.label }}
            @if (tab.count() !== null) {
              <span class="ml-1.5 px-1.5 py-0.5 text-xs rounded-full bg-slate-600">
                {{ tab.count() }}
              </span>
            }
          </button>
        }
      </div>

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
              title="Recently Added"
              cardType="recent"
              [recentItems]="recentItems()"
              (recentItemClick)="onRecentItemClick($event)"
            />
          }

          @if (movies().length > 0) {
            <section class="mb-10">
              <div class="flex justify-between items-center mb-4">
                <h2 class="text-xl font-bold text-white">Movies</h2>
                <button
                  (click)="setActiveTab('movies')"
                  class="text-blue-400 hover:text-blue-300 text-sm font-medium transition"
                >
                  See all
                </button>
              </div>
              <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-5 gap-4">
                @for (movie of movies().slice(0, 6); track movie.media_id) {
                  <app-media-card
                    cardType="movie"
                    [movie]="movie"
                    (cardClick)="navigateToMovie(movie)"
                  />
                }
              </div>
            </section>
          }

          @if (series().length > 0) {
            <section class="mb-10">
              <div class="flex justify-between items-center mb-4">
                <h2 class="text-xl font-bold text-white">Series</h2>
                <button
                  (click)="setActiveTab('series')"
                  class="text-blue-400 hover:text-blue-300 text-sm font-medium transition"
                >
                  See all
                </button>
              </div>
              <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-5 gap-4">
                @for (s of series().slice(0, 6); track s.series_metadata_id) {
                  <app-media-card
                    cardType="series"
                    [series]="s"
                    (cardClick)="navigateToSeries(s)"
                  />
                }
              </div>
            </section>
          }

          @if (movies().length === 0 && series().length === 0) {
            <div class="text-center text-slate-400 py-16">
              <p class="text-xl mb-4">No media found</p>
              <p>Click "Scan Library" to discover media files</p>
            </div>
          }
        }

        <!-- Movies Tab -->
        @if (activeTab() === 'movies') {
          @if (movies().length === 0) {
            <div class="text-center text-slate-400 py-16">
              <p class="text-xl">No movies found</p>
            </div>
          } @else {
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-5 gap-4">
              @for (movie of movies(); track movie.media_id) {
                <app-media-card
                  cardType="movie"
                  [movie]="movie"
                  (cardClick)="navigateToMovie(movie)"
                />
              }
            </div>
          }
        }

        <!-- Series Tab -->
        @if (activeTab() === 'series') {
          @if (series().length === 0) {
            <div class="text-center text-slate-400 py-16">
              <p class="text-xl">No series found</p>
            </div>
          } @else {
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-5 gap-4">
              @for (s of series(); track s.series_metadata_id) {
                <app-media-card
                  cardType="series"
                  [series]="s"
                  (cardClick)="navigateToSeries(s)"
                />
              }
            </div>
          }
        }

        <!-- Unmatched Tab -->
        @if (activeTab() === 'unmatched') {
          @if (unmatched().length === 0) {
            <div class="text-center text-slate-400 py-16">
              <p class="text-xl mb-2">No unmatched media</p>
              <p class="text-sm">All your media files have been matched with metadata</p>
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
                    Fix Match
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
  `
})
export class LibraryComponent implements OnInit, OnDestroy {
  private readonly api = inject(ApiService);
  private readonly router = inject(Router);
  private readonly auth = inject(AuthService);

  private languageSubscription: Subscription | null = null;

  @ViewChild('matchModal') matchModal!: MetadataMatchModalComponent;

  // Data signals
  movies = signal<MovieSummary[]>([]);
  series = signal<SeriesSummary[]>([]);
  recentItems = signal<RecentlyAddedItem[]>([]);
  unmatched = signal<UnmatchedMediaSummary[]>([]);
  selectedUnmatchedMedia = signal<UnmatchedMediaSummary | null>(null);

  // UI state
  loading = signal(true);
  error = signal<string | null>(null);
  scanning = signal(false);
  activeTab = signal<FilterTab>('all');

  // Tab configuration with computed counts
  tabs = [
    { id: 'all' as const, label: 'All', count: computed(() => null) },
    { id: 'movies' as const, label: 'Movies', count: computed(() => this.movies().length) },
    { id: 'series' as const, label: 'Series', count: computed(() => this.series().length) },
    { id: 'unmatched' as const, label: 'Unmatched', count: computed(() => this.unmatched().length) }
  ];

  constructor() {
    // Re-fetch library when language changes (skip initial emission)
    this.languageSubscription = toObservable(this.auth.language)
      .pipe(skip(1))
      .subscribe(() => {
        this.loadLibrary();
      });
  }

  ngOnDestroy(): void {
    this.languageSubscription?.unsubscribe();
  }

  ngOnInit(): void {
    this.loadLibrary();
  }

  loadLibrary(): void {
    this.loading.set(true);
    this.error.set(null);

    forkJoin({
      movies: this.api.listMovies(1, 100),
      series: this.api.listSeries(),
      recent: this.api.listRecent(),
      unmatched: this.api.listUnmatched(1, 100)
    }).subscribe({
      next: (responses) => {
        this.movies.set(responses.movies.data.items || []);
        this.series.set(responses.series.data.items || []);
        this.recentItems.set(responses.recent.data.items || []);
        this.unmatched.set(responses.unmatched.data.items || []);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Failed to load library');
        this.loading.set(false);
      }
    });
  }

  setActiveTab(tab: FilterTab): void {
    this.activeTab.set(tab);
  }

  getTabClass(tabId: FilterTab): string {
    const baseClass = 'px-4 py-2 font-medium text-sm transition border-b-2 -mb-px';
    if (this.activeTab() === tabId) {
      return `${baseClass} text-blue-400 border-blue-400`;
    }
    return `${baseClass} text-slate-400 border-transparent hover:text-white hover:border-slate-500`;
  }

  onMovieClick(item: MovieSummary | SeriesSummary): void {
    if ('media_id' in item) {
      this.navigateToMovie(item);
    }
  }

  onRecentItemClick(item: RecentlyAddedItem): void {
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
    this.router.navigate(['/movie', movie.media_id]);
  }

  navigateToSeries(s: SeriesSummary): void {
    this.router.navigate(['/series', s.series_metadata_id]);
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
      case 'pending': return 'Pending';
      case 'not_found': return 'Not Found';
      case 'failed': return 'Failed';
      case 'skipped': return 'Skipped';
      default: return status;
    }
  }
}
