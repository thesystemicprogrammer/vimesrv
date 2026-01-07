import { Injectable } from '@angular/core';
import { Subject, Observable } from 'rxjs';

export interface PlaybackHealth {
  bufferAhead: number;      // Seconds of buffered content ahead of current position
  stallCount: number;       // Number of stalls since monitoring started
  lastStallTime: number;    // Timestamp of last stall (0 if none)
  isHealthy: boolean;       // Overall health assessment
  reason?: string;          // Reason if unhealthy
}

export interface FallbackEvent {
  reason: string;
  stallCount: number;
  bufferAhead: number;
  currentTime: number;
}

export type MonitoringMode = 'direct' | 'direct-stream';

// Thresholds for triggering fallback
const STALL_COUNT_THRESHOLD = 2;           // Fallback after 2 stalls
const LOW_BUFFER_THRESHOLD = 3;            // Buffer below 3 seconds is "low"
const LOW_BUFFER_DURATION_MS = 5000;       // Low buffer for 5+ seconds triggers fallback
const MONITORING_INTERVAL_MS = 1000;       // Check every second

// Direct-stream specific settings
// Fragmented MP4 streams need more time to build initial buffer
const DIRECT_STREAM_GRACE_PERIOD_MS = 10000;  // 10 second grace period for direct-stream
const DIRECT_STREAM_STALL_THRESHOLD = 4;      // Higher stall threshold for direct-stream

@Injectable({
  providedIn: 'root'
})
export class PlaybackMonitorService {
  private videoElement: HTMLVideoElement | null = null;
  private monitoringInterval: any = null;
  private isMonitoring = false;
  private mode: MonitoringMode = 'direct';
  private monitoringStartTime = 0;

  // State tracking
  private stallCount = 0;
  private lastStallTime = 0;
  private lowBufferStartTime = 0;
  private wasWaiting = false;

  // Event emitters
  private fallbackNeededSubject = new Subject<FallbackEvent>();
  private healthUpdateSubject = new Subject<PlaybackHealth>();

  /** Observable that emits when fallback to DASH is recommended */
  readonly fallbackNeeded$: Observable<FallbackEvent> = this.fallbackNeededSubject.asObservable();

  /** Observable that emits health updates during monitoring */
  readonly healthUpdate$: Observable<PlaybackHealth> = this.healthUpdateSubject.asObservable();

  /**
   * Start monitoring a video element for playback issues
   * @param video The video element to monitor
   * @param mode The playback mode - affects thresholds and grace period
   */
  startMonitoring(video: HTMLVideoElement, mode: MonitoringMode = 'direct'): void {
    if (this.isMonitoring) {
      this.stopMonitoring();
    }

    this.videoElement = video;
    this.isMonitoring = true;
    this.mode = mode;
    this.monitoringStartTime = Date.now();
    this.resetState();

    // Listen for stall events
    video.addEventListener('waiting', this.onWaiting);
    video.addEventListener('playing', this.onPlaying);
    video.addEventListener('stalled', this.onStalled);

    // Start periodic health checks
    this.monitoringInterval = setInterval(() => this.checkHealth(), MONITORING_INTERVAL_MS);

    console.log(`Playback monitoring started (mode: ${mode})`);
  }

  /**
   * Stop monitoring and clean up
   */
  stopMonitoring(): void {
    if (!this.isMonitoring) return;

    if (this.videoElement) {
      this.videoElement.removeEventListener('waiting', this.onWaiting);
      this.videoElement.removeEventListener('playing', this.onPlaying);
      this.videoElement.removeEventListener('stalled', this.onStalled);
    }

    if (this.monitoringInterval) {
      clearInterval(this.monitoringInterval);
      this.monitoringInterval = null;
    }

    this.videoElement = null;
    this.isMonitoring = false;
    this.resetState();

    console.log('Playback monitoring stopped');
  }

  /**
   * Get current playback health status
   */
  getHealth(): PlaybackHealth {
    if (!this.videoElement) {
      return {
        bufferAhead: 0,
        stallCount: 0,
        lastStallTime: 0,
        isHealthy: true
      };
    }

    const bufferAhead = this.getBufferAhead();
    const stallThreshold = this.getStallThreshold();
    const isHealthy = this.stallCount < stallThreshold && bufferAhead >= LOW_BUFFER_THRESHOLD;

    return {
      bufferAhead,
      stallCount: this.stallCount,
      lastStallTime: this.lastStallTime,
      isHealthy,
      reason: isHealthy ? undefined : this.getUnhealthyReason(bufferAhead)
    };
  }

  /**
   * Reset monitoring state (call when switching playback modes)
   */
  resetState(): void {
    this.stallCount = 0;
    this.lastStallTime = 0;
    this.lowBufferStartTime = 0;
    this.wasWaiting = false;
  }

  /**
   * Check if we're still in the grace period for direct-stream mode
   */
  private isInGracePeriod(): boolean {
    if (this.mode !== 'direct-stream') {
      return false;
    }
    return Date.now() - this.monitoringStartTime < DIRECT_STREAM_GRACE_PERIOD_MS;
  }

  /**
   * Get the stall threshold based on mode
   */
  private getStallThreshold(): number {
    return this.mode === 'direct-stream' ? DIRECT_STREAM_STALL_THRESHOLD : STALL_COUNT_THRESHOLD;
  }

  // Event handlers (arrow functions to preserve 'this')
  private onWaiting = (): void => {
    if (!this.wasWaiting) {
      this.wasWaiting = true;
      this.stallCount++;
      this.lastStallTime = Date.now();
      
      const inGrace = this.isInGracePeriod();
      console.log(`Playback stall detected (count: ${this.stallCount}, grace period: ${inGrace})`);

      // Don't trigger fallback during grace period
      if (!inGrace) {
        this.checkFallbackThresholds();
      }
    }
  };

  private onPlaying = (): void => {
    this.wasWaiting = false;
  };

  private onStalled = (): void => {
    // 'stalled' event means browser is trying to fetch but not receiving data
    console.log('Network stall detected');
  };

  private checkHealth(): void {
    if (!this.videoElement || !this.isMonitoring) return;

    const bufferAhead = this.getBufferAhead();
    const now = Date.now();

    // Skip low buffer checks during grace period
    if (this.isInGracePeriod()) {
      // Just emit health update, don't check thresholds
      const health = this.getHealth();
      this.healthUpdateSubject.next(health);
      return;
    }

    // Track how long we've been in low buffer state
    if (bufferAhead < LOW_BUFFER_THRESHOLD && !this.videoElement.paused) {
      if (this.lowBufferStartTime === 0) {
        this.lowBufferStartTime = now;
      } else if (now - this.lowBufferStartTime >= LOW_BUFFER_DURATION_MS) {
        // Low buffer for too long - trigger fallback
        this.triggerFallback(`Low buffer (${bufferAhead.toFixed(1)}s) for ${LOW_BUFFER_DURATION_MS / 1000}+ seconds`);
        return;
      }
    } else {
      this.lowBufferStartTime = 0;
    }

    // Emit health update
    const health = this.getHealth();
    this.healthUpdateSubject.next(health);
  }

  private checkFallbackThresholds(): void {
    const stallThreshold = this.getStallThreshold();
    if (this.stallCount >= stallThreshold) {
      this.triggerFallback(`Too many stalls (${this.stallCount})`);
    }
  }

  private triggerFallback(reason: string): void {
    if (!this.videoElement) return;

    const event: FallbackEvent = {
      reason,
      stallCount: this.stallCount,
      bufferAhead: this.getBufferAhead(),
      currentTime: this.videoElement.currentTime
    };

    console.log('Fallback triggered:', event);
    this.fallbackNeededSubject.next(event);

    // Stop monitoring after triggering fallback
    this.stopMonitoring();
  }

  private getBufferAhead(): number {
    if (!this.videoElement) return 0;

    const video = this.videoElement;
    const currentTime = video.currentTime;
    const buffered = video.buffered;

    // Find the buffer range that contains current position
    for (let i = 0; i < buffered.length; i++) {
      const start = buffered.start(i);
      const end = buffered.end(i);

      if (currentTime >= start && currentTime <= end) {
        return end - currentTime;
      }
    }

    return 0;
  }

  private getUnhealthyReason(bufferAhead: number): string {
    const stallThreshold = this.getStallThreshold();
    if (this.stallCount >= stallThreshold) {
      return `${this.stallCount} playback stalls detected`;
    }
    if (bufferAhead < LOW_BUFFER_THRESHOLD) {
      return `Low buffer: ${bufferAhead.toFixed(1)}s`;
    }
    return 'Unknown issue';
  }
}
