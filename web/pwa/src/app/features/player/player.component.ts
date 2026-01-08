import { Component, OnInit, OnDestroy, ViewChild, ElementRef, signal, inject, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { ApiService, MediaDetail } from '../../core/services/api.service';
import { PlaybackDecisionService } from '../../core/services/playback-decision.service';
import { PlaybackEngineService } from './services/playback-engine.service';
import { WatchProgressService } from '../../core/services/watch-progress.service';
import { PlayerDebugPanelComponent } from './components/player-debug-panel.component';
import { PlayerCenterControlsComponent } from './components/player-center-controls.component';
import { PlayerBottomControlsComponent } from './components/player-bottom-controls.component';
import { PlayerSettingsSheetComponent } from './components/player-settings-sheet.component';

/** Time in milliseconds before player controls auto-hide after user interaction */
const CONTROLS_HIDE_TIMEOUT_MS = 6000;

@Component({
  selector: 'app-player',
  standalone: true,
  imports: [
    CommonModule,
    RouterModule,
    PlayerDebugPanelComponent,
    PlayerCenterControlsComponent,
    PlayerBottomControlsComponent,
    PlayerSettingsSheetComponent
  ],
  providers: [PlaybackEngineService],
  templateUrl: './player.component.html',
  styleUrl: './player.component.css'
})
export class PlayerComponent implements OnInit, OnDestroy {
  @ViewChild('videoElement', { static: true }) videoElement!: ElementRef<HTMLVideoElement>;
  @ViewChild('playerContainer') playerContainer!: ElementRef<HTMLDivElement>;

  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly api = inject(ApiService);
  private readonly playbackDecision = inject(PlaybackDecisionService);
  private readonly watchProgress = inject(WatchProgressService);
  
  // Inject playback engine (provided at component level)
  readonly playbackEngine = inject(PlaybackEngineService);

  // UI state signals
  loading = signal(true);
  error = signal<string | null>(null);
  media = signal<MediaDetail | null>(null);
  notificationMessage = signal<string | null>(null);
  showControls = signal(true);
  isFullscreen = signal(false);
  showDebugPanel = signal(false);
  showSettingsSheet = signal(false);

  // Touch/click handling
  private tapCount = 0;
  private tapTimeout: any = null;
  private controlsTimeout: any = null;
  private notificationTimeout: any = null;
  private debugPanelTimeout: any = null;
  
  // Progress tracking
  private progressInterval: any = null;
  private currentMediaId: string | null = null;
  private currentEpisodeId: number | null = null;

  @HostListener('document:keydown', ['$event'])
  onKeyDown(event: KeyboardEvent): void {
    if (event.target instanceof HTMLInputElement || event.target instanceof HTMLSelectElement) {
      return;
    }
    
    switch (event.code) {
      case 'Space':
        event.preventDefault();
        this.playbackEngine.togglePlay();
        break;
      case 'ArrowLeft':
        event.preventDefault();
        this.playbackEngine.seek(-10);
        break;
      case 'ArrowRight':
        event.preventDefault();
        this.playbackEngine.seek(10);
        break;
      case 'ArrowUp':
        event.preventDefault();
        this.playbackEngine.adjustVolume(0.1);
        break;
      case 'ArrowDown':
        event.preventDefault();
        this.playbackEngine.adjustVolume(-0.1);
        break;
      case 'KeyM':
        event.preventDefault();
        this.playbackEngine.toggleMute();
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
    this.playbackEngine.destroy();
    
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
    if (this.progressInterval) {
      clearInterval(this.progressInterval);
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

      // Get stream token
      const tokenResponse = await this.api.getStreamToken().toPromise();
      if (!tokenResponse?.data?.token) {
        throw new Error('Failed to get stream token');
      }
      const streamToken = tokenResponse.data.token;

      // Make playback decision
      const decision = await this.playbackDecision.decide(media);
      console.log('Playback decision:', decision);

      // Initialize playback engine
      await this.playbackEngine.initialize(this.videoElement.nativeElement, {
        streamToken,
        media,
        decision,
        onFallbackNeeded: (resumePosition) => {
          // Handled internally by the engine
        },
        onNotification: (message, duration) => {
          this.showNotification(message, duration);
        },
        onLoadingChange: (loading) => {
          this.loading.set(loading);
        },
        onError: (error) => {
          this.error.set(error);
        }
      });

      // Set up watch progress tracking
      this.setupWatchProgress(media);
      
    } catch (err) {
      console.error('Failed to load media:', err);
      this.error.set(err instanceof Error ? err.message : 'Failed to load media');
      this.loading.set(false);
    }
  }

  /**
   * Set up watch progress tracking for the current media
   * - Restore previous position
   * - Start periodic progress saving
   */
  private setupWatchProgress(media: MediaDetail): void {
    // Store media identifier for progress tracking
    // Use the media_id (media_files.id) for both movies and episodes
    this.currentMediaId = media.id;
    this.currentEpisodeId = null; // Not needed - backend looks it up from media_files

    // Restore previous watch position
    this.watchProgress.getProgress(
      this.currentMediaId, 
      undefined
    ).subscribe({
      next: (response) => {
        if (response.data && response.data.position_seconds > 10) {
          // Only restore if position is > 10 seconds (skip intro resume)
          const video = this.videoElement.nativeElement;
          video.currentTime = response.data.position_seconds;
          console.log('Restored watch position:', response.data.position_seconds);
        }
      },
      error: (err) => {
        console.log('No previous watch progress found:', err);
      }
    });

    // Start periodic progress saving (every 10 seconds)
    this.startProgressTracking();
  }

  /**
   * Start interval to save progress every 10 seconds during playback
   */
  private startProgressTracking(): void {
    if (this.progressInterval) {
      clearInterval(this.progressInterval);
    }

    this.progressInterval = setInterval(() => {
      const video = this.videoElement.nativeElement;
      
      // Only save if video is playing and has valid duration
      if (!video.paused && video.duration > 0 && !isNaN(video.duration)) {
        this.saveProgress();
      }
    }, 10000); // Every 10 seconds
  }

  /**
   * Save current playback progress
   */
  private saveProgress(): void {
    const video = this.videoElement.nativeElement;
    
    if (!video.duration || isNaN(video.duration)) {
      return;
    }

    const position = Math.floor(video.currentTime);
    const duration = Math.floor(video.duration);

    // Only save if we have valid data
    if (position <= 0 || duration <= 0) {
      return;
    }

    console.log('Saving watch progress:', {
      media_id: this.currentMediaId,
      position_seconds: position,
      duration_seconds: duration
    });

    this.watchProgress.recordProgress({
      media_id: this.currentMediaId || undefined,
      episode_metadata_id: undefined, // Backend looks this up from media_files
      position_seconds: position,
      duration_seconds: duration
    }).subscribe({
      next: () => {
        console.log('Watch progress saved successfully');
      },
      error: (err) => {
        console.error('Failed to save watch progress:', err);
      }
    });
  }

  // ==================== UI Event Handlers ====================

  /**
   * Handle clicks/taps on the video container
   * Single tap: toggle controls visibility
   * Triple tap: toggle debug panel
   */
  onContainerClick(event: Event): void {
    const target = event.target as HTMLElement;
    
    // Only handle clicks on the container or video element directly
    // Ignore clicks on child components (controls, buttons, etc.)
    if (target !== this.playerContainer?.nativeElement && 
        target !== this.videoElement?.nativeElement) {
      return;
    }

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
          // Single tap - toggle controls visibility (NOT play/pause)
          this.toggleControlsVisibility();
        }
        // Double tap is handled by dblclick for fullscreen
        this.tapCount = 0;
      }, 300);
    }
  }

  onMouseMove(): void {
    this.showControls.set(true);
    this.resetControlsTimeout();
  }

  private toggleControlsVisibility(): void {
    this.showControls.update(v => !v);
    if (this.showControls()) {
      this.resetControlsTimeout();
    }
  }

  private resetControlsTimeout(): void {
    if (this.controlsTimeout) {
      clearTimeout(this.controlsTimeout);
    }
    
    this.controlsTimeout = setTimeout(() => {
      if (this.playbackEngine.isPlaying()) {
        this.showControls.set(false);
      }
    }, CONTROLS_HIDE_TIMEOUT_MS);
  }

  // ==================== Playback Control Handlers ====================

  onPlayPause(): void {
    this.playbackEngine.togglePlay();
  }

  onSkip(seconds: number): void {
    const sign = seconds > 0 ? '+' : '';
    this.showNotification(`${sign}${seconds}s`);
    this.playbackEngine.seek(seconds);
  }

  onSeekPercent(percent: number): void {
    this.playbackEngine.seekToPercent(percent);
  }

  toggleFullscreen(): void {
    if (!document.fullscreenElement) {
      this.playerContainer.nativeElement.requestFullscreen();
    } else {
      document.exitFullscreen();
    }
  }

  // ==================== Settings Sheet ====================

  openSettingsSheet(): void {
    this.showSettingsSheet.set(true);
    // Keep controls visible while settings are open
    if (this.controlsTimeout) {
      clearTimeout(this.controlsTimeout);
    }
  }

  closeSettingsSheet(): void {
    this.showSettingsSheet.set(false);
    this.resetControlsTimeout();
  }

  // ==================== Debug Panel ====================

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
      this.playbackEngine.startDebugStats();

      // Auto-hide after 1 minute
      this.debugPanelTimeout = setTimeout(() => {
        this.showDebugPanel.set(false);
        this.playbackEngine.stopDebugStats();
      }, 60_000);
    } else {
      // Stop updating stats
      this.playbackEngine.stopDebugStats();
    }
  }

  // ==================== Notifications ====================

  private showNotification(message: string, durationMs: number = 3000): void {
    this.notificationMessage.set(message);

    if (this.notificationTimeout) {
      clearTimeout(this.notificationTimeout);
    }

    this.notificationTimeout = setTimeout(() => {
      this.notificationMessage.set(null);
    }, durationMs);
  }
}
