import {
  Component,
  inject,
  Input,
  Output,
  EventEmitter,
  signal,
  OnInit,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
  ApiService,
  MetadataCandidate,
  SearchResult,
  UnmatchedMediaSummary,
} from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p/w185';

type ModalView = 'candidates' | 'search';
type MediaType = 'movie' | 'tv';

@Component({
  selector: 'app-metadata-match-modal',
  standalone: true,
  imports: [FormsModule],
  templateUrl: './metadata-match-modal.component.html',
  styles: [
    `
      .line-clamp-2 {
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
      }
    `,
  ],
})
export class MetadataMatchModalComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);

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
        const candidateList = response.data.candidates || [];
        this.candidates.set(candidateList);
        this.loading.set(false);

        // Auto-select tab based on candidates availability
        if (candidateList.length > 0) {
          this.currentView.set('candidates');
        } else {
          this.currentView.set('search');
        }
      },
      error: (err) => {
        // 404 means no candidates, which is fine
        if (err.status === 404) {
          this.candidates.set([]);
          this.currentView.set('search');
        } else {
          this.error.set(
            err.error?.error?.message || 'Failed to load candidates',
          );
        }
        this.loading.set(false);
      },
    });
  }

  performSearch(): void {
    if (!this.media || !this.searchQuery.trim()) return;

    this.searching.set(true);
    this.error.set(null);

    const request = {
      query: this.searchQuery.trim(),
      media_type: this.searchType || undefined,
      max_results: 10,
      language: this.auth.language(),
    };

    this.api.searchMetadata(this.media.media_id, request).subscribe({
      next: (response) => {
        this.searchResults.set(response.data?.results || []);
        this.hasSearched.set(true);
        this.searching.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Search failed');
        this.searching.set(false);
        this.hasSearched.set(true);
      },
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
      },
    });
  }

  selectSearchResult(result: SearchResult): void {
    if (!this.media) return;

    this.linking.set(true);
    this.error.set(null);

    if ((result.media_type = 'tv')) {
      alert('TV Show detected');
      return;
    }

    // const request = {
    //   tmdb_id: result.tmdb_id,
    //   media_type: result.media_type,
    // };
    //
    // this.api.linkSearchResult(this.media.media_id, request).subscribe({
    //   next: () => {
    //     this.linking.set(false);
    //     this.matched.emit();
    //     this.close();
    //   },
    //   error: (err) => {
    //     this.error.set(err.error?.error?.message || 'Failed to link metadata');
    //     this.linking.set(false);
    //   },
    // });
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
        this.error.set(
          err.error?.error?.message || 'Failed to skip enrichment',
        );
        this.linking.set(false);
      },
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
