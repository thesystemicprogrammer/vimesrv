import { Component, inject, signal, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { TranslateModule, TranslateService } from '@ngx-translate/core';
import { Subject, debounceTime, distinctUntilChanged, takeUntil } from 'rxjs';
import {
  ApiService,
  MediaTranscodingDetails,
  MediaSearchResult,
  QualityProfileInfo,
  TranscodeInfo,
  AudioStreamInfo,
  SubtitleStreamInfo,
} from '../../core/services/api.service';

type TabType = 'video' | 'audio' | 'subtitle';
type ModalMode = 'add-video' | 'add-audio' | 'add-subtitle' | 'recreate' | 'delete' | null;

@Component({
  selector: 'app-transcoding-admin',
  standalone: true,
  imports: [CommonModule, FormsModule, TranslateModule],
  template: `
    <div class="min-h-screen bg-slate-900 py-8 px-4">
      <div class="max-w-6xl mx-auto">
        <!-- Header -->
        <div class="flex items-center justify-between mb-8">
          <div>
            <h1 class="text-3xl font-bold text-white">{{ 'transcodingAdmin.title' | translate }}</h1>
            <p class="text-slate-400 mt-1">{{ 'transcodingAdmin.subtitle' | translate }}</p>
          </div>
          <button
            (click)="goBack()"
            class="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
          >
            {{ 'common.back' | translate }}
          </button>
        </div>

        <!-- Toast Messages -->
        @if (error()) {
          <div class="mb-6 bg-red-500/10 border border-red-500 text-red-400 px-4 py-3 rounded-lg flex items-center justify-between">
            <span>{{ error() }}</span>
            <button (click)="error.set(null)" class="text-red-400 hover:text-red-300">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>
        }

        @if (success()) {
          <div class="mb-6 bg-green-500/10 border border-green-500 text-green-400 px-4 py-3 rounded-lg flex items-center justify-between">
            <span>{{ success() }}</span>
            <button (click)="success.set(null)" class="text-green-400 hover:text-green-300">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>
        }

        <!-- Search Section -->
        <div class="bg-slate-800 rounded-lg border border-slate-700 p-6 mb-6">
          <label class="block text-sm font-medium text-slate-300 mb-2">{{ 'transcodingAdmin.searchMedia' | translate }}</label>
          <div class="relative">
            <input
              type="text"
              [(ngModel)]="searchQuery"
              (ngModelChange)="onSearchChange($event)"
              class="w-full px-4 py-3 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
              [placeholder]="'transcodingAdmin.searchPlaceholder' | translate"
            />
            @if (searching()) {
              <div class="absolute right-3 top-1/2 -translate-y-1/2">
                <svg class="animate-spin h-5 w-5 text-blue-500" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
              </div>
            }
          </div>

          <!-- Search Results -->
          @if (searchResults().length > 0 && !selectedMedia()) {
            <div class="mt-3 bg-slate-700 rounded-lg border border-slate-600 max-h-64 overflow-y-auto">
              @for (result of searchResults(); track result.id) {
                <button
                  (click)="selectMedia(result)"
                  class="w-full px-4 py-3 text-left hover:bg-slate-600 transition-colors border-b border-slate-600 last:border-b-0"
                >
                  <div class="text-white font-medium">{{ result.title }}</div>
                  <div class="text-slate-400 text-sm flex gap-4">
                    <span>{{ result.filename }}</span>
                    <span class="text-blue-400">{{ result.resolution }}</span>
                  </div>
                </button>
              }
            </div>
          }
        </div>

        <!-- Selected Media -->
        @if (selectedMedia()) {
          <div class="bg-slate-800 rounded-lg border border-slate-700 overflow-hidden">
            <!-- Media Header -->
            <div class="p-6 border-b border-slate-700">
              <div class="flex items-start justify-between">
                <div>
                  <h2 class="text-xl font-bold text-white">{{ mediaDetails()?.title }}</h2>
                  <div class="text-slate-400 text-sm mt-1 flex gap-4">
                    <span>{{ mediaDetails()?.filename }}</span>
                    <span class="text-blue-400">{{ mediaDetails()?.resolution }}</span>
                  </div>
                </div>
                <button
                  (click)="clearSelection()"
                  class="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
                >
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                  </svg>
                </button>
              </div>
            </div>

            <!-- Loading State -->
            @if (loadingDetails()) {
              <div class="flex justify-center py-12">
                <svg class="animate-spin h-8 w-8 text-blue-500" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
              </div>
            } @else if (mediaDetails()) {
              <!-- Tabs -->
              <div class="border-b border-slate-700">
                <div class="flex">
                  <button
                    (click)="activeTab.set('video')"
                    class="px-6 py-3 text-sm font-medium transition-colors"
                    [class]="activeTab() === 'video' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-slate-400 hover:text-white'"
                  >
                    {{ 'transcodingAdmin.videoTab' | translate }}
                    <span class="ml-2 px-2 py-0.5 text-xs rounded-full bg-slate-700">{{ mediaDetails()?.video_transcodes?.length || 0 }}</span>
                  </button>
                  <button
                    (click)="activeTab.set('audio')"
                    class="px-6 py-3 text-sm font-medium transition-colors"
                    [class]="activeTab() === 'audio' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-slate-400 hover:text-white'"
                  >
                    {{ 'transcodingAdmin.audioTab' | translate }}
                    <span class="ml-2 px-2 py-0.5 text-xs rounded-full bg-slate-700">{{ mediaDetails()?.audio_transcodes?.length || 0 }}</span>
                  </button>
                  <button
                    (click)="activeTab.set('subtitle')"
                    class="px-6 py-3 text-sm font-medium transition-colors"
                    [class]="activeTab() === 'subtitle' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-slate-400 hover:text-white'"
                  >
                    {{ 'transcodingAdmin.subtitleTab' | translate }}
                    <span class="ml-2 px-2 py-0.5 text-xs rounded-full bg-slate-700">{{ mediaDetails()?.subtitle_transcodes?.length || 0 }}</span>
                  </button>
                </div>
              </div>

              <!-- Tab Content -->
              <div class="p-6">
                <!-- Video Tab -->
                @if (activeTab() === 'video') {
                  <div class="space-y-4">
                    <!-- Add Video Button -->
                    <div class="flex justify-end">
                      <button
                        (click)="openAddVideoModal()"
                        class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors flex items-center gap-2"
                      >
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/>
                        </svg>
                        {{ 'transcodingAdmin.addTranscoding' | translate }}
                      </button>
                    </div>

                    <!-- Video Transcodes List -->
                    @if (mediaDetails()?.video_transcodes?.length) {
                      <div class="bg-slate-700 rounded-lg border border-slate-600 overflow-hidden">
                        <table class="w-full">
                          <thead class="bg-slate-600/50">
                            <tr>
                              <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'transcodingAdmin.quality' | translate }}</th>
                              <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'transcodingAdmin.status' | translate }}</th>
                              <th class="px-4 py-3 text-right text-xs font-medium text-slate-400 uppercase">{{ 'transcodingAdmin.actions' | translate }}</th>
                            </tr>
                          </thead>
                          <tbody class="divide-y divide-slate-600">
                            @for (transcode of mediaDetails()?.video_transcodes; track transcode.id) {
                              <tr>
                                <td class="px-4 py-3 text-white font-medium">{{ transcode.quality }}</td>
                                <td class="px-4 py-3">
                                  <span [class]="getStatusClass(transcode.status)">{{ transcode.status }}</span>
                                </td>
                                <td class="px-4 py-3 text-right">
                                  <div class="flex justify-end gap-2">
                                    <button
                                      (click)="openRecreateModal(transcode)"
                                      [disabled]="isProcessing(transcode.status)"
                                      class="p-1.5 text-slate-400 hover:text-yellow-400 hover:bg-slate-600 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                      [title]="'transcodingAdmin.recreate' | translate"
                                    >
                                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                                      </svg>
                                    </button>
                                    <button
                                      (click)="openDeleteModal(transcode)"
                                      [disabled]="isProcessing(transcode.status)"
                                      class="p-1.5 text-slate-400 hover:text-red-400 hover:bg-slate-600 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                      [title]="'transcodingAdmin.delete' | translate"
                                    >
                                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                                      </svg>
                                    </button>
                                  </div>
                                </td>
                              </tr>
                            }
                          </tbody>
                        </table>
                      </div>
                    } @else {
                      <div class="text-center py-8 text-slate-400">
                        {{ 'transcodingAdmin.noVideoTranscodes' | translate }}
                      </div>
                    }
                  </div>
                }

                <!-- Audio Tab -->
                @if (activeTab() === 'audio') {
                  <div class="space-y-4">
                    <!-- Add Audio Button -->
                    <div class="flex justify-end">
                      <button
                        (click)="openAddAudioModal()"
                        [disabled]="!getAvailableAudioStreams().length"
                        class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/>
                        </svg>
                        {{ 'transcodingAdmin.addTranscoding' | translate }}
                      </button>
                    </div>

                    <!-- Audio Transcodes List -->
                    @if (mediaDetails()?.audio_transcodes?.length) {
                      <div class="bg-slate-700 rounded-lg border border-slate-600 overflow-hidden">
                        <table class="w-full">
                          <thead class="bg-slate-600/50">
                            <tr>
                              <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'transcodingAdmin.track' | translate }}</th>
                              <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'transcodingAdmin.language' | translate }}</th>
                              <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'transcodingAdmin.channels' | translate }}</th>
                              <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'transcodingAdmin.status' | translate }}</th>
                              <th class="px-4 py-3 text-right text-xs font-medium text-slate-400 uppercase">{{ 'transcodingAdmin.actions' | translate }}</th>
                            </tr>
                          </thead>
                          <tbody class="divide-y divide-slate-600">
                            @for (transcode of mediaDetails()?.audio_transcodes; track transcode.id) {
                              <tr>
                                <td class="px-4 py-3 text-white">{{ transcode.track_index }}</td>
                                <td class="px-4 py-3 text-white">{{ transcode.language || '-' }}</td>
                                <td class="px-4 py-3 text-white">{{ transcode.channels || '-' }}</td>
                                <td class="px-4 py-3">
                                  <span [class]="getStatusClass(transcode.status)">{{ transcode.status }}</span>
                                </td>
                                <td class="px-4 py-3 text-right">
                                  <div class="flex justify-end gap-2">
                                    <button
                                      (click)="openRecreateModal(transcode)"
                                      [disabled]="isProcessing(transcode.status)"
                                      class="p-1.5 text-slate-400 hover:text-yellow-400 hover:bg-slate-600 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                      [title]="'transcodingAdmin.recreate' | translate"
                                    >
                                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                                      </svg>
                                    </button>
                                    <button
                                      (click)="openDeleteModal(transcode)"
                                      [disabled]="isProcessing(transcode.status)"
                                      class="p-1.5 text-slate-400 hover:text-red-400 hover:bg-slate-600 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                      [title]="'transcodingAdmin.delete' | translate"
                                    >
                                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                                      </svg>
                                    </button>
                                  </div>
                                </td>
                              </tr>
                            }
                          </tbody>
                        </table>
                      </div>
                    } @else {
                      <div class="text-center py-8 text-slate-400">
                        {{ 'transcodingAdmin.noAudioTranscodes' | translate }}
                      </div>
                    }
                  </div>
                }

                <!-- Subtitle Tab -->
                @if (activeTab() === 'subtitle') {
                  <div class="space-y-4">
                    <!-- Add Subtitle Button -->
                    <div class="flex justify-end">
                      <button
                        (click)="openAddSubtitleModal()"
                        [disabled]="!getAvailableSubtitleStreams().length"
                        class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/>
                        </svg>
                        {{ 'transcodingAdmin.addTranscoding' | translate }}
                      </button>
                    </div>

                    <!-- Subtitle Transcodes List -->
                    @if (mediaDetails()?.subtitle_transcodes?.length) {
                      <div class="bg-slate-700 rounded-lg border border-slate-600 overflow-hidden">
                        <table class="w-full">
                          <thead class="bg-slate-600/50">
                            <tr>
                              <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'transcodingAdmin.track' | translate }}</th>
                              <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'transcodingAdmin.language' | translate }}</th>
                              <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'transcodingAdmin.status' | translate }}</th>
                              <th class="px-4 py-3 text-right text-xs font-medium text-slate-400 uppercase">{{ 'transcodingAdmin.actions' | translate }}</th>
                            </tr>
                          </thead>
                          <tbody class="divide-y divide-slate-600">
                            @for (transcode of mediaDetails()?.subtitle_transcodes; track transcode.id) {
                              <tr>
                                <td class="px-4 py-3 text-white">{{ transcode.track_index }}</td>
                                <td class="px-4 py-3 text-white">{{ transcode.language || '-' }}</td>
                                <td class="px-4 py-3">
                                  <span [class]="getStatusClass(transcode.status)">{{ transcode.status }}</span>
                                </td>
                                <td class="px-4 py-3 text-right">
                                  <div class="flex justify-end gap-2">
                                    <button
                                      (click)="openRecreateModal(transcode)"
                                      [disabled]="isProcessing(transcode.status)"
                                      class="p-1.5 text-slate-400 hover:text-yellow-400 hover:bg-slate-600 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                      [title]="'transcodingAdmin.recreate' | translate"
                                    >
                                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                                      </svg>
                                    </button>
                                    <button
                                      (click)="openDeleteModal(transcode)"
                                      [disabled]="isProcessing(transcode.status)"
                                      class="p-1.5 text-slate-400 hover:text-red-400 hover:bg-slate-600 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                                      [title]="'transcodingAdmin.delete' | translate"
                                    >
                                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                                      </svg>
                                    </button>
                                  </div>
                                </td>
                              </tr>
                            }
                          </tbody>
                        </table>
                      </div>
                    } @else {
                      <div class="text-center py-8 text-slate-400">
                        {{ 'transcodingAdmin.noSubtitleTranscodes' | translate }}
                      </div>
                    }
                  </div>
                }
              </div>
            }
          </div>
        }
      </div>
    </div>

    <!-- Modal Backdrop -->
    @if (modalMode()) {
      <div 
        class="fixed inset-0 bg-black/50 z-40 flex items-center justify-center p-4"
        (click)="closeModal()"
      >
        <div 
          class="bg-slate-800 rounded-lg border border-slate-700 w-full max-w-md shadow-xl"
          (click)="$event.stopPropagation()"
        >
          <!-- Add Video Modal -->
          @if (modalMode() === 'add-video') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">{{ 'transcodingAdmin.addVideoTranscoding' | translate }}</h2>
              <form (ngSubmit)="addVideoTranscoding()">
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-2">{{ 'transcodingAdmin.selectQuality' | translate }}</label>
                  <div class="space-y-2 max-h-64 overflow-y-auto">
                    @for (profile of getAvailableQualityProfiles(); track profile.name) {
                      <label
                        class="flex items-center p-3 bg-slate-700 rounded-lg cursor-pointer hover:bg-slate-600 transition-colors"
                        [class.opacity-50]="!profile.is_qualified"
                      >
                        <input
                          type="radio"
                          [value]="profile.name"
                          [(ngModel)]="selectedQuality"
                          name="quality"
                          class="w-4 h-4 text-blue-600 bg-slate-600 border-slate-500 focus:ring-blue-500"
                        />
                        <div class="ml-3 flex-1">
                          <span class="text-white font-medium">{{ profile.name }}</span>
                          <span class="text-slate-400 text-sm ml-2">({{ profile.resolution }})</span>
                        </div>
                        @if (!profile.is_qualified) {
                          <span class="text-yellow-400 text-xs">{{ 'transcodingAdmin.exceedsSource' | translate }}</span>
                        }
                      </label>
                    }
                  </div>
                </div>
                <div class="flex gap-3 mt-6">
                  <button
                    type="button"
                    (click)="closeModal()"
                    class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                  >
                    {{ 'common.cancel' | translate }}
                  </button>
                  <button
                    type="submit"
                    [disabled]="modalLoading() || !selectedQuality"
                    class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50"
                  >
                    @if (modalLoading()) {
                      {{ 'common.processing' | translate }}
                    } @else {
                      {{ 'transcodingAdmin.addTranscoding' | translate }}
                    }
                  </button>
                </div>
              </form>
            </div>
          }

          <!-- Add Audio Modal -->
          @if (modalMode() === 'add-audio') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">{{ 'transcodingAdmin.addAudioTranscoding' | translate }}</h2>
              <form (ngSubmit)="addAudioTranscoding()">
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-2">{{ 'transcodingAdmin.selectAudioStream' | translate }}</label>
                  <div class="space-y-2 max-h-64 overflow-y-auto">
                    @for (stream of getAvailableAudioStreams(); track stream.index) {
                      <label class="flex items-center p-3 bg-slate-700 rounded-lg cursor-pointer hover:bg-slate-600 transition-colors">
                        <input
                          type="radio"
                          [value]="stream.index"
                          [(ngModel)]="selectedTrackIndex"
                          name="track"
                          class="w-4 h-4 text-blue-600 bg-slate-600 border-slate-500 focus:ring-blue-500"
                        />
                        <div class="ml-3">
                          <span class="text-white font-medium">{{ 'transcodingAdmin.track' | translate }} {{ stream.index }}</span>
                          <span class="text-slate-400 text-sm ml-2">{{ stream.language || 'Unknown' }}</span>
                          <span class="text-slate-500 text-sm ml-2">({{ stream.channels }}ch)</span>
                        </div>
                      </label>
                    }
                  </div>
                </div>
                <div class="flex gap-3 mt-6">
                  <button
                    type="button"
                    (click)="closeModal()"
                    class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                  >
                    {{ 'common.cancel' | translate }}
                  </button>
                  <button
                    type="submit"
                    [disabled]="modalLoading() || selectedTrackIndex === null"
                    class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50"
                  >
                    @if (modalLoading()) {
                      {{ 'common.processing' | translate }}
                    } @else {
                      {{ 'transcodingAdmin.addTranscoding' | translate }}
                    }
                  </button>
                </div>
              </form>
            </div>
          }

          <!-- Add Subtitle Modal -->
          @if (modalMode() === 'add-subtitle') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">{{ 'transcodingAdmin.addSubtitleTranscoding' | translate }}</h2>
              <form (ngSubmit)="addSubtitleTranscoding()">
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-2">{{ 'transcodingAdmin.selectSubtitleStream' | translate }}</label>
                  <div class="space-y-2 max-h-64 overflow-y-auto">
                    @for (stream of getAvailableSubtitleStreams(); track stream.index) {
                      <label class="flex items-center p-3 bg-slate-700 rounded-lg cursor-pointer hover:bg-slate-600 transition-colors">
                        <input
                          type="radio"
                          [value]="stream.index"
                          [(ngModel)]="selectedTrackIndex"
                          name="track"
                          class="w-4 h-4 text-blue-600 bg-slate-600 border-slate-500 focus:ring-blue-500"
                        />
                        <div class="ml-3">
                          <span class="text-white font-medium">{{ 'transcodingAdmin.track' | translate }} {{ stream.index }}</span>
                          <span class="text-slate-400 text-sm ml-2">{{ stream.language || 'Unknown' }}</span>
                        </div>
                      </label>
                    }
                  </div>
                </div>
                <div class="flex gap-3 mt-6">
                  <button
                    type="button"
                    (click)="closeModal()"
                    class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                  >
                    {{ 'common.cancel' | translate }}
                  </button>
                  <button
                    type="submit"
                    [disabled]="modalLoading() || selectedTrackIndex === null"
                    class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50"
                  >
                    @if (modalLoading()) {
                      {{ 'common.processing' | translate }}
                    } @else {
                      {{ 'transcodingAdmin.addTranscoding' | translate }}
                    }
                  </button>
                </div>
              </form>
            </div>
          }

          <!-- Recreate Modal -->
          @if (modalMode() === 'recreate') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">{{ 'transcodingAdmin.recreateTranscoding' | translate }}</h2>
              <p class="text-slate-400 mb-4">{{ 'transcodingAdmin.recreateConfirm' | translate }}</p>
              <p class="text-yellow-400 text-sm mb-6">{{ 'transcodingAdmin.recreateWarning' | translate }}</p>
              <div class="flex gap-3">
                <button
                  (click)="closeModal()"
                  class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                >
                  {{ 'common.cancel' | translate }}
                </button>
                <button
                  (click)="recreateTranscoding()"
                  [disabled]="modalLoading()"
                  class="flex-1 px-4 py-2 bg-yellow-600 text-white rounded-md hover:bg-yellow-700 transition-colors disabled:opacity-50"
                >
                  @if (modalLoading()) {
                    {{ 'common.processing' | translate }}
                  } @else {
                    {{ 'transcodingAdmin.recreate' | translate }}
                  }
                </button>
              </div>
            </div>
          }

          <!-- Delete Modal -->
          @if (modalMode() === 'delete') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">{{ 'transcodingAdmin.deleteTranscoding' | translate }}</h2>
              <p class="text-slate-400 mb-4">{{ 'transcodingAdmin.deleteConfirm' | translate }}</p>
              <p class="text-red-400 text-sm mb-6">{{ 'transcodingAdmin.deleteWarning' | translate }}</p>
              <div class="flex gap-3">
                <button
                  (click)="closeModal()"
                  class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                >
                  {{ 'common.cancel' | translate }}
                </button>
                <button
                  (click)="deleteTranscoding()"
                  [disabled]="modalLoading()"
                  class="flex-1 px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 transition-colors disabled:opacity-50"
                >
                  @if (modalLoading()) {
                    {{ 'common.processing' | translate }}
                  } @else {
                    {{ 'transcodingAdmin.delete' | translate }}
                  }
                </button>
              </div>
            </div>
          }
        </div>
      </div>
    }
  `
})
export class TranscodingAdminComponent implements OnInit, OnDestroy {
  private readonly api = inject(ApiService);
  private readonly router = inject(Router);
  private readonly translate = inject(TranslateService);
  private readonly destroy$ = new Subject<void>();

  // Search state
  searchQuery = '';
  searchResults = signal<MediaSearchResult[]>([]);
  searching = signal(false);
  private searchSubject = new Subject<string>();

  // Selected media state
  selectedMedia = signal<MediaSearchResult | null>(null);
  mediaDetails = signal<MediaTranscodingDetails | null>(null);
  loadingDetails = signal(false);

  // Tab state
  activeTab = signal<TabType>('video');

  // Toast messages
  error = signal<string | null>(null);
  success = signal<string | null>(null);

  // Modal state
  modalMode = signal<ModalMode>(null);
  modalLoading = signal(false);
  selectedTranscode = signal<TranscodeInfo | null>(null);
  selectedQuality = '';
  selectedTrackIndex: number | null = null;

  ngOnInit(): void {
    // Setup debounced search
    this.searchSubject.pipe(
      debounceTime(300),
      distinctUntilChanged(),
      takeUntil(this.destroy$)
    ).subscribe(query => {
      this.performSearch(query);
    });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  goBack(): void {
    this.router.navigate(['/']);
  }

  onSearchChange(query: string): void {
    if (query.length < 2) {
      this.searchResults.set([]);
      return;
    }
    this.searchSubject.next(query);
  }

  private performSearch(query: string): void {
    if (!query || query.length < 2) {
      this.searchResults.set([]);
      return;
    }

    this.searching.set(true);
    this.api.searchMediaForTranscodings(query).subscribe({
      next: (response) => {
        this.searchResults.set(response.data.results);
        this.searching.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
        this.searching.set(false);
      }
    });
  }

  selectMedia(media: MediaSearchResult): void {
    this.selectedMedia.set(media);
    this.searchResults.set([]);
    this.searchQuery = '';
    this.loadMediaDetails(media.id);
  }

  clearSelection(): void {
    this.selectedMedia.set(null);
    this.mediaDetails.set(null);
    this.activeTab.set('video');
  }

  private loadMediaDetails(mediaId: string): void {
    this.loadingDetails.set(true);
    this.api.getMediaTranscodings(mediaId).subscribe({
      next: (response) => {
        this.mediaDetails.set(response.data);
        this.loadingDetails.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
        this.loadingDetails.set(false);
      }
    });
  }

  getStatusClass(status: string): string {
    switch (status) {
      case 'completed':
        return 'text-green-400';
      case 'processing':
        return 'text-blue-400';
      case 'pending':
        return 'text-yellow-400';
      case 'failed':
        return 'text-red-400';
      default:
        return 'text-slate-400';
    }
  }

  isProcessing(status: string): boolean {
    return status === 'processing';
  }

  // Get available quality profiles (not already transcoded)
  getAvailableQualityProfiles(): QualityProfileInfo[] {
    const details = this.mediaDetails();
    if (!details) return [];

    const existingQualities = new Set(details.video_transcodes?.map(t => t.quality) || []);
    return details.qualified_quality_profiles?.filter(p => !existingQualities.has(p.name)) || [];
  }

  // Get available audio streams (not already transcoded)
  getAvailableAudioStreams(): AudioStreamInfo[] {
    const details = this.mediaDetails();
    if (!details) return [];

    const existingIndices = new Set(details.audio_transcodes?.map(t => t.track_index) || []);
    return details.available_audio_streams?.filter(s => !existingIndices.has(s.index)) || [];
  }

  // Get available subtitle streams (not already transcoded)
  getAvailableSubtitleStreams(): SubtitleStreamInfo[] {
    const details = this.mediaDetails();
    if (!details) return [];

    const existingIndices = new Set(details.subtitle_transcodes?.map(t => t.track_index) || []);
    return details.available_subtitle_streams?.filter(s => !existingIndices.has(s.index)) || [];
  }

  // Modal handlers
  openAddVideoModal(): void {
    this.selectedQuality = '';
    this.modalMode.set('add-video');
  }

  openAddAudioModal(): void {
    this.selectedTrackIndex = null;
    this.modalMode.set('add-audio');
  }

  openAddSubtitleModal(): void {
    this.selectedTrackIndex = null;
    this.modalMode.set('add-subtitle');
  }

  openRecreateModal(transcode: TranscodeInfo): void {
    this.selectedTranscode.set(transcode);
    this.modalMode.set('recreate');
  }

  openDeleteModal(transcode: TranscodeInfo): void {
    this.selectedTranscode.set(transcode);
    this.modalMode.set('delete');
  }

  closeModal(): void {
    this.modalMode.set(null);
    this.selectedTranscode.set(null);
    this.selectedQuality = '';
    this.selectedTrackIndex = null;
  }

  // Actions
  addVideoTranscoding(): void {
    const media = this.selectedMedia();
    if (!media || !this.selectedQuality) return;

    this.modalLoading.set(true);
    this.api.addTranscoding(media.id, {
      type: 'video',
      quality: this.selectedQuality
    }).subscribe({
      next: () => {
        this.modalLoading.set(false);
        this.success.set(this.translate.instant('transcodingAdmin.jobEnqueued'));
        this.closeModal();
        this.loadMediaDetails(media.id);
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }

  addAudioTranscoding(): void {
    const media = this.selectedMedia();
    if (!media || this.selectedTrackIndex === null) return;

    this.modalLoading.set(true);
    this.api.addTranscoding(media.id, {
      type: 'audio',
      track_index: this.selectedTrackIndex
    }).subscribe({
      next: () => {
        this.modalLoading.set(false);
        this.success.set(this.translate.instant('transcodingAdmin.jobEnqueued'));
        this.closeModal();
        this.loadMediaDetails(media.id);
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }

  addSubtitleTranscoding(): void {
    const media = this.selectedMedia();
    if (!media || this.selectedTrackIndex === null) return;

    this.modalLoading.set(true);
    this.api.addTranscoding(media.id, {
      type: 'subtitle',
      track_index: this.selectedTrackIndex
    }).subscribe({
      next: () => {
        this.modalLoading.set(false);
        this.success.set(this.translate.instant('transcodingAdmin.jobEnqueued'));
        this.closeModal();
        this.loadMediaDetails(media.id);
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }

  recreateTranscoding(): void {
    const transcode = this.selectedTranscode();
    const media = this.selectedMedia();
    if (!transcode || !media) return;

    this.modalLoading.set(true);
    this.api.recreateTranscoding(transcode.id).subscribe({
      next: () => {
        this.modalLoading.set(false);
        this.success.set(this.translate.instant('transcodingAdmin.jobEnqueued'));
        this.closeModal();
        this.loadMediaDetails(media.id);
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }

  deleteTranscoding(): void {
    const transcode = this.selectedTranscode();
    const media = this.selectedMedia();
    if (!transcode || !media) return;

    this.modalLoading.set(true);
    this.api.deleteTranscoding(transcode.id).subscribe({
      next: () => {
        this.modalLoading.set(false);
        this.success.set(this.translate.instant('transcodingAdmin.deleteSuccess'));
        this.closeModal();
        this.loadMediaDetails(media.id);
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }
}
