import { Component, inject, OnInit, OnDestroy, signal, computed, ViewChild } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { toObservable } from '@angular/core/rxjs-interop';
import { Subscription, skip } from 'rxjs';
import {
  ApiService,
  SeriesDetail,
  SeasonSummary,
  EpisodeSummary,
  SimilarSeriesItem,
  UnmatchedMediaSummary,
  SeriesCreditPerson
} from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';
import { MetadataMatchModalComponent } from './metadata-match-modal.component';

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p';

@Component({
  selector: 'app-series-detail',
  standalone: true,
  imports: [RouterLink, MetadataMatchModalComponent],
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
    } @else if (series()) {
      <!-- Backdrop hero section -->
      <div class="relative min-h-[40vh] md:min-h-[50vh]">
        <!-- Backdrop image -->
        @if (backdropUrl()) {
          <div
            class="absolute inset-0 bg-cover bg-top"
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
            <div class="hidden md:block flex-shrink-0 w-56 lg:w-64 xl:w-72">
              <div class="aspect-[2/3] rounded-lg overflow-hidden shadow-2xl bg-slate-800">
                @if (posterUrl()) {
                  <img
                    [src]="posterUrl()"
                    [alt]="series()!.name"
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
              <h1 class="text-2xl md:text-3xl lg:text-4xl font-bold text-white mb-2">
                {{ series()!.name }}
              </h1>

              <!-- Meta info row -->
              <div class="flex flex-wrap items-center gap-3 text-sm text-slate-300 mb-3">
                @if (series()!.year) {
                  <span>{{ series()!.year }}</span>
                }
                <span class="text-slate-500">•</span>
                <span>{{ series()!.number_of_seasons }} seasons</span>
                <span class="text-slate-500">•</span>
                <span>{{ series()!.available_episodes }} / {{ series()!.number_of_episodes }} episodes</span>
                @if (series()!.vote_average > 0) {
                  <span class="text-slate-500">•</span>
                  <div class="flex items-center gap-1">
                    <svg class="w-4 h-4 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                      <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                    </svg>
                    <span class="font-medium">{{ series()!.vote_average.toFixed(1) }}</span>
                  </div>
                }
              </div>

              <!-- Genres -->
              @if (series()!.genres) {
                <div class="flex flex-wrap gap-2">
                  @for (genre of parseGenres(series()!.genres); track genre) {
                    <span class="px-3 py-1 bg-slate-700/80 text-slate-300 rounded-full text-sm">
                      {{ genre }}
                    </span>
                  }
                </div>
              }
            </div>
          </div>
        </div>
      </div>

      <!-- Overview section -->
      @if (series()!.overview) {
        <div class="container mx-auto px-4 py-6">
          <p class="text-slate-300 leading-relaxed max-w-4xl">
            {{ series()!.overview }}
          </p>
        </div>
      }

      <!-- Cast preview section -->
      @if (castPreview().length > 0) {
        <div class="container mx-auto px-4 pb-6">
          <section class="mb-4">
            <div class="flex items-center justify-between mb-3">
              <h2 class="text-xl font-semibold text-white">Cast</h2>
              <a
                [routerLink]="['/series', series()!.series_metadata_id, 'cast']"
                class="text-blue-400 hover:text-blue-300 text-sm transition"
              >
                View all &rarr;
              </a>
            </div>
            <div class="flex gap-3 overflow-x-auto pb-2 -mx-4 px-4 scrollbar-thin scrollbar-thumb-slate-600 scrollbar-track-transparent">
              @for (person of castPreview(); track person.id) {
                <a
                  [href]="getTmdbPersonUrl(person.tmdb_person_id)"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="flex-shrink-0 w-28 group"
                >
                  <div class="aspect-[2/3] rounded-lg overflow-hidden bg-slate-700 group-hover:ring-2 group-hover:ring-blue-500 transition">
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
                  <p class="font-medium text-white text-sm mt-2 truncate group-hover:text-blue-400 transition">{{ person.name }}</p>
                  <p class="text-slate-400 text-xs truncate">
                    {{ parseRoles(person.roles) }}
                  </p>
                  <p class="text-slate-500 text-xs">
                    {{ person.total_episode_count }} {{ person.total_episode_count === 1 ? 'episode' : 'episodes' }}
                  </p>
                </a>
              }
            </div>
          </section>
        </div>
      }

      <!-- Season selector and episodes -->
      <div class="container mx-auto px-4 py-6">
        <!-- Admin action buttons -->
        @if (auth.isAdmin()) {
          <div class="flex items-center gap-3 mb-4">
            <button
              (click)="confirmDeleteSeries()"
              [disabled]="deleting()"
              class="flex items-center gap-2 px-3 py-2 bg-red-600 hover:bg-red-700 text-white text-sm rounded-lg transition disabled:opacity-50"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
              </svg>
              Delete Series
            </button>
          </div>
        }

        <!-- Season dropdown -->
        @if (seasons().length > 0) {
          <div class="flex items-center gap-4 mb-6">
            <label class="text-white font-medium">Season:</label>
            <select
              [value]="selectedSeasonNumber()"
              (change)="onSeasonChange($event)"
              class="bg-slate-800 text-white border border-slate-600 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              @for (season of seasons(); track season.season_metadata_id) {
                <option [value]="season.season_number">
                  {{ season.name || 'Season ' + season.season_number }}
                  ({{ season.episode_count }} episodes)
                </option>
              }
            </select>

            <!-- Delete Season button (admin only) -->
            @if (auth.isAdmin() && selectedSeason()) {
              <button
                (click)="confirmDeleteSeason()"
                [disabled]="deleting()"
                class="flex items-center gap-2 px-3 py-2 bg-red-600/80 hover:bg-red-600 text-white text-sm rounded-lg transition disabled:opacity-50"
                title="Delete all episodes in this season"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                </svg>
                Delete Season
              </button>
            }
          </div>

          <!-- Episode list -->
          @if (selectedSeason()) {
            <div class="space-y-3">
              @for (episode of selectedSeason()!.episodes || []; track episode.episode_metadata_id) {
                <div
                  class="flex items-center gap-4 p-4 bg-slate-800 rounded-lg hover:bg-slate-700 transition"
                  [class.opacity-50]="!episode.media_id"
                >
                  <!-- Episode still image -->
                  <div class="flex-shrink-0 w-32 md:w-40">
                    <div class="aspect-video rounded overflow-hidden bg-slate-700">
                      @if (episode.still_path) {
                        <img
                          [src]="getStillUrl(episode.still_path)"
                          [alt]="episode.name"
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

                  <!-- Episode info -->
                  <div class="flex-1 min-w-0">
                    <div class="flex items-start justify-between gap-4">
                      <div>
                        <h3 class="text-white font-medium">
                          {{ episode.episode_number }}. {{ episode.name }}
                        </h3>
                        <div class="flex items-center gap-2 mt-1 text-xs text-slate-400">
                          @if (episode.air_date) {
                            <span>{{ formatDate(episode.air_date) }}</span>
                          }
                          @if (episode.duration) {
                            <span class="text-slate-500">•</span>
                            <span>{{ formatDuration(episode.duration) }}</span>
                          }
                          @if (episode.vote_average > 0) {
                            <span class="text-slate-500">•</span>
                            <div class="flex items-center gap-1">
                              <svg class="w-3 h-3 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                                <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                              </svg>
                              <span>{{ episode.vote_average.toFixed(1) }}</span>
                            </div>
                          }
                        </div>
                        @if (episode.overview) {
                          <p class="mt-2 text-sm text-slate-400 line-clamp-2">
                            {{ episode.overview }}
                          </p>
                        }
                      </div>

                      <!-- Action buttons -->
                      <div class="flex-shrink-0 flex items-center gap-2">
                        <!-- Play button -->
                        @if (episode.media_id && episode.transcode_status === 'completed') {
                          <button
                            (click)="playEpisode(episode)"
                            class="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition"
                          >
                            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                              <path d="M8 5v14l11-7z"/>
                            </svg>
                            <span class="hidden sm:inline">Play</span>
                          </button>
                        } @else if (episode.media_id && episode.transcode_status === 'pending') {
                          <div class="px-3 py-2 text-yellow-500 text-sm">
                            <svg class="w-5 h-5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                            </svg>
                          </div>
                        } @else if (!episode.media_id) {
                          <div class="px-3 py-2 text-slate-500 text-sm">
                            Not available
                          </div>
                        }

                        <!-- Change Metadata button (only for episodes with media) -->
                        @if (episode.media_id) {
                          <button
                            (click)="openChangeMetadata(episode)"
                            [disabled]="resettingMetadata()"
                            class="p-2 bg-slate-700 hover:bg-slate-600 text-slate-300 hover:text-white rounded-lg transition disabled:opacity-50"
                            title="Change Metadata"
                          >
                            @if (resettingMetadata() && selectedEpisode()?.media_id === episode.media_id) {
                              <div class="animate-spin w-4 h-4 border-2 border-white border-t-transparent rounded-full"></div>
                            } @else {
                              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                              </svg>
                            }
                          </button>
                        }

                        <!-- Delete Episode button (admin only, only for episodes with media) -->
                        @if (episode.media_id && auth.isAdmin()) {
                          <button
                            (click)="confirmDeleteEpisode(episode)"
                            [disabled]="deleting()"
                            class="p-2 bg-red-600/80 hover:bg-red-600 text-white rounded-lg transition disabled:opacity-50"
                            title="Delete Episode"
                          >
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                            </svg>
                          </button>
                        }
                      </div>
                    </div>
                  </div>
                </div>
              }

              @if (!selectedSeason()!.episodes || selectedSeason()!.episodes!.length === 0) {
                <div class="text-center text-slate-400 py-8">
                  No episodes found for this season
                </div>
              }
            </div>
          }
        } @else {
          <div class="text-center text-slate-400 py-8">
            No seasons available
          </div>
        }

        <!-- Similar Series -->
        @if (series()!.similar_series && series()!.similar_series!.length > 0) {
          <section class="mt-8">
            <h2 class="text-xl font-semibold text-white mb-3">Similar Series</h2>
            <div class="flex gap-5 overflow-x-auto pb-2 -mx-4 px-4 scrollbar-thin scrollbar-thumb-slate-600 scrollbar-track-transparent">
              @for (similar of series()!.similar_series; track similar.tmdb_id) {
                @if (similar.in_library && similar.series_metadata_id) {
                  <a
                    [routerLink]="['/series', similar.series_metadata_id]"
                    class="flex-shrink-0 w-28 group cursor-pointer"
                  >
                    <div class="aspect-[2/3] rounded-lg overflow-hidden bg-slate-700 ring-2 ring-blue-500">
                      @if (similar.poster_path) {
                        <img
                          [src]="getSimilarPosterUrl(similar.poster_path)"
                          [alt]="similar.name"
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
                    <p class="font-medium text-white text-sm mt-2 truncate group-hover:text-blue-400 transition-colors">{{ similar.name }}</p>
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
                    [href]="getTmdbSeriesUrl(similar.tmdb_id)"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="flex-shrink-0 w-28 opacity-60 hover:opacity-90 transition-opacity group cursor-pointer"
                  >
                    <div class="aspect-[2/3] rounded-lg overflow-hidden bg-slate-700">
                      @if (similar.poster_path) {
                        <img
                          [src]="getSimilarPosterUrl(similar.poster_path)"
                          [alt]="similar.name"
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
                    <p class="font-medium text-white text-sm mt-2 truncate group-hover:text-slate-200 transition-colors">{{ similar.name }}</p>
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
      </div>
    }

    <!-- Metadata Match Modal -->
    <app-metadata-match-modal
      #matchModal
      [media]="selectedMedia()"
      (matched)="onMetadataChanged()"
      (skipped)="onMetadataChanged()"
    />

    <!-- Delete Confirmation Modal -->
    @if (deleteType()) {
      <div class="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4">
        <div class="bg-slate-800 rounded-lg max-w-md w-full p-6">
          <h3 class="text-xl font-bold text-white mb-4">
            @switch (deleteType()) {
              @case ('episode') { Delete Episode }
              @case ('season') { Delete Season }
              @case ('series') { Delete Series }
            }
          </h3>
          <p class="text-slate-300 mb-2">
            Are you sure you want to delete <strong>{{ deleteTarget()?.name }}</strong>?
          </p>
          <p class="text-slate-400 text-sm mb-4">
            @switch (deleteType()) {
              @case ('episode') {
                The source file will be moved to trash. Transcoded files will be permanently deleted.
              }
              @case ('season') {
                All episode media files in this season will be deleted. Source files will be moved to trash.
              }
              @case ('series') {
                All episode media files in this series will be deleted. Source files will be moved to trash.
              }
            }
          </p>
          
          @if (deleteError()) {
            <div class="bg-red-500/10 border border-red-500 text-red-400 px-3 py-2 rounded text-sm mb-4">
              {{ deleteError() }}
            </div>
          }
          
          <div class="flex justify-end gap-3">
            <button
              (click)="cancelDelete()"
              [disabled]="deleting()"
              class="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg transition disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              (click)="executeDelete()"
              [disabled]="deleting()"
              class="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition disabled:opacity-50"
            >
              @if (deleting()) {
                <div class="animate-spin w-4 h-4 border-2 border-white border-t-transparent rounded-full"></div>
              }
              Delete
            </button>
          </div>
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
