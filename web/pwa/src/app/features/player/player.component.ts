import { Component, OnInit, OnDestroy, ViewChild, ElementRef, signal, inject, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { ApiService, MediaDetail } from '../../core/services/api.service';
import { PlaybackDecisionService } from '../../core/services/playback-decision.service';
import { PlaybackEngineService } from './services/playback-engine.service';
import { PlayerDebugPanelComponent } from './components/player-debug-panel.component';
import { PlayerCenterControlsComponent } from './components/player-center-controls.component';
import { PlayerBottomControlsComponent } from './components/player-bottom-controls.component';

@Component({
  selector: 'app-player',
  standalone: true,
  imports: [
    CommonModule,
    RouterModule,
    PlayerDebugPanelComponent,
    PlayerCenterControlsComponent,
    PlayerBottomControlsComponent
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

  // Touch/click handling
  private tapCount = 0;
  private tapTimeout: any = null;
  private controlsTimeout: any = null;
  private notificationTimeout: any = null;
  private debugPanelTimeout: any = null;

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
      
    } catch (err) {
      console.error('Failed to load media:', err);
      this.error.set(err instanceof Error ? err.message : 'Failed to load media');
      this.loading.set(false);
    }
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
    }, 3000);
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
