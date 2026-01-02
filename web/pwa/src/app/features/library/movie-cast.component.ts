import { Component, inject, OnInit, signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { ApiService, MovieDetail, CreditPerson } from '../../core/services/api.service';

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p';

@Component({
  selector: 'app-movie-cast',
  standalone: true,
  template: `
    @if (loading()) {
      <div class="flex justify-center items-center h-screen">
        <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    } @else if (error()) {
      <div class="container mx-auto px-4 py-8">
        <div class="bg-red-500/10 border border-red-500 text-red-400 px-4 py-3 rounded">
          {{ error() }}
        </div>
        <button
          (click)="goBack()"
          class="mt-4 text-blue-400 hover:text-blue-300 transition"
        >
          &larr; Back
        </button>
      </div>
    } @else if (movie()) {
      <!-- Backdrop hero section -->
      <div class="relative min-h-[30vh]">
        @if (backdropUrl()) {
          <div
            class="absolute inset-0 bg-cover bg-center"
            [style.background-image]="'url(' + backdropUrl() + ')'"
          ></div>
        }
        <div class="absolute inset-0 bg-gradient-to-t from-slate-900 via-slate-900/80 to-slate-900/30"></div>

        <!-- Back button -->
        <div class="absolute top-4 left-4 z-10">
          <button
            (click)="goBack()"
            class="flex items-center gap-2 px-3 py-2 bg-black/50 hover:bg-black/70 text-white rounded-lg transition"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
            </svg>
            <span class="text-sm">Back</span>
          </button>
        </div>

        <!-- Title overlay -->
        <div class="absolute bottom-0 left-0 right-0 p-4 md:p-8">
          <div class="container mx-auto">
            <h1 class="text-2xl md:text-3xl font-bold text-white">Cast & Crew</h1>
            <p class="text-slate-300">{{ movie()!.title }} ({{ movie()!.year }})</p>
          </div>
        </div>
      </div>

      <!-- Main content -->
      <div class="container mx-auto px-4 py-8">
        <!-- Directors -->
        @if (movie()!.directors && movie()!.directors!.length > 0) {
          <section class="mb-10">
            <h2 class="text-xl font-semibold text-white mb-4 border-b border-slate-700 pb-2">
              {{ movie()!.directors!.length > 1 ? 'Directors' : 'Director' }}
            </h2>
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
              @for (person of movie()!.directors; track person.id) {
                <div class="bg-slate-800 rounded-lg overflow-hidden">
                  <div class="aspect-[2/3] bg-slate-700">
                    @if (person.profile_path) {
                      <img
                        [src]="getProfileUrl(person.profile_path)"
                        [alt]="person.name"
                        class="w-full h-full object-cover"
                      />
                    } @else {
                      <div class="w-full h-full flex items-center justify-center text-slate-500">
                        <svg class="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                        </svg>
                      </div>
                    }
                  </div>
                  <div class="p-3">
                    <p class="font-medium text-white text-sm">{{ person.name }}</p>
                    <p class="text-slate-400 text-xs">Director</p>
                  </div>
                </div>
              }
            </div>
          </section>
        }

        <!-- Cast -->
        @if (movie()!.cast && movie()!.cast!.length > 0) {
          <section class="mb-10">
            <h2 class="text-xl font-semibold text-white mb-4 border-b border-slate-700 pb-2">Cast</h2>
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
              @for (person of movie()!.cast; track person.id) {
                <div class="bg-slate-800 rounded-lg overflow-hidden">
                  <div class="aspect-[2/3] bg-slate-700">
                    @if (person.profile_path) {
                      <img
                        [src]="getProfileUrl(person.profile_path)"
                        [alt]="person.name"
                        class="w-full h-full object-cover"
                      />
                    } @else {
                      <div class="w-full h-full flex items-center justify-center text-slate-500">
                        <svg class="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                        </svg>
                      </div>
                    }
                  </div>
                  <div class="p-3">
                    <p class="font-medium text-white text-sm">{{ person.name }}</p>
                    @if (person.character) {
                      <p class="text-slate-400 text-xs truncate">{{ person.character }}</p>
                    }
                  </div>
                </div>
              }
            </div>
          </section>
        }

        <!-- Crew -->
        @if (movie()!.crew && movie()!.crew!.length > 0) {
          <section class="mb-10">
            <h2 class="text-xl font-semibold text-white mb-4 border-b border-slate-700 pb-2">Crew</h2>
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
              @for (person of movie()!.crew; track person.id) {
                <div class="bg-slate-800 rounded-lg overflow-hidden">
                  <div class="aspect-[2/3] bg-slate-700">
                    @if (person.profile_path) {
                      <img
                        [src]="getProfileUrl(person.profile_path)"
                        [alt]="person.name"
                        class="w-full h-full object-cover"
                      />
                    } @else {
                      <div class="w-full h-full flex items-center justify-center text-slate-500">
                        <svg class="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                        </svg>
                      </div>
                    }
                  </div>
                  <div class="p-3">
                    <p class="font-medium text-white text-sm">{{ person.name }}</p>
                    @if (person.job) {
                      <p class="text-slate-400 text-xs truncate">{{ person.job }}</p>
                    }
                  </div>
                </div>
              }
            </div>
          </section>
        }
      </div>
    }
  `
})
export class MovieCastComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly api = inject(ApiService);

  movie = signal<MovieDetail | null>(null);
  loading = signal(true);
  error = signal<string | null>(null);
  backdropUrl = signal<string | null>(null);

  ngOnInit(): void {
    const mediaId = this.route.snapshot.paramMap.get('id');
    if (!mediaId) {
      this.error.set('Invalid movie ID');
      this.loading.set(false);
      return;
    }

    this.loadMovie(mediaId);
  }

  loadMovie(mediaId: string): void {
    this.loading.set(true);
    this.error.set(null);

    this.api.getMovie(mediaId).subscribe({
      next: (response) => {
        const movieData = response.data;
        this.movie.set(movieData);
        if (movieData.backdrop_path) {
          this.backdropUrl.set(`${TMDB_IMAGE_BASE}/w1280${movieData.backdrop_path}`);
        }
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Failed to load movie');
        this.loading.set(false);
      }
    });
  }

  goBack(): void {
    const m = this.movie();
    if (m) {
      this.router.navigate(['/movie', m.media_id]);
    } else {
      this.router.navigate(['/library']);
    }
  }

  getProfileUrl(profilePath: string): string {
    return `${TMDB_IMAGE_BASE}/w185${profilePath}`;
  }
}
