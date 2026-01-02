import { Component, inject, OnInit, OnDestroy, signal } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { toObservable } from '@angular/core/rxjs-interop';
import { Subscription, skip } from 'rxjs';
import { ApiService, MovieDetail, CreditPerson, SimilarMovieItem, CollectionMovieItem, MovieCollectionInfo } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p';

@Component({
  selector: 'app-movie-detail',
  standalone: true,
  imports: [RouterLink],
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
          &larr; Back to Library
        </button>
      </div>
    } @else if (movie()) {
      <!-- Backdrop hero section -->
      <div class="relative min-h-[50vh] md:min-h-[60vh]">
        <!-- Backdrop image -->
        @if (backdropUrl()) {
          <div
            class="absolute inset-0 bg-cover bg-center"
            [style.background-image]="'url(' + backdropUrl() + ')'"
          ></div>
        }
        <!-- Gradient overlay -->
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

        <!-- Content overlay -->
        <div class="absolute bottom-0 left-0 right-0 p-4 md:p-8">
          <div class="container mx-auto flex gap-6 items-end">
            <!-- Poster -->
            <div class="hidden md:block flex-shrink-0 w-48 lg:w-56">
              <div class="aspect-[2/3] rounded-lg overflow-hidden shadow-2xl bg-slate-800">
                @if (posterUrl()) {
                  <img
                    [src]="posterUrl()"
                    [alt]="movie()!.title"
                    class="w-full h-full object-cover"
                  />
                } @else {
                  <div class="w-full h-full flex items-center justify-center">
                    <svg class="w-16 h-16 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                    </svg>
                  </div>
                }
              </div>
            </div>

            <!-- Info -->
            <div class="flex-1 min-w-0">
              <h1 class="text-3xl md:text-4xl lg:text-5xl font-bold text-white mb-3">
                {{ movie()!.title }}
              </h1>

              <!-- Meta info row -->
              <div class="flex flex-wrap items-center gap-3 text-sm md:text-base text-slate-300 mb-4">
                @if (movie()!.certification) {
                  <span class="px-2 py-0.5 border border-slate-400 rounded text-xs font-medium">{{ movie()!.certification }}</span>
                }
                @if (movie()!.year) {
                  <span>{{ movie()!.year }}</span>
                }
                @if (movie()!.runtime) {
                  <span class="text-slate-500">•</span>
                  <span>{{ formatRuntime(movie()!.runtime!) }}</span>
                } @else if (movie()!.duration) {
                  <span class="text-slate-500">•</span>
                  <span>{{ formatDuration(movie()!.duration) }}</span>
                }
                @if (movie()!.resolution) {
                  <span class="text-slate-500">•</span>
                  <span class="px-2 py-0.5 bg-slate-700 rounded text-xs font-medium">{{ movie()!.resolution }}</span>
                }
                @if (movie()!.vote_average > 0) {
                  <span class="text-slate-500">•</span>
                  <div class="flex items-center gap-1">
                    <svg class="w-4 h-4 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                      <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                    </svg>
                    <span class="font-medium">{{ movie()!.vote_average.toFixed(1) }}</span>
                  </div>
                }
              </div>

              <!-- Genres -->
              @if (movie()!.genres) {
                <div class="flex flex-wrap gap-2 mb-4">
                  @for (genre of parseGenres(movie()!.genres); track genre) {
                    <span class="px-3 py-1 bg-slate-700/80 text-slate-300 rounded-full text-sm">
                      {{ genre }}
                    </span>
                  }
                </div>
              }

              <!-- Tagline -->
              @if (movie()!.tagline) {
                <p class="text-slate-400 italic mb-4">"{{ movie()!.tagline }}"</p>
              }

              <!-- Play button -->
              @if (canPlay()) {
                <button
                  (click)="playMovie()"
                  class="flex items-center gap-3 px-6 py-3 bg-blue-600 hover:bg-blue-700 text-white font-semibold rounded-lg transition shadow-lg"
                >
                  <svg class="w-6 h-6" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M8 5v14l11-7z"/>
                  </svg>
                  Play
                </button>
              } @else {
                <div class="flex items-center gap-2 text-yellow-500">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
                  </svg>
                  <span>Transcoding in progress...</span>
                </div>
              }
            </div>
          </div>
        </div>
      </div>

      <!-- Main content section -->
      <div class="container mx-auto px-4 py-8">
        <!-- Overview -->
        @if (movie()!.overview) {
          <section class="mb-8">
            <h2 class="text-xl font-semibold text-white mb-3">Overview</h2>
            <p class="text-slate-300 leading-relaxed max-w-4xl">{{ movie()!.overview }}</p>
          </section>
        }

        <!-- Directors -->
        @if (movie()!.directors && movie()!.directors!.length > 0) {
          <section class="mb-8">
            <h2 class="text-xl font-semibold text-white mb-3">
              {{ movie()!.directors!.length > 1 ? 'Directors' : 'Director' }}
            </h2>
            <div class="flex flex-wrap gap-4">
              @for (director of movie()!.directors; track director.id) {
                <div class="flex items-center gap-3">
                  <div class="w-12 h-12 rounded-full bg-slate-700 overflow-hidden flex-shrink-0">
                    @if (director.profile_path) {
                      <img
                        [src]="getProfileUrl(director.profile_path)"
                        [alt]="director.name"
                        class="w-full h-full object-cover"
                      />
                    } @else {
                      <div class="w-full h-full flex items-center justify-center text-slate-500">
                        <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                        </svg>
                      </div>
                    }
                  </div>
                  <span class="text-slate-300">{{ director.name }}</span>
                </div>
              }
            </div>
          </section>
        }

        <!-- Cast -->
        @if (movie()!.cast && movie()!.cast!.length > 0) {
          <section class="mb-8">
            <div class="flex items-center justify-between mb-3">
              <h2 class="text-xl font-semibold text-white">Cast</h2>
              <a
                [routerLink]="['/movie', movie()!.media_id, 'cast']"
                class="text-blue-400 hover:text-blue-300 text-sm transition"
              >
                View all &rarr;
              </a>
            </div>
            <div class="flex gap-3 overflow-x-auto pb-2 -mx-4 px-4 scrollbar-thin scrollbar-thumb-slate-600 scrollbar-track-transparent">
              @for (person of movie()!.cast; track person.id) {
                <div class="flex-shrink-0 w-28">
                  <div class="aspect-[2/3] rounded-lg overflow-hidden bg-slate-700">
                    @if (person.profile_path) {
                      <img
                        [src]="getProfileUrl(person.profile_path)"
                        [alt]="person.name"
                        class="w-full h-full object-cover"
                      />
                    } @else {
                      <div class="w-full h-full flex items-center justify-center text-slate-500">
                        <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                        </svg>
                      </div>
                    }
                  </div>
                  <p class="font-medium text-white text-sm mt-2 truncate">{{ person.name }}</p>
                  @if (person.character) {
                    <p class="text-slate-400 text-xs truncate">{{ person.character }}</p>
                  }
                </div>
              }
            </div>
          </section>
        }

        <!-- Crew -->
        @if (movie()!.crew && movie()!.crew!.length > 0) {
          <section class="mb-8">
            <h2 class="text-xl font-semibold text-white mb-3">Crew</h2>
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
              @for (person of movie()!.crew; track person.id) {
                <div class="flex items-center gap-3 bg-slate-800 rounded-lg p-3">
                  <div class="w-10 h-10 rounded-full bg-slate-700 overflow-hidden flex-shrink-0">
                    @if (person.profile_path) {
                      <img
                        [src]="getProfileUrl(person.profile_path)"
                        [alt]="person.name"
                        class="w-full h-full object-cover"
                      />
                    } @else {
                      <div class="w-full h-full flex items-center justify-center text-slate-500">
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                        </svg>
                      </div>
                    }
                  </div>
                  <div class="min-w-0">
                    <p class="font-medium text-white text-sm truncate">{{ person.name }}</p>
                    @if (person.job) {
                      <p class="text-slate-400 text-xs truncate">{{ person.job }}</p>
                    }
                  </div>
                </div>
              }
            </div>
          </section>
        }

        <!-- Movie Collection -->
        @if (movie()!.collection && movie()!.collection!.movies.length > 0) {
          <section class="mb-8">
            <h2 class="text-xl font-semibold text-white mb-3">
              Part of the {{ movie()!.collection!.name }} ({{ movie()!.collection!.position }} of {{ movie()!.collection!.total_movies }})
            </h2>
            <div class="flex gap-5 overflow-x-auto pb-2 -mx-4 px-4 scrollbar-thin scrollbar-thumb-slate-600 scrollbar-track-transparent">
              @for (colMovie of movie()!.collection!.movies; track colMovie.tmdb_id) {
                @if (colMovie.is_current) {
                  <!-- Current movie - yellow ring with badge -->
                  <div class="flex-shrink-0 w-28">
                    <div class="relative aspect-[2/3] rounded-lg overflow-hidden bg-slate-700 ring-2 ring-yellow-400">
                      @if (colMovie.poster_path) {
                        <img
                          [src]="getCollectionPosterUrl(colMovie.poster_path)"
                          [alt]="colMovie.title"
                          class="w-full h-full object-cover"
                        />
                      } @else {
                        <div class="w-full h-full flex items-center justify-center text-slate-500">
                          <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                          </svg>
                        </div>
                      }
                      <div class="absolute top-1 right-1 px-1.5 py-0.5 bg-yellow-400 text-slate-900 text-xs font-bold rounded">
                        Current
                      </div>
                    </div>
                    <p class="font-medium text-white text-sm mt-2 truncate">{{ colMovie.title }}</p>
                    <div class="flex items-center gap-1 text-xs text-slate-400">
                      @if (colMovie.year) {
                        <span>{{ colMovie.year }}</span>
                      }
                      @if (colMovie.vote_average > 0) {
                        <span class="text-slate-500">•</span>
                        <svg class="w-3 h-3 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                          <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                        </svg>
                        <span>{{ colMovie.vote_average.toFixed(1) }}</span>
                      }
                    </div>
                  </div>
                } @else if (colMovie.in_library && colMovie.media_id) {
                  <!-- In library - blue ring, clickable -->
                  <a
                    [routerLink]="['/movie', colMovie.media_id]"
                    class="flex-shrink-0 w-28 group cursor-pointer"
                  >
                    <div class="aspect-[2/3] rounded-lg overflow-hidden bg-slate-700 ring-2 ring-blue-500">
                      @if (colMovie.poster_path) {
                        <img
                          [src]="getCollectionPosterUrl(colMovie.poster_path)"
                          [alt]="colMovie.title"
                          class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200"
                        />
                      } @else {
                        <div class="w-full h-full flex items-center justify-center text-slate-500">
                          <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                          </svg>
                        </div>
                      }
                    </div>
                    <p class="font-medium text-white text-sm mt-2 truncate group-hover:text-blue-400 transition-colors">{{ colMovie.title }}</p>
                    <div class="flex items-center gap-1 text-xs text-slate-400">
                      @if (colMovie.year) {
                        <span>{{ colMovie.year }}</span>
                      }
                      @if (colMovie.vote_average > 0) {
                        <span class="text-slate-500">•</span>
                        <svg class="w-3 h-3 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                          <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                        </svg>
                        <span>{{ colMovie.vote_average.toFixed(1) }}</span>
                      }
                    </div>
                    <span class="text-xs text-blue-400">In Library</span>
                  </a>
                } @else {
                  <!-- Not in library - link to TMDB -->
                  <a
                    [href]="getTmdbMovieUrl(colMovie.tmdb_id)"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="flex-shrink-0 w-28 opacity-60 hover:opacity-90 transition-opacity group cursor-pointer"
                  >
                    <div class="aspect-[2/3] rounded-lg overflow-hidden bg-slate-700">
                      @if (colMovie.poster_path) {
                        <img
                          [src]="getCollectionPosterUrl(colMovie.poster_path)"
                          [alt]="colMovie.title"
                          class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200"
                        />
                      } @else {
                        <div class="w-full h-full flex items-center justify-center text-slate-500">
                          <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                          </svg>
                        </div>
                      }
                    </div>
                    <p class="font-medium text-white text-sm mt-2 truncate group-hover:text-slate-200 transition-colors">{{ colMovie.title }}</p>
                    <div class="flex items-center gap-1 text-xs text-slate-400">
                      @if (colMovie.year) {
                        <span>{{ colMovie.year }}</span>
                      }
                      @if (colMovie.vote_average > 0) {
                        <span class="text-slate-500">•</span>
                        <svg class="w-3 h-3 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                          <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                        </svg>
                        <span>{{ colMovie.vote_average.toFixed(1) }}</span>
                      }
                    </div>
                    <span class="text-xs text-slate-500 flex items-center gap-1">
                      TMDB
                      <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
                      </svg>
                    </span>
                  </a>
                }
              }
            </div>
          </section>
        }

        <!-- Similar Movies -->
        @if (movie()!.similar_movies && movie()!.similar_movies!.length > 0) {
          <section class="mb-8">
            <h2 class="text-xl font-semibold text-white mb-3">Similar Movies</h2>
            <div class="flex gap-5 overflow-x-auto pb-2 -mx-4 px-4 scrollbar-thin scrollbar-thumb-slate-600 scrollbar-track-transparent">
              @for (similar of movie()!.similar_movies; track similar.tmdb_id) {
                @if (similar.in_library && similar.media_id) {
                  <a
                    [routerLink]="['/movie', similar.media_id]"
                    class="flex-shrink-0 w-28 group cursor-pointer"
                  >
                    <div class="aspect-[2/3] rounded-lg overflow-hidden bg-slate-700 ring-2 ring-blue-500">
                      @if (similar.poster_path) {
                        <img
                          [src]="getSimilarPosterUrl(similar.poster_path)"
                          [alt]="similar.title"
                          class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200"
                        />
                      } @else {
                        <div class="w-full h-full flex items-center justify-center text-slate-500">
                          <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                          </svg>
                        </div>
                      }
                    </div>
                    <p class="font-medium text-white text-sm mt-2 truncate group-hover:text-blue-400 transition-colors">{{ similar.title }}</p>
                    <div class="flex items-center gap-1 text-xs text-slate-400">
                      @if (similar.year) {
                        <span>{{ similar.year }}</span>
                      }
                      @if (similar.vote_average > 0) {
                        <span class="text-slate-500">•</span>
                        <svg class="w-3 h-3 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                          <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                        </svg>
                        <span>{{ similar.vote_average.toFixed(1) }}</span>
                      }
                    </div>
                    <span class="text-xs text-blue-400">In Library</span>
                  </a>
                } @else {
                  <!-- Not in library - link to TMDB -->
                  <a
                    [href]="getTmdbMovieUrl(similar.tmdb_id)"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="flex-shrink-0 w-28 opacity-60 hover:opacity-90 transition-opacity group cursor-pointer"
                  >
                    <div class="aspect-[2/3] rounded-lg overflow-hidden bg-slate-700">
                      @if (similar.poster_path) {
                        <img
                          [src]="getSimilarPosterUrl(similar.poster_path)"
                          [alt]="similar.title"
                          class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-200"
                        />
                      } @else {
                        <div class="w-full h-full flex items-center justify-center text-slate-500">
                          <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                          </svg>
                        </div>
                      }
                    </div>
                    <p class="font-medium text-white text-sm mt-2 truncate group-hover:text-slate-200 transition-colors">{{ similar.title }}</p>
                    <div class="flex items-center gap-1 text-xs text-slate-400">
                      @if (similar.year) {
                        <span>{{ similar.year }}</span>
                      }
                      @if (similar.vote_average > 0) {
                        <span class="text-slate-500">•</span>
                        <svg class="w-3 h-3 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                          <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                        </svg>
                        <span>{{ similar.vote_average.toFixed(1) }}</span>
                      }
                    </div>
                    <span class="text-xs text-slate-500 flex items-center gap-1">
                      TMDB
                      <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/>
                      </svg>
                    </span>
                  </a>
                }
              }
            </div>
          </section>
        }

        <!-- Status indicators -->
        <div class="flex flex-wrap gap-4 text-sm">
          @if (movie()!.transcode_status === 'pending') {
            <div class="flex items-center gap-2 px-3 py-2 bg-yellow-500/20 text-yellow-400 rounded-lg">
              <svg class="w-4 h-4 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
              <span>Transcoding pending</span>
            </div>
          }
          @if (movie()!.enrichment_status !== 'matched') {
            <div class="flex items-center gap-2 px-3 py-2 bg-orange-500/20 text-orange-400 rounded-lg">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
              </svg>
              <span>{{ formatEnrichmentStatus(movie()!.enrichment_status) }}</span>
            </div>
          }
        </div>
      </div>
    }
  `
})
export class MovieDetailComponent implements OnInit, OnDestroy {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);

  private mediaId: string | null = null;
  private languageSubscription: Subscription | null = null;

  movie = signal<MovieDetail | null>(null);
  loading = signal(true);
  error = signal<string | null>(null);

  posterUrl = signal<string | null>(null);
  backdropUrl = signal<string | null>(null);

  constructor() {
    // Re-fetch movie details when language changes (skip initial emission)
    this.languageSubscription = toObservable(this.auth.language)
      .pipe(skip(1))
      .subscribe(() => {
        if (this.mediaId) {
          this.loadMovie(this.mediaId);
        }
      });
  }

  ngOnDestroy(): void {
    this.languageSubscription?.unsubscribe();
  }

  ngOnInit(): void {
    this.mediaId = this.route.snapshot.paramMap.get('id');
    if (!this.mediaId) {
      this.error.set('Invalid movie ID');
      this.loading.set(false);
      return;
    }

    this.loadMovie(this.mediaId);
  }

  loadMovie(mediaId: string): void {
    this.loading.set(true);
    this.error.set(null);

    this.api.getMovie(mediaId).subscribe({
      next: (response) => {
        const movieData = response.data;
        this.movie.set(movieData);

        if (movieData.poster_path) {
          this.posterUrl.set(`${TMDB_IMAGE_BASE}/w500${movieData.poster_path}`);
        }
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

  canPlay(): boolean {
    const m = this.movie();
    return m !== null && m.transcode_status === 'completed';
  }

  playMovie(): void {
    const m = this.movie();
    if (m) {
      this.router.navigate(['/play', m.media_id]);
    }
  }

  goBack(): void {
    this.router.navigate(['/library']);
  }

  formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }

  formatRuntime(minutes: number): string {
    const hours = Math.floor(minutes / 60);
    const mins = minutes % 60;
    if (hours > 0) {
      return `${hours}h ${mins}m`;
    }
    return `${mins}m`;
  }

  parseGenres(genres: string | undefined): string[] {
    if (!genres) return [];
    return genres.split(',').map(g => g.trim()).filter(g => g.length > 0);
  }

  getProfileUrl(profilePath: string): string {
    return `${TMDB_IMAGE_BASE}/w185${profilePath}`;
  }

  getSimilarPosterUrl(posterPath: string): string {
    return `${TMDB_IMAGE_BASE}/w185${posterPath}`;
  }

  getCollectionPosterUrl(posterPath: string): string {
    return `${TMDB_IMAGE_BASE}/w185${posterPath}`;
  }

  getTmdbMovieUrl(tmdbId: number): string {
    return `https://www.themoviedb.org/movie/${tmdbId}?language=${this.auth.language()}`;
  }

  formatEnrichmentStatus(status: string): string {
    switch (status) {
      case 'pending': return 'Metadata pending';
      case 'not_found': return 'No metadata match found';
      case 'failed': return 'Metadata lookup failed';
      case 'skipped': return 'Metadata skipped';
      default: return status;
    }
  }
}
