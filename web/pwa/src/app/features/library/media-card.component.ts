import { Component, Input } from '@angular/core';
import { MediaListItem } from '../../core/services/api.service';

@Component({
  selector: 'app-media-card',
  standalone: true,
  template: `
    <div class="bg-slate-800 rounded-lg overflow-hidden shadow-lg hover:ring-2 hover:ring-blue-500 transition cursor-pointer group">
      <!-- Placeholder thumbnail -->
      <div class="aspect-video bg-slate-700 flex items-center justify-center relative overflow-hidden">
        <svg class="w-16 h-16 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/>
        </svg>
        <!-- Play overlay on hover -->
        <div class="absolute inset-0 bg-black/40 flex items-center justify-center opacity-0 group-hover:opacity-100 transition">
          <svg class="w-12 h-12 text-white" fill="currentColor" viewBox="0 0 24 24">
            <path d="M8 5v14l11-7z"/>
          </svg>
        </div>
        <!-- Resolution badge -->
        @if (media.resolution) {
          <span class="absolute top-2 right-2 px-2 py-1 bg-black/60 text-xs text-white rounded">
            {{ media.resolution }}
          </span>
        }
      </div>

      <div class="p-4">
        <h3 class="text-white font-medium truncate" [title]="media.title">
          {{ media.title }}
        </h3>
        <div class="mt-2 flex items-center text-sm text-slate-400 space-x-3">
          <span>{{ formatDuration(media.duration) }}</span>
          @if (media.audio_tracks > 1) {
            <span class="flex items-center">
              <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.536 8.464a5 5 0 010 7.072m2.828-9.9a9 9 0 010 12.728M5.586 15.536a5 5 0 001.414 1.414m2.828-9.9a9 9 0 012.829-2.829"/>
              </svg>
              {{ media.audio_tracks }}
            </span>
          }
          @if (media.has_subtitles) {
            <span class="flex items-center">
              <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z"/>
              </svg>
              {{ media.subtitle_tracks }}
            </span>
          }
        </div>
      </div>
    </div>
  `
})
export class MediaCardComponent {
  @Input({ required: true }) media!: MediaListItem;

  formatDuration(seconds: number): string {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);

    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  }
}
