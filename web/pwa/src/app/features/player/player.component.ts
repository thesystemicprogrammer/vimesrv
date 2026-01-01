import { Component, OnInit, OnDestroy, ViewChild, ElementRef, signal, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import * as dashjs from 'dashjs';
import { ApiService, MediaDetail, AudioStream, SubtitleStream } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';

interface QualityLevel {
  index: number;
  id: string;
  height: number;
  bandwidth: number;
}

@Component({
  selector: 'app-player',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="min-h-screen bg-black flex flex-col">
      <!-- Header -->
      <header class="bg-zinc-900 px-4 py-3 flex items-center gap-4">
        <a routerLink="/" class="text-zinc-400 hover:text-white transition-colors">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
        </a>
        <h1 class="text-white text-lg font-medium truncate">{{ media()?.title || 'Loading...' }}</h1>
      </header>

      <!-- Video Container -->
      <div class="flex-1 flex items-center justify-center bg-black relative">
        @if (loading()) {
          <div class="absolute inset-0 flex items-center justify-center">
            <div class="w-12 h-12 border-4 border-zinc-600 border-t-white rounded-full animate-spin"></div>
          </div>
        }
        
        @if (error()) {
          <div class="absolute inset-0 flex items-center justify-center">
            <div class="text-center text-red-500">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <p class="text-lg">{{ error() }}</p>
              <a routerLink="/" class="mt-4 inline-block text-blue-400 hover:text-blue-300">Back to Library</a>
            </div>
          </div>
        }

        <video
          #videoElement
          class="w-full h-full max-h-[calc(100vh-120px)]"
          controls
          autoplay
          [class.hidden]="loading() || error()"
        ></video>
      </div>

      <!-- Controls Panel -->
      @if (media() && !error()) {
        <div class="bg-zinc-900 px-4 py-3 flex flex-wrap gap-4 items-center justify-center">
          <!-- Audio Track Selector -->
          @if (audioTracks().length > 1) {
            <div class="flex items-center gap-2">
              <label class="text-zinc-400 text-sm">Audio:</label>
              <select
                class="bg-zinc-800 text-white text-sm rounded px-3 py-1.5 border border-zinc-700 focus:border-blue-500 focus:outline-none"
                [value]="currentAudioIndex()"
                (change)="onAudioChange($event)"
              >
                @for (track of audioTracks(); track track.index) {
                  <option [value]="$index">
                    {{ track.language || 'Unknown' }}{{ track.title ? ' - ' + track.title : '' }}
                    {{ track.channels ? ' (' + track.channels + 'ch)' : '' }}
                  </option>
                }
              </select>
            </div>
          }

          <!-- Subtitle Track Selector -->
          @if (subtitleTracks().length > 0) {
            <div class="flex items-center gap-2">
              <label class="text-zinc-400 text-sm">Subtitles:</label>
              <select
                class="bg-zinc-800 text-white text-sm rounded px-3 py-1.5 border border-zinc-700 focus:border-blue-500 focus:outline-none"
                [value]="currentSubtitleIndex()"
                (change)="onSubtitleChange($event)"
              >
                <option value="-1">Off</option>
                @for (track of subtitleTracks(); track track.index) {
                  <option [value]="$index">
                    {{ track.language || 'Unknown' }}{{ track.title ? ' - ' + track.title : '' }}
                  </option>
                }
              </select>
            </div>
          }

          <!-- Quality Selector -->
          @if (qualityLevels().length > 1) {
            <div class="flex items-center gap-2">
              <label class="text-zinc-400 text-sm">Quality:</label>
              <select
                class="bg-zinc-800 text-white text-sm rounded px-3 py-1.5 border border-zinc-700 focus:border-blue-500 focus:outline-none"
                [value]="currentQualityIndex()"
                (change)="onQualityChange($event)"
              >
                <option value="-1">Auto</option>
                @for (quality of qualityLevels(); track quality.index; let i = $index) {
                  <option [value]="i">
                    {{ quality.height }}p{{ quality.bandwidth ? ' (' + formatBitrate(quality.bandwidth) + ')' : '' }}
                  </option>
                }
              </select>
            </div>
          }
        </div>
      }
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }
    
    video::-webkit-media-controls-enclosure {
      border-radius: 0;
    }
  `]
})
export class PlayerComponent implements OnInit, OnDestroy {
  @ViewChild('videoElement', { static: true }) videoElement!: ElementRef<HTMLVideoElement>;

  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);

  private player: dashjs.MediaPlayerClass | null = null;
  private streamToken: string | null = null;

  // State signals
  loading = signal(true);
  error = signal<string | null>(null);
  media = signal<MediaDetail | null>(null);
  
  // Track signals
  audioTracks = signal<AudioStream[]>([]);
  subtitleTracks = signal<SubtitleStream[]>([]);
  currentAudioIndex = signal(0);
  currentSubtitleIndex = signal(-1);
  
  // Quality signals
  qualityLevels = signal<QualityLevel[]>([]);
  currentQualityIndex = signal(-1); // -1 = auto

  ngOnInit(): void {
    const mediaId = this.route.snapshot.paramMap.get('id');
    if (!mediaId) {
      this.error.set('Invalid media ID');
      this.loading.set(false);
      return;
    }

    this.loadMedia(mediaId);
  }

  ngOnDestroy(): void {
    this.destroyPlayer();
  }

  private async loadMedia(mediaId: string): Promise<void> {
    try {
      // Fetch media details
      const mediaResponse = await this.api.getMedia(mediaId).toPromise();
      if (!mediaResponse?.data) {
        throw new Error('Media not found');
      }
      
      const media = mediaResponse.data;
      this.media.set(media);
      this.audioTracks.set(media.audio_streams || []);
      this.subtitleTracks.set(media.subtitle_streams || []);

      // Get stream token
      const tokenResponse = await this.api.getStreamToken().toPromise();
      if (!tokenResponse?.data?.token) {
        throw new Error('Failed to get stream token');
      }
      this.streamToken = tokenResponse.data.token;

      // Build manifest URL (token will be sent via Authorization header)
      const manifestUrl = media.dash_manifest_url;
      
      // Initialize dash.js player
      this.initializePlayer(manifestUrl);
      
    } catch (err) {
      console.error('Failed to load media:', err);
      this.error.set(err instanceof Error ? err.message : 'Failed to load media');
      this.loading.set(false);
    }
  }

  private initializePlayer(manifestUrl: string): void {
    try {
      this.player = dashjs.MediaPlayer().create();
      
      // Configure player to send Authorization header with stream token
      const streamToken = this.streamToken;
      if (streamToken) {
        this.player.addRequestInterceptor((request: any) => {
          if (!request.headers) {
            request.headers = {};
          }
          request.headers['Authorization'] = 'Bearer ' + streamToken;
          return Promise.resolve(request);
        });
      }
      
      // Configure player
      this.player.updateSettings({
        streaming: {
          abr: {
            autoSwitchBitrate: { video: true, audio: true }
          },
          buffer: {
            fastSwitchEnabled: true
          }
        }
      });

      // Set up event listeners
      this.player.on(dashjs.MediaPlayer.events.STREAM_INITIALIZED, () => {
        this.loading.set(false);
        this.updateQualityLevels();
      });

      this.player.on(dashjs.MediaPlayer.events.ERROR, (e: any) => {
        console.error('DASH player error:', e);
        this.error.set('Playback error: ' + (e.error?.message || 'Unknown error'));
      });

      this.player.on(dashjs.MediaPlayer.events.QUALITY_CHANGE_RENDERED, (e: any) => {
        if (e.mediaType === 'video') {
          this.updateCurrentQuality();
        }
      });

      // Initialize player with video element and manifest
      this.player.initialize(
        this.videoElement.nativeElement,
        manifestUrl,
        true // autoplay
      );

    } catch (err) {
      console.error('Failed to initialize player:', err);
      this.error.set('Failed to initialize video player');
      this.loading.set(false);
    }
  }

  private updateQualityLevels(): void {
    if (!this.player) return;
    
    try {
      // Use the new API: getRepresentationsByType
      const representations = this.player.getRepresentationsByType('video');
      const levels: QualityLevel[] = representations.map((rep: any, index: number) => ({
        index,
        id: rep.id,
        height: rep.height || 0,
        bandwidth: rep.bandwidth || 0
      }));
      
      // Sort by height descending
      levels.sort((a, b) => b.height - a.height);
      this.qualityLevels.set(levels);
    } catch (err) {
      console.warn('Failed to get quality levels:', err);
    }
  }

  private updateCurrentQuality(): void {
    if (!this.player) return;
    
    const settings = this.player.getSettings();
    if (settings.streaming?.abr?.autoSwitchBitrate?.video) {
      this.currentQualityIndex.set(-1);
    }
    // When not auto, we track the index ourselves via onQualityChange
  }

  private destroyPlayer(): void {
    if (this.player) {
      this.player.reset();
      this.player = null;
    }
  }

  onAudioChange(event: Event): void {
    const select = event.target as HTMLSelectElement;
    const index = parseInt(select.value, 10);
    this.currentAudioIndex.set(index);
    
    if (this.player) {
      const tracks = this.player.getTracksFor('audio');
      if (tracks[index]) {
        this.player.setCurrentTrack(tracks[index]);
      }
    }
  }

  onSubtitleChange(event: Event): void {
    const select = event.target as HTMLSelectElement;
    const index = parseInt(select.value, 10);
    this.currentSubtitleIndex.set(index);
    
    if (this.player) {
      if (index === -1) {
        this.player.setTextTrack(-1);
      } else {
        const tracks = this.player.getTracksFor('text');
        if (tracks[index]) {
          this.player.setCurrentTrack(tracks[index]);
        }
      }
    }
  }

  onQualityChange(event: Event): void {
    const select = event.target as HTMLSelectElement;
    const index = parseInt(select.value, 10);
    this.currentQualityIndex.set(index);
    
    if (this.player) {
      if (index === -1) {
        // Enable auto quality
        this.player.updateSettings({
          streaming: {
            abr: {
              autoSwitchBitrate: { video: true }
            }
          }
        });
      } else {
        // Disable auto and set specific quality using the new API
        this.player.updateSettings({
          streaming: {
            abr: {
              autoSwitchBitrate: { video: false }
            }
          }
        });
        // Use setRepresentationForTypeByIndex for dash.js v5
        const levels = this.qualityLevels();
        if (levels[index]) {
          this.player.setRepresentationForTypeByIndex('video', index);
        }
      }
    }
  }

  formatBitrate(bitrate: number): string {
    if (bitrate >= 1000000) {
      return (bitrate / 1000000).toFixed(1) + ' Mbps';
    }
    return (bitrate / 1000).toFixed(0) + ' Kbps';
  }
}
