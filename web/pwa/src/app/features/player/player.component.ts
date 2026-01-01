import { Component, OnInit, OnDestroy, ViewChild, ElementRef, signal, inject, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import * as dashjs from 'dashjs';
import { ApiService, MediaDetail, AudioStream, SubtitleStream } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';

interface QualityLevel {
  index: number;       // Original index from dash.js getRepresentationsByType()
  id: string;          // Representation ID
  height: number;
  bandwidth: number;
}

@Component({
  selector: 'app-player',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="h-screen bg-black flex flex-col overflow-hidden">
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
      <div 
        class="flex-1 flex items-center justify-center bg-black relative group overflow-hidden"
        #playerContainer
        (mousemove)="onMouseMove()"
        (click)="onVideoClick($event)"
      >
        @if (loading()) {
          <div class="absolute inset-0 flex items-center justify-center z-10">
            <div class="w-12 h-12 border-4 border-zinc-600 border-t-white rounded-full animate-spin"></div>
          </div>
        }
        
        @if (error()) {
          <div class="absolute inset-0 flex items-center justify-center z-10">
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
          class="w-full h-full max-h-full object-contain cursor-pointer"
          [class.hidden]="loading() || error()"
          (timeupdate)="onTimeUpdate()"
          (play)="onPlayStateChange()"
          (pause)="onPlayStateChange()"
          (volumechange)="onVolumeChange()"
          (loadedmetadata)="onMetadataLoaded()"
          (dblclick)="toggleFullscreen()"
        ></video>

        <!-- Custom Controls Overlay -->
        @if (media() && !error() && !loading()) {
          <div 
            class="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/90 via-black/60 to-transparent pt-16 pb-4 px-4 transition-opacity duration-300"
            [class.opacity-0]="!showControls() && isPlaying()"
            [class.pointer-events-none]="!showControls() && isPlaying()"
          >
            <!-- Progress Bar -->
            <div 
              class="w-full h-1 bg-zinc-600 rounded-full mb-4 cursor-pointer group/progress"
              (click)="onSeek($event)"
              #progressBar
            >
              <div 
                class="h-full bg-blue-500 rounded-full relative"
                [style.width.%]="progress()"
              >
                <div class="absolute right-0 top-1/2 -translate-y-1/2 w-3 h-3 bg-white rounded-full opacity-0 group-hover/progress:opacity-100 transition-opacity"></div>
              </div>
            </div>

            <!-- Controls Row -->
            <div class="flex items-center gap-4">
              <!-- Play/Pause -->
              <button 
                class="text-white hover:text-blue-400 transition-colors"
                (click)="togglePlay($event)"
              >
                @if (isPlaying()) {
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M6 4h4v16H6V4zm8 0h4v16h-4V4z"/>
                  </svg>
                } @else {
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M8 5v14l11-7z"/>
                  </svg>
                }
              </button>

              <!-- Volume -->
              <div class="flex items-center gap-2">
                <button 
                  class="text-white hover:text-blue-400 transition-colors"
                  (click)="toggleMute($event)"
                >
                  @if (isMuted() || volume() === 0) {
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2" />
                    </svg>
                  } @else {
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.536 8.464a5 5 0 010 7.072m2.828-9.9a9 9 0 010 12.728M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
                    </svg>
                  }
                </button>
                <input
                  type="range"
                  min="0"
                  max="1"
                  step="0.1"
                  [value]="volume()"
                  (input)="onVolumeSliderChange($event)"
                  class="w-20 h-1 bg-zinc-600 rounded-lg appearance-none cursor-pointer accent-blue-500"
                />
              </div>

              <!-- Time Display -->
              <div class="text-white text-sm">
                {{ formatTime(currentTime()) }} / {{ formatTime(duration()) }}
              </div>

              <!-- Spacer -->
              <div class="flex-1"></div>

              <!-- Track Selectors -->
              <!-- Audio Track Selector -->
              @if (audioTracks().length > 1) {
                <div class="flex items-center gap-2">
                  <label class="text-zinc-400 text-sm">Audio:</label>
                  <select
                    class="bg-zinc-800 text-white text-sm rounded px-2 py-1 border border-zinc-700 focus:border-blue-500 focus:outline-none"
                    [value]="currentAudioIndex()"
                    (change)="onAudioChange($event)"
                  >
                    @for (track of audioTracks(); track track.index) {
                      <option [value]="$index">
                        {{ track.title || track.language || 'Track ' + ($index + 1) }}
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
                    class="bg-zinc-800 text-white text-sm rounded px-2 py-1 border border-zinc-700 focus:border-blue-500 focus:outline-none"
                    [value]="currentSubtitleIndex()"
                    (change)="onSubtitleChange($event)"
                  >
                    <option value="-1">Off</option>
                    @for (track of subtitleTracks(); track track.index) {
                      <option [value]="$index">
                        {{ track.title || track.language || 'Track ' + ($index + 1) }}
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
                    class="bg-zinc-800 text-white text-sm rounded px-2 py-1 border border-zinc-700 focus:border-blue-500 focus:outline-none"
                    [value]="currentQualityIndex()"
                    (change)="onQualityChange($event)"
                  >
                    <option value="-1">Auto</option>
                    @for (quality of qualityLevels(); track quality.index; let i = $index) {
                      <option [value]="i">
                        {{ quality.height }}p
                      </option>
                    }
                  </select>
                </div>
              }

              <!-- Fullscreen -->
              <button 
                class="text-white hover:text-blue-400 transition-colors"
                (click)="toggleFullscreen()"
              >
                @if (isFullscreen()) {
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                } @else {
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
                  </svg>
                }
              </button>
            </div>
          </div>
        }
      </div>
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }
    
    input[type="range"]::-webkit-slider-thumb {
      -webkit-appearance: none;
      width: 12px;
      height: 12px;
      background: white;
      border-radius: 50%;
      cursor: pointer;
    }
    
    input[type="range"]::-moz-range-thumb {
      width: 12px;
      height: 12px;
      background: white;
      border-radius: 50%;
      cursor: pointer;
      border: none;
    }
  `]
})
export class PlayerComponent implements OnInit, OnDestroy {
  @ViewChild('videoElement', { static: true }) videoElement!: ElementRef<HTMLVideoElement>;
  @ViewChild('playerContainer') playerContainer!: ElementRef<HTMLDivElement>;
  @ViewChild('progressBar') progressBar!: ElementRef<HTMLDivElement>;

  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);

  private player: dashjs.MediaPlayerClass | null = null;
  private streamToken: string | null = null;
  private controlsTimeout: any = null;

  // State signals
  loading = signal(true);
  error = signal<string | null>(null);
  media = signal<MediaDetail | null>(null);
  
  // Playback signals
  isPlaying = signal(false);
  isMuted = signal(false);
  volume = signal(1);
  currentTime = signal(0);
  duration = signal(0);
  progress = signal(0);
  showControls = signal(true);
  isFullscreen = signal(false);
  
  // Track signals
  audioTracks = signal<AudioStream[]>([]);
  subtitleTracks = signal<SubtitleStream[]>([]);
  currentAudioIndex = signal(0);
  currentSubtitleIndex = signal(-1);
  
  // Quality signals
  qualityLevels = signal<QualityLevel[]>([]);
  currentQualityIndex = signal(-1); // -1 = auto

  @HostListener('document:keydown', ['$event'])
  onKeyDown(event: KeyboardEvent): void {
    if (event.target instanceof HTMLInputElement || event.target instanceof HTMLSelectElement) {
      return;
    }
    
    switch (event.code) {
      case 'Space':
        event.preventDefault();
        this.togglePlay();
        break;
      case 'ArrowLeft':
        event.preventDefault();
        this.seek(-10);
        break;
      case 'ArrowRight':
        event.preventDefault();
        this.seek(10);
        break;
      case 'ArrowUp':
        event.preventDefault();
        this.adjustVolume(0.1);
        break;
      case 'ArrowDown':
        event.preventDefault();
        this.adjustVolume(-0.1);
        break;
      case 'KeyM':
        event.preventDefault();
        this.toggleMute();
        break;
      case 'KeyF':
        event.preventDefault();
        this.toggleFullscreen();
        break;
    }
  }

  @HostListener('document:fullscreenchange')
  onFullscreenChange(): void {
    this.isFullscreen.set(!!document.fullscreenElement);
  }

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
    if (this.controlsTimeout) {
      clearTimeout(this.controlsTimeout);
    }
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
          },
          text: {
            defaultEnabled: false
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
        // Extract more detailed error info
        const errorMsg = e.error?.message || e.error?.code || e.event?.type || 'Unknown error';
        const errorDetail = e.error?.data?.response?.status ? ` (HTTP ${e.error.data.response.status})` : '';
        this.error.set('Playback error: ' + errorMsg + errorDetail);
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

  // Video event handlers
  onTimeUpdate(): void {
    const video = this.videoElement.nativeElement;
    this.currentTime.set(video.currentTime);
    if (video.duration > 0) {
      this.progress.set((video.currentTime / video.duration) * 100);
    }
  }

  onPlayStateChange(): void {
    const video = this.videoElement.nativeElement;
    this.isPlaying.set(!video.paused);
  }

  onVolumeChange(): void {
    const video = this.videoElement.nativeElement;
    this.volume.set(video.volume);
    this.isMuted.set(video.muted);
  }

  onMetadataLoaded(): void {
    const video = this.videoElement.nativeElement;
    this.duration.set(video.duration);
  }

  // Control handlers
  onVideoClick(event: Event): void {
    // Only toggle play if clicking directly on video, not on controls
    if (event.target === this.videoElement.nativeElement) {
      this.togglePlay();
    }
  }

  onMouseMove(): void {
    this.showControls.set(true);
    
    if (this.controlsTimeout) {
      clearTimeout(this.controlsTimeout);
    }
    
    this.controlsTimeout = setTimeout(() => {
      if (this.isPlaying()) {
        this.showControls.set(false);
      }
    }, 3000);
  }

  togglePlay(event?: Event): void {
    event?.stopPropagation();
    const video = this.videoElement.nativeElement;
    if (video.paused) {
      video.play();
    } else {
      video.pause();
    }
  }

  toggleMute(event?: Event): void {
    event?.stopPropagation();
    const video = this.videoElement.nativeElement;
    video.muted = !video.muted;
  }

  onVolumeSliderChange(event: Event): void {
    const input = event.target as HTMLInputElement;
    const video = this.videoElement.nativeElement;
    video.volume = parseFloat(input.value);
    if (video.volume > 0) {
      video.muted = false;
    }
  }

  adjustVolume(delta: number): void {
    const video = this.videoElement.nativeElement;
    video.volume = Math.max(0, Math.min(1, video.volume + delta));
    if (video.volume > 0) {
      video.muted = false;
    }
  }

  onSeek(event: MouseEvent): void {
    const progressBar = this.progressBar.nativeElement;
    const rect = progressBar.getBoundingClientRect();
    const percent = (event.clientX - rect.left) / rect.width;
    const video = this.videoElement.nativeElement;
    video.currentTime = percent * video.duration;
  }

  seek(seconds: number): void {
    const video = this.videoElement.nativeElement;
    video.currentTime = Math.max(0, Math.min(video.duration, video.currentTime + seconds));
  }

  toggleFullscreen(): void {
    if (!document.fullscreenElement) {
      this.playerContainer.nativeElement.requestFullscreen();
    } else {
      document.exitFullscreen();
    }
  }

  formatTime(seconds: number): string {
    if (!seconds || isNaN(seconds)) return '0:00';
    
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);
    
    if (hrs > 0) {
      return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${mins}:${secs.toString().padStart(2, '0')}`;
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
        // Disable text tracks
        this.player.enableText(false);
      } else {
        // Enable text first, then set the track
        this.player.enableText(true);
        // Use setTextTrack with the track index
        // dash.js text track indices are 0-based
        this.player.setTextTrack(index);
      }
    }
  }

  onQualityChange(event: Event): void {
    const select = event.target as HTMLSelectElement;
    const uiIndex = parseInt(select.value, 10);
    this.currentQualityIndex.set(uiIndex);
    
    if (this.player) {
      if (uiIndex === -1) {
        // Enable auto quality
        this.player.updateSettings({
          streaming: {
            abr: {
              autoSwitchBitrate: { video: true }
            }
          }
        });
      } else {
        // Disable auto and set specific quality
        this.player.updateSettings({
          streaming: {
            abr: {
              autoSwitchBitrate: { video: false }
            }
          }
        });
        // Use the representation ID from our quality levels array
        const levels = this.qualityLevels();
        if (levels[uiIndex]) {
          // Use setRepresentationForTypeById which is more reliable
          this.player.setRepresentationForTypeById('video', levels[uiIndex].id);
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
