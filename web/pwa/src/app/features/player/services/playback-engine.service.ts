import { Injectable, OnDestroy, signal, computed } from '@angular/core';
import { Subscription } from 'rxjs';
import * as dashjs from 'dashjs';
import { MediaDetail, AudioStream, SubtitleStream } from '../../../core/services/api.service';
import { PlaybackMode, PlaybackDecision } from '../../../core/services/playback-decision.service';
import { PlaybackMonitorService, FallbackEvent, MonitoringMode } from '../../../core/services/playback-monitor.service';
import { BandwidthService } from '../../../core/services/bandwidth.service';

export interface QualityLevel {
  index: number;       // Original index from dash.js getRepresentationsByType()
  id: string;          // Representation ID
  height: number;
  bandwidth: number;
}

export interface DebugStats {
  playbackMode: 'Direct Play' | 'Direct Stream' | 'Adaptive';
  bandwidthMbps: number | null;
  bufferAhead: number;
  stallCount: number;
  mediaBitrateMbps: number;
  currentQuality: string | null;
  abrEnabled: boolean | null;
}

export interface PlaybackConfig {
  streamToken: string;
  media: MediaDetail;
  decision: PlaybackDecision;
  onFallbackNeeded: (resumePosition?: number) => void;
  onNotification: (message: string, duration?: number) => void;
  onLoadingChange: (loading: boolean) => void;
  onError: (error: string | null) => void;
}

/**
 * PlaybackEngineService handles all playback logic including:
 * - Direct player initialization/destruction
 * - DASH player initialization/destruction
 * - Playback state management
 * - Quality/track management
 * - Playback monitoring and fallback handling
 * 
 * This service is provided at the component level (not root) so each player
 * instance gets its own service instance that is destroyed with the component.
 */
@Injectable()
export class PlaybackEngineService implements OnDestroy {
  // Playback state signals (readonly externally)
  readonly isPlaying = signal(false);
  readonly currentTime = signal(0);
  readonly duration = signal(0);
  readonly progress = signal(0);
  readonly volume = signal(1);
  readonly isMuted = signal(false);
  readonly buffered = signal(0);
  
  // Track signals
  readonly audioTracks = signal<AudioStream[]>([]);
  readonly subtitleTracks = signal<SubtitleStream[]>([]);
  readonly currentAudioIndex = signal(0);
  readonly currentSubtitleIndex = signal(-1);
  
  // Quality signals
  readonly qualityLevels = signal<QualityLevel[]>([]);
  readonly currentQualityIndex = signal(-1); // -1 = auto
  
  // Mode signals
  readonly playbackMode = signal<PlaybackMode>('dash');
  readonly playbackModeLabel = signal<string>('');
  
  // Debug stats
  readonly debugStats = signal<DebugStats | null>(null);
  
  // Computed stats for buffer
  readonly bufferAhead = computed(() => {
    return this.buffered();
  });
  
  // Internal state
  private videoElement: HTMLVideoElement | null = null;
  private player: dashjs.MediaPlayerClass | null = null;
  private streamToken: string | null = null;
  private media: MediaDetail | null = null;
  private config: PlaybackConfig | null = null;
  
  // Direct stream state
  private directStreamBaseUrl: string | null = null;
  private selectedAudioStreamIndex: number = 0;
  
  // DASH pre-warming
  private manifestPrewarmPromise: Promise<void> | null = null;
  
  // Monitoring
  private fallbackSubscription: Subscription | null = null;
  private debugStatsInterval: any = null;
  
  constructor(
    private playbackMonitor: PlaybackMonitorService,
    private bandwidthService: BandwidthService
  ) {}
  
  ngOnDestroy(): void {
    this.destroy();
  }
  
  /**
   * Initialize the playback engine with video element and configuration
   */
  async initialize(videoElement: HTMLVideoElement, config: PlaybackConfig): Promise<void> {
    this.videoElement = videoElement;
    this.config = config;
    this.streamToken = config.streamToken;
    this.media = config.media;
    
    // Set tracks from media
    this.audioTracks.set(config.media.audio_streams || []);
    this.subtitleTracks.set(config.media.subtitle_streams || []);
    
    // Set up video event listeners
    this.setupVideoEventListeners();
    
    // Handle playback based on decision
    const decision = config.decision;
    this.playbackMode.set(decision.mode);
    
    if (decision.mode === 'direct' || decision.mode === 'direct-stream') {
      // Pre-warm DASH manifest in background for faster fallback
      this.prewarmDashManifest(config.media.dash_manifest_url);
      
      // For direct-stream, store the base URL and selected audio track
      if (decision.mode === 'direct-stream') {
        this.directStreamBaseUrl = decision.url;
        const audioTrackArrayIndex = decision.selectedAudioTrackIndex ?? 0;
        const audioStreams = config.media.audio_streams || [];
        if (audioStreams[audioTrackArrayIndex]) {
          this.selectedAudioStreamIndex = audioStreams[audioTrackArrayIndex].index;
        } else {
          this.selectedAudioStreamIndex = audioStreams[0]?.index ?? 0;
        }
        this.currentAudioIndex.set(audioTrackArrayIndex);
      }
      
      // Initialize direct player
      await this.initializeDirectPlayer(decision.url, decision.mode === 'direct-stream');
      this.playbackModeLabel.set(decision.mode === 'direct' ? 'Direct Play' : 'Direct Stream');
    } else {
      // DASH mode
      this.initializeDashPlayer(config.media.dash_manifest_url);
      this.playbackModeLabel.set('Adaptive');
    }
  }
  
  /**
   * Destroy the playback engine and clean up resources
   */
  destroy(): void {
    // Stop debug stats interval
    if (this.debugStatsInterval) {
      clearInterval(this.debugStatsInterval);
      this.debugStatsInterval = null;
    }
    
    // Stop monitoring
    this.playbackMonitor.stopMonitoring();
    if (this.fallbackSubscription) {
      this.fallbackSubscription.unsubscribe();
      this.fallbackSubscription = null;
    }
    
    // Clean up DASH player
    if (this.player) {
      this.player.reset();
      this.player = null;
    }
    
    // Clean up direct play resources
    this.cleanupDirectPlay();
    
    // Clear references
    this.videoElement = null;
    this.config = null;
    this.streamToken = null;
    this.media = null;
  }
  
  // ==================== Playback Controls ====================
  
  togglePlay(): void {
    if (!this.videoElement) return;
    
    if (this.videoElement.paused) {
      this.videoElement.play();
    } else {
      this.videoElement.pause();
    }
  }
  
  play(): void {
    this.videoElement?.play();
  }
  
  pause(): void {
    this.videoElement?.pause();
  }
  
  /**
   * Seek relative to current position
   * @param seconds Positive to seek forward, negative to seek backward
   */
  seek(seconds: number): void {
    if (!this.videoElement) return;
    
    const newTime = Math.max(0, Math.min(this.videoElement.duration, this.videoElement.currentTime + seconds));
    
    // In direct-stream mode, backward seek triggers fallback to DASH
    if (this.playbackMode() === 'direct-stream' && newTime < this.videoElement.currentTime) {
      this.config?.onNotification('Seeking backward, switching to adaptive streaming...');
      this.fallbackToDash(newTime);
      return;
    }
    
    this.videoElement.currentTime = newTime;
  }
  
  /**
   * Seek to an absolute time
   * @param time Time in seconds
   */
  seekTo(time: number): void {
    if (!this.videoElement) return;
    
    const newTime = Math.max(0, Math.min(this.videoElement.duration, time));
    
    // In direct-stream mode, backward seek triggers fallback to DASH
    if (this.playbackMode() === 'direct-stream' && newTime < this.videoElement.currentTime) {
      this.config?.onNotification('Seeking backward, switching to adaptive streaming...');
      this.fallbackToDash(newTime);
      return;
    }
    
    this.videoElement.currentTime = newTime;
  }
  
  /**
   * Seek to a percentage of the duration
   * @param percent Value between 0 and 100
   */
  seekToPercent(percent: number): void {
    if (!this.videoElement || !this.videoElement.duration) return;
    
    const newTime = (percent / 100) * this.videoElement.duration;
    this.seekTo(newTime);
  }
  
  setVolume(volume: number): void {
    if (!this.videoElement) return;
    
    this.videoElement.volume = Math.max(0, Math.min(1, volume));
    if (this.videoElement.volume > 0) {
      this.videoElement.muted = false;
    }
  }
  
  adjustVolume(delta: number): void {
    if (!this.videoElement) return;
    
    this.videoElement.volume = Math.max(0, Math.min(1, this.videoElement.volume + delta));
    if (this.videoElement.volume > 0) {
      this.videoElement.muted = false;
    }
  }
  
  toggleMute(): void {
    if (!this.videoElement) return;
    this.videoElement.muted = !this.videoElement.muted;
  }
  
  // ==================== Track Selection ====================
  
  setAudioTrack(arrayIndex: number): void {
    this.currentAudioIndex.set(arrayIndex);
    
    // Handle direct-stream mode: restart stream with new audio track
    if (this.playbackMode() === 'direct-stream' && this.directStreamBaseUrl) {
      const audioStreams = this.media?.audio_streams || [];
      if (audioStreams[arrayIndex]) {
        this.selectedAudioStreamIndex = audioStreams[arrayIndex].index;
        this.restartDirectStream();
      }
      return;
    }
    
    // DASH mode: use dash.js track switching
    if (this.player) {
      const tracks = this.player.getTracksFor('audio');
      if (tracks[arrayIndex]) {
        this.player.setCurrentTrack(tracks[arrayIndex]);
      }
    }
  }
  
  setSubtitleTrack(index: number): void {
    this.currentSubtitleIndex.set(index);
    
    // For direct play mode, subtitles are handled via <track> elements
    // The component should handle showing/hiding based on currentSubtitleIndex
    
    if (this.player) {
      if (index === -1) {
        this.player.enableText(false);
      } else {
        this.player.enableText(true);
        this.player.setTextTrack(index);
      }
    } else if (this.videoElement) {
      // Direct play mode - handle text tracks on video element
      const tracks = this.videoElement.textTracks;
      for (let i = 0; i < tracks.length; i++) {
        tracks[i].mode = (i === index) ? 'showing' : 'hidden';
      }
    }
  }
  
  setQuality(uiIndex: number): void {
    this.currentQualityIndex.set(uiIndex);
    
    if (!this.player) return;
    
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
      
      const levels = this.qualityLevels();
      if (levels[uiIndex]) {
        this.player.setRepresentationForTypeById('video', levels[uiIndex].id);
      }
    }
  }
  
  // ==================== Debug Panel ====================
  
  startDebugStats(): void {
    this.updateDebugStats();
    this.debugStatsInterval = setInterval(() => this.updateDebugStats(), 1000);
  }
  
  stopDebugStats(): void {
    if (this.debugStatsInterval) {
      clearInterval(this.debugStatsInterval);
      this.debugStatsInterval = null;
    }
    this.debugStats.set(null);
  }
  
  private updateDebugStats(): void {
    if (!this.videoElement || !this.media) {
      this.debugStats.set(null);
      return;
    }
    
    // Get buffer ahead
    let bufferAhead = 0;
    const buffered = this.videoElement.buffered;
    for (let i = 0; i < buffered.length; i++) {
      if (this.videoElement.currentTime >= buffered.start(i) && 
          this.videoElement.currentTime <= buffered.end(i)) {
        bufferAhead = buffered.end(i) - this.videoElement.currentTime;
        break;
      }
    }
    
    // Get bandwidth from cached measurement
    const bandwidthMeasurement = this.bandwidthService.getCached();
    const bandwidthMbps = bandwidthMeasurement
      ? bandwidthMeasurement.bitsPerSecond / 1_000_000
      : null;
    
    // Get playback mode label
    let playbackModeLabel: 'Direct Play' | 'Direct Stream' | 'Adaptive';
    const mode = this.playbackMode();
    if (mode === 'direct') {
      playbackModeLabel = 'Direct Play';
    } else if (mode === 'direct-stream') {
      playbackModeLabel = 'Direct Stream';
    } else {
      playbackModeLabel = 'Adaptive';
    }
    
    // Get quality info (DASH only)
    let currentQuality: string | null = null;
    let abrEnabled: boolean | null = null;
    
    if (this.player && mode === 'dash') {
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
    } else if (mode === 'direct' || mode === 'direct-stream') {
      if (this.media.height) {
        currentQuality = `${this.media.height}p (source)`;
      }
    }
    
    // Get stall count from monitor
    const health = this.playbackMonitor.getHealth();
    
    this.debugStats.set({
      playbackMode: playbackModeLabel,
      bandwidthMbps,
      bufferAhead,
      stallCount: health.stallCount,
      mediaBitrateMbps: this.media.bitrate / 1_000_000,
      currentQuality,
      abrEnabled
    });
  }
  
  // ==================== Private Methods ====================
  
  private setupVideoEventListeners(): void {
    if (!this.videoElement) return;
    
    this.videoElement.addEventListener('timeupdate', () => this.onTimeUpdate());
    this.videoElement.addEventListener('play', () => this.onPlayStateChange());
    this.videoElement.addEventListener('pause', () => this.onPlayStateChange());
    this.videoElement.addEventListener('volumechange', () => this.onVolumeChange());
    this.videoElement.addEventListener('loadedmetadata', () => this.onMetadataLoaded());
  }
  
  private onTimeUpdate(): void {
    if (!this.videoElement) return;
    
    this.currentTime.set(this.videoElement.currentTime);
    if (this.videoElement.duration > 0) {
      this.progress.set((this.videoElement.currentTime / this.videoElement.duration) * 100);
    }
    
    // Update buffered amount
    this.updateBuffered();
  }
  
  private updateBuffered(): void {
    if (!this.videoElement) return;
    
    const buffered = this.videoElement.buffered;
    let bufferAhead = 0;
    for (let i = 0; i < buffered.length; i++) {
      if (this.videoElement.currentTime >= buffered.start(i) && 
          this.videoElement.currentTime <= buffered.end(i)) {
        bufferAhead = buffered.end(i) - this.videoElement.currentTime;
        break;
      }
    }
    this.buffered.set(bufferAhead);
  }
  
  private onPlayStateChange(): void {
    if (!this.videoElement) return;
    this.isPlaying.set(!this.videoElement.paused);
  }
  
  private onVolumeChange(): void {
    if (!this.videoElement) return;
    this.volume.set(this.videoElement.volume);
    this.isMuted.set(this.videoElement.muted);
  }
  
  private onMetadataLoaded(): void {
    if (!this.videoElement) return;
    this.duration.set(this.videoElement.duration);
  }
  
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
  
  private async initializeDirectPlayer(url: string, appendAudioParam: boolean = false): Promise<void> {
    if (!this.videoElement || !this.streamToken) return;
    
    try {
      // Build URL with stream token as query param
      let directUrl = url;
      const separator = directUrl.includes('?') ? '&' : '?';
      directUrl = `${directUrl}${separator}token=${this.streamToken}`;
      
      // For direct-stream (remux), append the audio track parameter
      if (appendAudioParam) {
        directUrl = `${directUrl}&audio=${this.selectedAudioStreamIndex}`;
      }
      
      console.log('Direct player URL:', directUrl);
      
      // Set video source
      this.videoElement.src = directUrl;
      
      this.videoElement.onerror = () => {
        console.error('Direct playback error:', this.videoElement?.error);
        this.fallbackToDash();
      };
      
      this.videoElement.onloadedmetadata = () => {
        this.config?.onLoadingChange(false);
        this.loadSubtitlesForDirectPlay();
        this.startPlaybackMonitoring();
      };
      
      // Start playback
      try {
        await this.videoElement.play();
      } catch (playErr) {
        console.warn('Autoplay blocked:', playErr);
        this.config?.onLoadingChange(false);
      }
      
    } catch (err) {
      console.error('Direct playback initialization failed:', err);
      this.fallbackToDash();
    }
  }
  
  private loadSubtitlesForDirectPlay(): void {
    if (!this.videoElement || !this.media?.subtitle_streams?.length) return;
    
    // Remove any existing tracks
    while (this.videoElement.firstChild) {
      this.videoElement.removeChild(this.videoElement.firstChild);
    }
    
    // Add track elements for each subtitle
    this.media.subtitle_streams.forEach((sub, idx) => {
      const track = document.createElement('track');
      track.kind = 'subtitles';
      track.label = sub.title || sub.language || `Track ${idx + 1}`;
      track.srclang = sub.language || 'und';
      
      const baseUrl = `/stream/dash/content/${this.media!.id}/subtitle-${sub.index}.vtt`;
      track.src = this.streamToken ? `${baseUrl}?token=${this.streamToken}` : baseUrl;
      
      this.videoElement!.appendChild(track);
    });
  }
  
  private startPlaybackMonitoring(): void {
    if (!this.videoElement) return;
    
    // Subscribe to fallback events
    this.fallbackSubscription = this.playbackMonitor.fallbackNeeded$.subscribe(
      (event: FallbackEvent) => {
        console.log('Fallback triggered by monitor:', event);
        this.handleMonitorFallback(event);
      }
    );
    
    // Start monitoring with appropriate mode
    const monitorMode: MonitoringMode = this.playbackMode() === 'direct-stream' ? 'direct-stream' : 'direct';
    this.playbackMonitor.startMonitoring(this.videoElement, monitorMode);
  }
  
  private handleMonitorFallback(event: FallbackEvent): void {
    const currentPosition = event.currentTime;
    this.config?.onNotification('Switching to adaptive streaming...');
    this.fallbackToDash(currentPosition);
  }
  
  private async fallbackToDash(resumePosition?: number): Promise<void> {
    if (!this.media?.dash_manifest_url) {
      this.config?.onError('Playback failed and no fallback available');
      this.config?.onLoadingChange(false);
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
    this.playbackMode.set('dash');
    this.playbackModeLabel.set('Adaptive');
    this.initializeDashPlayer(this.media.dash_manifest_url, resumePosition);
  }
  
  private cleanupDirectPlay(): void {
    if (!this.videoElement) return;
    
    // Remove tracks
    while (this.videoElement.firstChild) {
      this.videoElement.removeChild(this.videoElement.firstChild);
    }
    
    // Clear event handlers
    this.videoElement.onerror = null;
    this.videoElement.onloadedmetadata = null;
    
    // Clear video source
    this.videoElement.removeAttribute('src');
    this.videoElement.load();
  }
  
  private initializeDashPlayer(manifestUrl: string, startPosition?: number): void {
    if (!this.videoElement) return;
    
    try {
      this.player = dashjs.MediaPlayer().create();
      
      // Configure player to send Authorization header with stream token
      if (this.streamToken) {
        const streamToken = this.streamToken;
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
        this.config?.onLoadingChange(false);
        this.updateQualityLevels();
        
        // Seek to resume position if provided
        if (startPosition && startPosition > 0 && this.player) {
          this.player.seek(startPosition);
        }
      });
      
      this.player.on(dashjs.MediaPlayer.events.ERROR, (e: any) => {
        console.error('DASH player error:', e);
        const errorMsg = e.error?.message || e.error?.code || e.event?.type || 'Unknown error';
        const errorDetail = e.error?.data?.response?.status ? ` (HTTP ${e.error.data.response.status})` : '';
        this.config?.onError('Playback error: ' + errorMsg + errorDetail);
      });
      
      this.player.on(dashjs.MediaPlayer.events.QUALITY_CHANGE_RENDERED, (e: any) => {
        if (e.mediaType === 'video') {
          this.updateCurrentQuality();
        }
      });
      
      // Initialize player with video element and manifest
      this.player.initialize(
        this.videoElement,
        manifestUrl,
        true // autoplay
      );
      
    } catch (err) {
      console.error('Failed to initialize player:', err);
      this.config?.onError('Failed to initialize video player');
      this.config?.onLoadingChange(false);
    }
  }
  
  private updateQualityLevels(): void {
    if (!this.player) return;
    
    try {
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
  }
  
  private async restartDirectStream(): Promise<void> {
    if (!this.directStreamBaseUrl || !this.streamToken || !this.videoElement) return;
    
    const currentTime = this.videoElement.currentTime;
    const wasPlaying = !this.videoElement.paused;
    
    console.log(`Restarting direct stream with audio=${this.selectedAudioStreamIndex} at ${currentTime}s`);
    
    this.config?.onNotification('Switching audio track...', 2000);
    
    // Build new URL with updated audio param
    let directUrl = this.directStreamBaseUrl;
    const separator = directUrl.includes('?') ? '&' : '?';
    directUrl = `${directUrl}${separator}token=${this.streamToken}&audio=${this.selectedAudioStreamIndex}`;
    
    // Set new source and seek to previous position
    this.videoElement.src = directUrl;
    
    this.videoElement.onloadedmetadata = () => {
      if (this.videoElement) {
        this.videoElement.currentTime = currentTime;
        if (wasPlaying) {
          this.videoElement.play().catch(err => console.warn('Autoplay after audio switch blocked:', err));
        }
      }
    };
    
    this.videoElement.load();
  }
  
  // ==================== Utility Methods ====================
  
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
  
  formatBitrate(bitrate: number): string {
    if (bitrate >= 1000000) {
      return (bitrate / 1000000).toFixed(1) + ' Mbps';
    }
    return (bitrate / 1000).toFixed(0) + ' Kbps';
  }
}
