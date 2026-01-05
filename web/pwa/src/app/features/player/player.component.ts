import { Component, OnInit, OnDestroy, ViewChild, ElementRef, signal, inject, HostListener } from '@angular/core';
import { CommonModule, DecimalPipe } from '@angular/common';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { Subscription } from 'rxjs';
import * as dashjs from 'dashjs';
import { ApiService, MediaDetail, AudioStream, SubtitleStream } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';
import { PlaybackDecisionService, PlaybackMode, PlaybackDecision } from '../../core/services/playback-decision.service';
import { PlaybackMonitorService, FallbackEvent } from '../../core/services/playback-monitor.service';
import { BandwidthService } from '../../core/services/bandwidth.service';

interface QualityLevel {
  index: number;       // Original index from dash.js getRepresentationsByType()
  id: string;          // Representation ID
  height: number;
  bandwidth: number;
}

interface DebugStats {
  playbackMode: 'Direct Play' | 'Direct Stream' | 'Adaptive';
  bandwidthMbps: number | null;
  bufferAhead: number;
  stallCount: number;
  mediaBitrateMbps: number;
  currentQuality: string | null;  // e.g., "1080p" or "1080p (auto)"
  abrEnabled: boolean | null;     // only for DASH
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

        <!-- Fallback Notification -->
        @if (notificationMessage()) {
          <div class="absolute top-4 left-1/2 -translate-x-1/2 z-20 bg-zinc-800/95 text-white px-4 py-2 rounded-lg shadow-lg flex items-center gap-2 animate-fade-in">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-blue-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span class="text-sm">{{ notificationMessage() }}</span>
          </div>
        }

        <!-- Debug Panel -->
        @if (showDebugPanel() && debugStats()) {
          <div class="absolute top-12 right-4 z-30 bg-black/85 text-white text-xs font-mono p-3 rounded-lg min-w-48 backdrop-blur-sm">
            <div class="font-bold mb-2 flex items-center gap-2">
              Playback Debug
              <span class="w-2 h-2 rounded-full"
                    [class.bg-green-500]="debugStats()?.playbackMode === 'Direct Play'"
                    [class.bg-yellow-500]="debugStats()?.playbackMode === 'Direct Stream'"
                    [class.bg-blue-500]="debugStats()?.playbackMode === 'Adaptive'">
              </span>
            </div>
            <div class="space-y-0.5 text-zinc-300">
              <div>Mode: <span class="text-white">{{ debugStats()?.playbackMode }}</span></div>
              <div>Bandwidth: <span class="text-white">{{ debugStats()?.bandwidthMbps !== null ? (debugStats()?.bandwidthMbps | number:'1.1-1') + ' Mbps' : '?' }}</span></div>
              <div>Buffer: <span class="text-white">{{ debugStats()?.bufferAhead | number:'1.1-1' }}s</span></div>
              <div>Stalls: <span class="text-white">{{ debugStats()?.stallCount }}</span></div>
              <div>Media: <span class="text-white">{{ debugStats()?.mediaBitrateMbps | number:'1.1-1' }} Mbps</span></div>
              @if (debugStats()?.currentQuality) {
                <div>Quality: <span class="text-white">{{ debugStats()?.currentQuality }}</span></div>
              }
              @if (debugStats()?.abrEnabled !== null) {
                <div>ABR: <span class="text-white">{{ debugStats()?.abrEnabled ? 'enabled' : 'disabled' }}</span></div>
              }
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

              <!-- Playback Mode Indicator -->
              @if (playbackModeLabel()) {
                <div class="text-xs px-2 py-0.5 rounded"
                     [class]="playbackModeLabel() === 'Adaptive' ? 'bg-zinc-700 text-zinc-300' : 'bg-green-700 text-green-100'">
                  {{ playbackModeLabel() }}
                </div>
              }

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
    
    @keyframes fadeIn {
      from { opacity: 0; transform: translate(-50%, -10px); }
      to { opacity: 1; transform: translate(-50%, 0); }
    }
    
    .animate-fade-in {
      animation: fadeIn 0.3s ease-out forwards;
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
  private readonly playbackDecision = inject(PlaybackDecisionService);
  private readonly playbackMonitor = inject(PlaybackMonitorService);
  private readonly bandwidthService = inject(BandwidthService);

  private player: dashjs.MediaPlayerClass | null = null;
  private streamToken: string | null = null;
  private controlsTimeout: any = null;
  private currentPlaybackMode: PlaybackMode = 'dash';
  private fallbackSubscription: Subscription | null = null;
  private notificationTimeout: any = null;

  // DASH pre-warming
  private manifestPrewarmPromise: Promise<void> | null = null;

  // Debug panel
  private tapCount = 0;
  private tapTimeout: any = null;
  private debugPanelTimeout: any = null;
  private debugStatsInterval: any = null;

  // State signals
  loading = signal(true);
  error = signal<string | null>(null);
  media = signal<MediaDetail | null>(null);
  playbackModeLabel = signal<string>('');  // "Direct Play", "Direct Stream", "Adaptive"
  notificationMessage = signal<string | null>(null);  // Fallback notification
  
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

  // Debug panel
  showDebugPanel = signal(false);
  debugStats = signal<DebugStats | null>(null);

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
      case 'KeyD':
        event.preventDefault();
        this.toggleDebugPanel();
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
    if (this.notificationTimeout) {
      clearTimeout(this.notificationTimeout);
    }
    if (this.tapTimeout) {
      clearTimeout(this.tapTimeout);
    }
    if (this.debugPanelTimeout) {
      clearTimeout(this.debugPanelTimeout);
    }
    if (this.debugStatsInterval) {
      clearInterval(this.debugStatsInterval);
    }
    if (this.fallbackSubscription) {
      this.fallbackSubscription.unsubscribe();
    }
    this.playbackMonitor.stopMonitoring();
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

      // Make playback decision
      const decision = await this.playbackDecision.decide(media);
      console.log('Playback decision:', decision);

      this.currentPlaybackMode = decision.mode;

      if (decision.mode === 'direct' || decision.mode === 'direct-stream') {
        // Pre-warm DASH manifest in background for faster fallback
        this.prewarmDashManifest(media.dash_manifest_url);

        // Initialize direct player
        await this.initializeDirectPlayer(decision.url);
        this.playbackModeLabel.set(decision.mode === 'direct' ? 'Direct Play' : 'Direct Stream');
      } else {
        // Fall back to DASH
        this.initializeDashPlayer(media.dash_manifest_url);
        this.playbackModeLabel.set('Adaptive');
      }
      
    } catch (err) {
      console.error('Failed to load media:', err);
      this.error.set(err instanceof Error ? err.message : 'Failed to load media');
      this.loading.set(false);
    }
  }

  /**
   * Pre-warm DASH manifest in background for faster fallback
   */
  private prewarmDashManifest(manifestUrl: string): void {
    if (!manifestUrl || !this.streamToken) return;

    this.manifestPrewarmPromise = fetch(manifestUrl, {
      headers: { 'Authorization': `Bearer ${this.streamToken}` }
    })
    .then(response => {
      if (response.ok) {
        console.log('DASH manifest pre-warmed');
      }
    })
    .catch(err => {
      console.warn('DASH manifest pre-warm failed:', err);
    });
  }

  /**
   * Initialize direct player for native video playback
   * Uses query param token for authentication (Range requests require this)
   */
  private async initializeDirectPlayer(url: string): Promise<void> {
    try {
      const video = this.videoElement.nativeElement;

      // Build URL with stream token as query param
      // This is necessary because video element Range requests can't use custom headers
      const separator = url.includes('?') ? '&' : '?';
      const directUrl = `${url}${separator}token=${this.streamToken}`;

      // Set video source
      video.src = directUrl;

      video.onerror = () => {
        console.error('Direct playback error:', video.error);
        // Attempt fallback to DASH
        this.fallbackToDash();
      };

      video.onloadedmetadata = () => {
        this.loading.set(false);
        // Load VTT subtitles for direct play
        this.loadSubtitlesForDirectPlay();

        // Start monitoring for playback issues
        this.startPlaybackMonitoring();
      };

      // Start playback
      try {
        await video.play();
      } catch (playErr) {
        console.warn('Autoplay blocked:', playErr);
        // Autoplay blocked - user will need to click play
        this.loading.set(false);
      }

    } catch (err) {
      console.error('Direct playback initialization failed:', err);
      // Attempt fallback to DASH
      this.fallbackToDash();
    }
  }

  /**
   * Load VTT subtitles as <track> elements for direct play mode
   */
  private loadSubtitlesForDirectPlay(): void {
    const media = this.media();
    if (!media || !media.subtitle_streams?.length) return;

    const video = this.videoElement.nativeElement;

    // Remove any existing tracks
    while (video.firstChild) {
      video.removeChild(video.firstChild);
    }

    // Add track elements for each subtitle
    media.subtitle_streams.forEach((sub, idx) => {
      const track = document.createElement('track');
      track.kind = 'subtitles';
      track.label = sub.title || sub.language || `Track ${idx + 1}`;
      track.srclang = sub.language || 'und';

      // VTT files are served via DASH handler at same path pattern
      // Add stream token as query param since track elements can't use custom headers
      const baseUrl = `/stream/dash/content/${media.id}/subtitle-${sub.index}.vtt`;
      track.src = this.streamToken ? `${baseUrl}?token=${this.streamToken}` : baseUrl;

      video.appendChild(track);
    });
  }

  /**
   * Start monitoring playback for issues that would trigger fallback
   */
  private startPlaybackMonitoring(): void {
    const video = this.videoElement.nativeElement;

    // Subscribe to fallback events
    this.fallbackSubscription = this.playbackMonitor.fallbackNeeded$.subscribe(
      (event: FallbackEvent) => {
        console.log('Fallback triggered by monitor:', event);
        this.handleMonitorFallback(event);
      }
    );

    // Start monitoring
    this.playbackMonitor.startMonitoring(video);
  }

  /**
   * Handle fallback triggered by the playback monitor
   */
  private handleMonitorFallback(event: FallbackEvent): void {
    // Save current position before switching
    const currentPosition = event.currentTime;

    // Show notification
    this.showNotification('Switching to adaptive streaming...');

    // Perform fallback
    this.fallbackToDash(currentPosition);
  }

  /**
   * Show a temporary notification message
   */
  private showNotification(message: string, durationMs: number = 3000): void {
    this.notificationMessage.set(message);

    if (this.notificationTimeout) {
      clearTimeout(this.notificationTimeout);
    }

    this.notificationTimeout = setTimeout(() => {
      this.notificationMessage.set(null);
    }, durationMs);
  }

  /**
   * Fall back from direct play to DASH streaming
   */
  private async fallbackToDash(resumePosition?: number): Promise<void> {
    const media = this.media();
    if (!media?.dash_manifest_url) {
      this.error.set('Playback failed and no fallback available');
      this.loading.set(false);
      return;
    }

    console.log('Falling back to DASH streaming', resumePosition ? `at ${resumePosition}s` : '');

    // Wait for pre-warmed manifest (max 500ms)
    if (this.manifestPrewarmPromise) {
      await Promise.race([
        this.manifestPrewarmPromise,
        new Promise(resolve => setTimeout(resolve, 500))
      ]);
    }

    // Stop monitoring
    this.playbackMonitor.stopMonitoring();
    if (this.fallbackSubscription) {
      this.fallbackSubscription.unsubscribe();
      this.fallbackSubscription = null;
    }

    // Clean up direct play
    this.cleanupDirectPlay();

    // Initialize DASH player
    this.currentPlaybackMode = 'dash';
    this.playbackModeLabel.set('Adaptive');
    this.initializeDashPlayer(media.dash_manifest_url, resumePosition);
  }

  /**
   * Clean up direct play resources (video src, tracks)
   */
  private cleanupDirectPlay(): void {
    const video = this.videoElement.nativeElement;

    // Remove tracks
    while (video.firstChild) {
      video.removeChild(video.firstChild);
    }

    // Clear event handlers
    video.onerror = null;
    video.onloadedmetadata = null;

    // Clear video source
    video.removeAttribute('src');
    video.load();
  }

  private initializeDashPlayer(manifestUrl: string, startPosition?: number): void {
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

        // Seek to resume position if provided
        if (startPosition && startPosition > 0 && this.player) {
          this.player.seek(startPosition);
        }
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
    // Clean up DASH player
    if (this.player) {
      this.player.reset();
      this.player = null;
    }

    // Clean up direct play resources
    this.cleanupDirectPlay();
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
    // Only handle clicks directly on video, not on controls
    if (event.target !== this.videoElement.nativeElement) {
      return;
    }

    // Track taps for triple-tap detection
    this.tapCount++;

    if (this.tapTimeout) {
      clearTimeout(this.tapTimeout);
    }

    if (this.tapCount >= 3) {
      // Triple tap - toggle debug panel
      this.toggleDebugPanel();
      this.tapCount = 0;
    } else {
      // Wait to see if more taps are coming
      this.tapTimeout = setTimeout(() => {
        if (this.tapCount === 1) {
          // Single tap - toggle play
          this.togglePlay();
        }
        // Double tap is handled by dblclick for fullscreen
        this.tapCount = 0;
      }, 300);
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
    const newTime = percent * video.duration;

    // In direct-stream mode, backward seek triggers fallback to DASH
    if (this.currentPlaybackMode === 'direct-stream' && newTime < video.currentTime) {
      this.showNotification('Seeking backward, switching to adaptive streaming...');
      this.fallbackToDash(newTime);
      return;
    }

    video.currentTime = newTime;
  }

  seek(seconds: number): void {
    const video = this.videoElement.nativeElement;
    const newTime = Math.max(0, Math.min(video.duration, video.currentTime + seconds));

    // In direct-stream mode, backward seek triggers fallback to DASH
    if (this.currentPlaybackMode === 'direct-stream' && newTime < video.currentTime) {
      this.showNotification('Seeking backward, switching to adaptive streaming...');
      this.fallbackToDash(newTime);
      return;
    }

    video.currentTime = newTime;
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

  /**
   * Toggle debug panel visibility
   */
  toggleDebugPanel(): void {
    const newValue = !this.showDebugPanel();
    this.showDebugPanel.set(newValue);

    // Clear existing timeout
    if (this.debugPanelTimeout) {
      clearTimeout(this.debugPanelTimeout);
      this.debugPanelTimeout = null;
    }

    if (newValue) {
      // Start updating stats
      this.updateDebugStats();
      this.debugStatsInterval = setInterval(() => this.updateDebugStats(), 1000);

      // Auto-hide after 1 minute
      this.debugPanelTimeout = setTimeout(() => {
        this.showDebugPanel.set(false);
        if (this.debugStatsInterval) {
          clearInterval(this.debugStatsInterval);
          this.debugStatsInterval = null;
        }
      }, 60_000);
    } else {
      // Stop updating stats
      if (this.debugStatsInterval) {
        clearInterval(this.debugStatsInterval);
        this.debugStatsInterval = null;
      }
    }
  }

  /**
   * Update debug stats from current playback state
   */
  private updateDebugStats(): void {
    const video = this.videoElement?.nativeElement;
    const media = this.media();

    if (!video || !media) {
      this.debugStats.set(null);
      return;
    }

    // Get buffer ahead
    let bufferAhead = 0;
    const buffered = video.buffered;
    for (let i = 0; i < buffered.length; i++) {
      if (video.currentTime >= buffered.start(i) && video.currentTime <= buffered.end(i)) {
        bufferAhead = buffered.end(i) - video.currentTime;
        break;
      }
    }

    // Get bandwidth from cached measurement
    const bandwidthMeasurement = this.bandwidthService.getCached();
    const bandwidthMbps = bandwidthMeasurement
      ? bandwidthMeasurement.bitsPerSecond / 1_000_000
      : null;

    // Get playback mode label
    let playbackMode: 'Direct Play' | 'Direct Stream' | 'Adaptive';
    if (this.currentPlaybackMode === 'direct') {
      playbackMode = 'Direct Play';
    } else if (this.currentPlaybackMode === 'direct-stream') {
      playbackMode = 'Direct Stream';
    } else {
      playbackMode = 'Adaptive';
    }

    // Get quality info (DASH only)
    let currentQuality: string | null = null;
    let abrEnabled: boolean | null = null;

    if (this.player && this.currentPlaybackMode === 'dash') {
      const settings = this.player.getSettings();
      abrEnabled = settings.streaming?.abr?.autoSwitchBitrate?.video ?? true;

      try {
        const currentRep = this.player.getCurrentRepresentationForType('video');
        if (currentRep) {
          currentQuality = `${currentRep.height}p${abrEnabled ? ' (auto)' : ''}`;
        }
      } catch {
        // Ignore errors getting representation
      }
    } else if (this.currentPlaybackMode === 'direct' || this.currentPlaybackMode === 'direct-stream') {
      // For direct play, show source resolution
      if (media.height) {
        currentQuality = `${media.height}p (source)`;
      }
    }

    // Get stall count from monitor
    const health = this.playbackMonitor.getHealth();

    this.debugStats.set({
      playbackMode,
      bandwidthMbps,
      bufferAhead,
      stallCount: health.stallCount,
      mediaBitrateMbps: media.bitrate / 1_000_000,
      currentQuality,
      abrEnabled
    });
  }
}
