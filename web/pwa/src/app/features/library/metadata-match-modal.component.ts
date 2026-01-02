import { Component, inject, Input, Output, EventEmitter, signal, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
  ApiService,
  MetadataCandidate,
  SearchResult,
  UnmatchedMediaSummary
} from '../../core/services/api.service';

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p/w185';

type ModalView = 'candidates' | 'search';
type MediaType = 'movie' | 'tv';

@Component({
  selector: 'app-metadata-match-modal',
  standalone: true,
  imports: [FormsModule],
  template: `
    @if (isOpen()) {
      <!-- Backdrop -->
      <div
        class="fixed inset-0 bg-black/70 z-40"
        (click)="close()"
      ></div>

      <!-- Modal -->
      <div class="fixed inset-4 md:inset-10 lg:inset-20 bg-slate-900 rounded-lg z-50 flex flex-col overflow-hidden">
        <!-- Header -->
        <div class="flex items-center justify-between p-4 border-b border-slate-700">
          <div>
            <h2 class="text-xl font-bold text-white">Fix Metadata Match</h2>
            @if (media) {
              <p class="text-sm text-slate-400 mt-1 truncate max-w-lg">{{ media.filename }}</p>
            }
          </div>
          <button
            (click)="close()"
            class="p-2 text-slate-400 hover:text-white transition"
          >
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <!-- Tab navigation -->
        <div class="flex border-b border-slate-700">
          <button
            (click)="setView('candidates')"
            [class]="getTabClass('candidates')"
          >
            Suggestions ({{ candidates().length }})
          </button>
          <button
            (click)="setView('search')"
            [class]="getTabClass('search')"
          >
            Search TMDB
          </button>
        </div>

        <!-- Content -->
        <div class="flex-1 overflow-y-auto p-4">
          @if (loading()) {
            <div class="flex justify-center items-center h-48">
              <div class="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-blue-500"></div>
            </div>
          } @else if (error()) {
            <div class="bg-red-500/10 border border-red-500 text-red-400 px-4 py-3 rounded">
              {{ error() }}
            </div>
          } @else {
            <!-- Candidates view -->
            @if (currentView() === 'candidates') {
              @if (candidates().length === 0) {
                <div class="text-center text-slate-400 py-12">
                  <p class="text-lg mb-2">No suggestions found</p>
                  <p class="text-sm">Try searching TMDB manually</p>
                </div>
              } @else {
                <div class="space-y-3">
                  @for (candidate of candidates(); track candidate.id) {
                    <div
                      class="flex items-center gap-4 p-3 bg-slate-800 rounded-lg hover:bg-slate-700 transition cursor-pointer"
                      (click)="selectCandidate(candidate)"
                    >
                      <!-- Poster -->
                      <div class="flex-shrink-0 w-16">
                        <div class="aspect-[2/3] rounded overflow-hidden bg-slate-700">
                          @if (candidate.poster_url) {
                            <img
                              [src]="candidate.poster_url"
                              [alt]="candidate.title"
                              class="w-full h-full object-cover"
                              loading="lazy"
                            />
                          } @else {
                            <div class="w-full h-full flex items-center justify-center">
                              <svg class="w-6 h-6 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                              </svg>
                            </div>
                          }
                        </div>
                      </div>

                      <!-- Info -->
                      <div class="flex-1 min-w-0">
                        <h3 class="text-white font-medium">{{ candidate.title }}</h3>
                        <div class="flex items-center gap-2 mt-1 text-sm text-slate-400">
                          <span class="px-2 py-0.5 bg-slate-600 rounded text-xs uppercase">
                            {{ candidate.media_type }}
                          </span>
                          @if (candidate.release_date) {
                            <span>{{ formatYear(candidate.release_date) }}</span>
                          }
                        </div>
                      </div>

                      <!-- Confidence -->
                      <div class="flex-shrink-0 text-right">
                        <div
                          class="text-lg font-bold"
                          [class]="getConfidenceClass(candidate.confidence)"
                        >
                          {{ candidate.confidence }}%
                        </div>
                        <div class="text-xs text-slate-500">match</div>
                      </div>
                    </div>
                  }
                </div>
              }
            }

            <!-- Search view -->
            @if (currentView() === 'search') {
              <!-- Search form -->
              <div class="flex gap-3 mb-6">
                <div class="flex-1">
                  <input
                    type="text"
                    [(ngModel)]="searchQuery"
                    (keydown.enter)="performSearch()"
                    placeholder="Search for a movie or TV show..."
                    class="w-full px-4 py-2 bg-slate-800 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <select
                  [(ngModel)]="searchType"
                  class="px-3 py-2 bg-slate-800 border border-slate-600 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="">All</option>
                  <option value="movie">Movies</option>
                  <option value="tv">TV Shows</option>
                </select>
                <button
                  (click)="performSearch()"
                  [disabled]="searching() || !searchQuery.trim()"
                  class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition disabled:opacity-50"
                >
                  @if (searching()) {
                    <div class="animate-spin w-5 h-5 border-2 border-white border-t-transparent rounded-full"></div>
                  } @else {
                    Search
                  }
                </button>
              </div>

              <!-- Search results -->
              @if (searchResults().length > 0) {
                <div class="space-y-3">
                  @for (result of searchResults(); track result.tmdb_id) {
                    <div
                      class="flex items-start gap-4 p-3 bg-slate-800 rounded-lg hover:bg-slate-700 transition cursor-pointer"
                      (click)="selectSearchResult(result)"
                    >
                      <!-- Poster -->
                      <div class="flex-shrink-0 w-20">
                        <div class="aspect-[2/3] rounded overflow-hidden bg-slate-700">
                          @if (result.poster_url) {
                            <img
                              [src]="getPosterUrl(result.poster_url)"
                              [alt]="result.title"
                              class="w-full h-full object-cover"
                              loading="lazy"
                            />
                          } @else {
                            <div class="w-full h-full flex items-center justify-center">
                              <svg class="w-8 h-8 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                              </svg>
                            </div>
                          }
                        </div>
                      </div>

                      <!-- Info -->
                      <div class="flex-1 min-w-0">
                        <h3 class="text-white font-medium">{{ result.title }}</h3>
                        @if (result.original_title && result.original_title !== result.title) {
                          <p class="text-sm text-slate-500">{{ result.original_title }}</p>
                        }
                        <div class="flex items-center gap-2 mt-1 text-sm text-slate-400">
                          <span class="px-2 py-0.5 bg-slate-600 rounded text-xs uppercase">
                            {{ result.media_type }}
                          </span>
                          @if (result.release_date) {
                            <span>{{ formatYear(result.release_date) }}</span>
                          }
                        </div>
                        @if (result.overview) {
                          <p class="mt-2 text-sm text-slate-400 line-clamp-2">
                            {{ result.overview }}
                          </p>
                        }
                      </div>
                    </div>
                  }
                </div>
              } @else if (hasSearched()) {
                <div class="text-center text-slate-400 py-12">
                  <p class="text-lg">No results found</p>
                  <p class="text-sm mt-1">Try a different search term</p>
                </div>
              }
            }
          }
        </div>

        <!-- Footer -->
        <div class="flex items-center justify-between p-4 border-t border-slate-700">
          <button
            (click)="skipMatch()"
            [disabled]="linking()"
            class="px-4 py-2 text-slate-400 hover:text-white transition disabled:opacity-50"
          >
            Skip Metadata
          </button>
          <button
            (click)="close()"
            class="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg transition"
          >
            Cancel
          </button>
        </div>
      </div>
    }
  `,
  styles: [`
    .line-clamp-2 {
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
    }
  `]
})
export class MetadataMatchModalComponent implements OnInit {
  private readonly api = inject(ApiService);

  @Input() media: UnmatchedMediaSummary | null = null;
  @Output() matched = new EventEmitter<void>();
  @Output() skipped = new EventEmitter<void>();
  @Output() closed = new EventEmitter<void>();

  isOpen = signal(false);
  currentView = signal<ModalView>('candidates');
  loading = signal(false);
  error = signal<string | null>(null);
  linking = signal(false);

  candidates = signal<MetadataCandidate[]>([]);
  searchResults = signal<SearchResult[]>([]);
  searching = signal(false);
  hasSearched = signal(false);

  searchQuery = '';
  searchType: MediaType | '' = '';

  ngOnInit(): void {
    // Load candidates when media is set
  }

  open(): void {
    this.isOpen.set(true);
    this.currentView.set('candidates');
    this.error.set(null);
    this.searchResults.set([]);
    this.hasSearched.set(false);
    this.searchQuery = this.media?.title || '';

    if (this.media) {
      this.loadCandidates();
    }
  }

  close(): void {
    this.isOpen.set(false);
    this.closed.emit();
  }

  setView(view: ModalView): void {
    this.currentView.set(view);
    this.error.set(null);
  }

  getTabClass(view: ModalView): string {
    const base = 'px-4 py-3 font-medium text-sm transition border-b-2 -mb-px';
    if (this.currentView() === view) {
      return `${base} text-blue-400 border-blue-400`;
    }
    return `${base} text-slate-400 border-transparent hover:text-white`;
  }

  loadCandidates(): void {
    if (!this.media) return;

    this.loading.set(true);
    this.error.set(null);

    this.api.getCandidates(this.media.media_id).subscribe({
      next: (response) => {
        this.candidates.set(response.data || []);
        this.loading.set(false);
      },
      error: (err) => {
        // 404 means no candidates, which is fine
        if (err.status === 404) {
          this.candidates.set([]);
        } else {
          this.error.set(err.error?.error?.message || 'Failed to load candidates');
        }
        this.loading.set(false);
      }
    });
  }

  performSearch(): void {
    if (!this.media || !this.searchQuery.trim()) return;

    this.searching.set(true);
    this.error.set(null);

    const request = {
      query: this.searchQuery.trim(),
      media_type: this.searchType || undefined,
      max_results: 10
    };

    this.api.searchMetadata(this.media.media_id, request).subscribe({
      next: (response) => {
        this.searchResults.set(response.data || []);
        this.hasSearched.set(true);
        this.searching.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Search failed');
        this.searching.set(false);
        this.hasSearched.set(true);
      }
    });
  }

  selectCandidate(candidate: MetadataCandidate): void {
    if (!this.media) return;

    this.linking.set(true);
    this.error.set(null);

    this.api.linkCandidate(this.media.media_id, candidate.id).subscribe({
      next: () => {
        this.linking.set(false);
        this.matched.emit();
        this.close();
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Failed to link metadata');
        this.linking.set(false);
      }
    });
  }

  selectSearchResult(result: SearchResult): void {
    if (!this.media) return;

    this.linking.set(true);
    this.error.set(null);

    const request = {
      tmdb_id: result.tmdb_id,
      media_type: result.media_type
    };

    this.api.linkSearchResult(this.media.media_id, request).subscribe({
      next: () => {
        this.linking.set(false);
        this.matched.emit();
        this.close();
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Failed to link metadata');
        this.linking.set(false);
      }
    });
  }

  skipMatch(): void {
    if (!this.media) return;

    this.linking.set(true);
    this.error.set(null);

    this.api.skipEnrichment(this.media.media_id).subscribe({
      next: () => {
        this.linking.set(false);
        this.skipped.emit();
        this.close();
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Failed to skip enrichment');
        this.linking.set(false);
      }
    });
  }

  formatYear(dateStr: string): string {
    if (!dateStr) return '';
    return dateStr.split('-')[0];
  }

  getConfidenceClass(confidence: number): string {
    if (confidence >= 80) return 'text-green-400';
    if (confidence >= 50) return 'text-yellow-400';
    return 'text-red-400';
  }

  getPosterUrl(posterPath: string): string {
    if (posterPath.startsWith('http')) {
      return posterPath;
    }
    return `${TMDB_IMAGE_BASE}${posterPath}`;
  }
}
