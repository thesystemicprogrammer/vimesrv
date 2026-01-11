import { Component, inject, OnInit, OnDestroy, signal, computed, ViewChild } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { toObservable } from '@angular/core/rxjs-interop';
import { Subscription, skip } from 'rxjs';
import {
  ApiService,
  SeriesDetail,
  EpisodeSummary,
  UnmatchedMediaSummary,
  SeriesCreditPerson
} from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';
import { MetadataMatchModalComponent } from './metadata-match-modal.component';
import { FavoriteButtonComponent } from '../../shared/components/favorite-button.component';
import { DecodeUnicodePipe } from "../../shared/pipes/decode-unicode.pipe";

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p';

@Component({
  selector: 'app-series-detail',
  standalone: true,
  imports: [RouterLink, MetadataMatchModalComponent, FavoriteButtonComponent, DecodeUnicodePipe],
  templateUrl: './series-detail.component.html',
  styles: [`
    .line-clamp-2 {
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
      overflow: hidden;
    }
  `]
})
export class SeriesDetailComponent implements OnInit, OnDestroy {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly api = inject(ApiService);
  readonly auth = inject(AuthService);

  private seriesId: number | null = null;
  private languageSubscription: Subscription | null = null;

  @ViewChild('matchModal') matchModal!: MetadataMatchModalComponent;

  series = signal<SeriesDetail | null>(null);
  loading = signal(true);
  error = signal<string | null>(null);
  selectedSeasonNumber = signal<number>(1);

  posterUrl = signal<string | null>(null);
  backdropUrl = signal<string | null>(null);

  // Cast data
  castPreview = signal<SeriesCreditPerson[]>([]);

  // Metadata change state
  selectedMedia = signal<UnmatchedMediaSummary | null>(null);
  selectedEpisode = signal<EpisodeSummary | null>(null);
  resettingMetadata = signal(false);

  // Delete state
  deleteType = signal<'episode' | 'season' | 'series' | null>(null);
  deleteTarget = signal<{ name: string; id: string | number } | null>(null);
  deleting = signal(false);
  deleteError = signal<string | null>(null);

  seasons = computed(() => {
    const s = this.series();
    return s?.seasons || [];
  });

  selectedSeason = computed(() => {
    const seasonNum = this.selectedSeasonNumber();
    return this.seasons().find(s => s.season_number === seasonNum) || null;
  });


  constructor() {
    // Re-fetch series details when language changes (skip initial emission)
    this.languageSubscription = toObservable(this.auth.language)
      .pipe(skip(1))
      .subscribe(() => {
        if (this.seriesId) {
          // Preserve current season selection when re-fetching
          const currentSeason = this.selectedSeasonNumber();
          this.loadSeries(this.seriesId, currentSeason);
        }
      });
  }

  ngOnDestroy(): void {
    this.languageSubscription?.unsubscribe();
  }

  ngOnInit(): void {
    const seriesIdParam = this.route.snapshot.paramMap.get('id');
    if (!seriesIdParam || isNaN(Number(seriesIdParam))) {
      this.error.set('Invalid series ID');
      this.loading.set(false);
      return;
    }
    this.seriesId = Number(seriesIdParam);

    // Read optional season query param (e.g., /series/1?season=2)
    const seasonParam = this.route.snapshot.queryParamMap.get('season');
    const requestedSeason = seasonParam && !isNaN(Number(seasonParam)) ? Number(seasonParam) : null;

    this.loadSeries(this.seriesId, requestedSeason);
  }

  loadSeries(seriesId: number, requestedSeason: number | null = null): void {
    this.loading.set(true);
    this.error.set(null);

    this.api.getSeriesDetail(seriesId).subscribe({
      next: (response) => {
        const seriesData = response.data;
        this.series.set(seriesData);

        if (seriesData.poster_path) {
          this.posterUrl.set(`${TMDB_IMAGE_BASE}/w500${seriesData.poster_path}`);
        }
        if (seriesData.backdrop_path) {
          this.backdropUrl.set(`${TMDB_IMAGE_BASE}/w1280${seriesData.backdrop_path}`);
        }

        // Select requested season from query param, or fall back to first available
        if (seriesData.seasons && seriesData.seasons.length > 0) {
          const seasonExists = requestedSeason !== null &&
            seriesData.seasons.some(s => s.season_number === requestedSeason);

          if (seasonExists) {
            this.selectedSeasonNumber.set(requestedSeason!);
          } else {
            this.selectedSeasonNumber.set(seriesData.seasons[0].season_number);
          }
        }

        // Fetch series credits for cast preview (non-blocking)
        this.api.getSeriesCredits(seriesId).subscribe({
          next: (creditsResponse) => {
            // Take top 10 cast members for preview
            const cast = creditsResponse.data.cast || [];
            this.castPreview.set(cast.slice(0, 10));
          },
          error: () => {
            // Silently fail - cast section just won't show
            this.castPreview.set([]);
          }
        });

        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Failed to load series');
        this.loading.set(false);
      }
    });
  }

  onSeasonChange(event: Event): void {
    const target = event.target as HTMLSelectElement;
    this.selectedSeasonNumber.set(Number(target.value));
  }

  playEpisode(episode: EpisodeSummary): void {
    if (episode.media_id) {
      this.router.navigate(['/play', episode.media_id]);
    }
  }

  goBack(): void {
    this.router.navigate(['/library']);
  }

  getStillUrl(stillPath: string): string {
    return `${TMDB_IMAGE_BASE}/w300${stillPath}`;
  }

  formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }

  formatDate(dateStr: string): string {
    try {
      const date = new Date(dateStr);
      return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
    } catch {
      return dateStr;
    }
  }

  parseGenres(genres: string | undefined): string[] {
    if (!genres) return [];
    return genres.split(',').map(g => g.trim()).filter(g => g.length > 0);
  }

  getSimilarPosterUrl(posterPath: string): string {
    return `${TMDB_IMAGE_BASE}/w185${posterPath}`;
  }

  getTmdbSeriesUrl(tmdbId: number): string {
    return `https://www.themoviedb.org/tv/${tmdbId}?language=${this.auth.language()}`;
  }

  getTmdbPersonUrl(tmdbPersonId: number): string {
    return `https://www.themoviedb.org/person/${tmdbPersonId}`;
  }

  getProfileUrl(profilePath: string): string {
    return `${TMDB_IMAGE_BASE}/w185${profilePath}`;
  }

  parseRoles(rolesJson: string | undefined): string {
    if (!rolesJson) return '';
    try {
      const roles = JSON.parse(rolesJson) as Array<{ character: string; episode_count: number }>;
      if (roles.length === 0) return '';
      // Return the first character name
      return roles[0].character || '';
    } catch {
      return '';
    }
  }

  openChangeMetadata(episode: EpisodeSummary): void {
    if (!episode.media_id) return;

    this.selectedEpisode.set(episode);
    this.resettingMetadata.set(true);

    // First reset the enrichment status, then open the modal
    this.api.resetEnrichment(episode.media_id).subscribe({
      next: () => {
        // Create an UnmatchedMediaSummary from the episode data
        const s = this.series();
        const mediaSummary: UnmatchedMediaSummary = {
          media_id: episode.media_id!,
          filename: `${s?.name || 'Episode'} - S${episode.season_number}E${episode.episode_number}`,
          title: episode.name,
          duration: episode.duration,
          resolution: '',
          enrichment_status: 'pending',
          created_at: ''
        };

        this.selectedMedia.set(mediaSummary);
        this.resettingMetadata.set(false);

        // Open the modal
        setTimeout(() => {
          this.matchModal.open();
        }, 0);
      },
      error: (err) => {
        console.error('Failed to reset enrichment:', err);
        this.resettingMetadata.set(false);
        this.selectedEpisode.set(null);
        this.error.set(err.error?.error?.message || 'Failed to reset metadata');
      }
    });
  }

  onMetadataChanged(): void {
    // Reload the series to reflect the new metadata
    if (this.seriesId) {
      const currentSeason = this.selectedSeasonNumber();
      this.loadSeries(this.seriesId, currentSeason);
    }
    this.selectedEpisode.set(null);
  }

  // Delete confirmation methods
  confirmDeleteEpisode(episode: EpisodeSummary): void {
    if (!episode.media_id) return;

    const s = this.series();
    this.deleteType.set('episode');
    this.deleteTarget.set({
      name: `S${episode.season_number}E${episode.episode_number}: ${episode.name}`,
      id: episode.media_id
    });
    this.deleteError.set(null);
  }

  confirmDeleteSeason(): void {
    const season = this.selectedSeason();
    if (!season) return;

    this.deleteType.set('season');
    this.deleteTarget.set({
      name: season.name || `Season ${season.season_number}`,
      id: season.season_metadata_id
    });
    this.deleteError.set(null);
  }

  confirmDeleteSeries(): void {
    const s = this.series();
    if (!s) return;

    this.deleteType.set('series');
    this.deleteTarget.set({
      name: s.name,
      id: s.series_metadata_id
    });
    this.deleteError.set(null);
  }

  cancelDelete(): void {
    this.deleteType.set(null);
    this.deleteTarget.set(null);
    this.deleteError.set(null);
  }

  executeDelete(): void {
    const type = this.deleteType();
    const target = this.deleteTarget();
    if (!type || !target) return;

    this.deleting.set(true);
    this.deleteError.set(null);

    let deleteObs;
    switch (type) {
      case 'episode':
        deleteObs = this.api.deleteMedia(target.id as string);
        break;
      case 'season':
        deleteObs = this.api.deleteSeason(target.id as number);
        break;
      case 'series':
        deleteObs = this.api.deleteSeries(target.id as number);
        break;
    }

    deleteObs.subscribe({
      next: () => {
        this.deleting.set(false);
        this.deleteType.set(null);
        this.deleteTarget.set(null);

        if (type === 'series') {
          // Navigate back to library after deleting series
          this.router.navigate(['/library']);
        } else {
          // Reload the series to reflect changes
          if (this.seriesId) {
            const currentSeason = this.selectedSeasonNumber();
            this.loadSeries(this.seriesId, currentSeason);
          }
        }
      },
      error: (err) => {
        this.deleting.set(false);
        const message = err.error?.error?.message || 'Failed to delete';
        // Check for running jobs error (409 Conflict)
        if (err.status === 409) {
          this.deleteError.set('Cannot delete: transcoding is in progress. Please wait for it to complete.');
        } else {
          this.deleteError.set(message);
        }
      }
    });
  }
}
