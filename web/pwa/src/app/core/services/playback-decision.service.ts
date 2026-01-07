import { Injectable, inject } from '@angular/core';
import { MediaDetail } from './api.service';
import { PlaybackCapabilityService } from './playback-capability.service';
import { BandwidthService } from './bandwidth.service';

export type PlaybackMode = 'direct' | 'direct-stream' | 'dash';

export interface PlaybackDecision {
  mode: PlaybackMode;
  url: string;
  reason: string;
  fallbackAvailable: boolean;
  bandwidthMbps?: number;
  // For direct-stream mode: which audio track index (in audio_streams array) to use
  selectedAudioTrackIndex?: number;
}

@Injectable({
  providedIn: 'root'
})
export class PlaybackDecisionService {
  private readonly capabilityService = inject(PlaybackCapabilityService);
  private readonly bandwidthService = inject(BandwidthService);

  /**
   * Decide the best playback mode for the given media
   * Follows the decision flow:
   * 1. Check container format
   * 2. Check codec support
   * 3. Check subtitles
   * 4. Check bandwidth
   * 5. Return decision
   */
  async decide(media: MediaDetail): Promise<PlaybackDecision> {
    // Always have DASH as fallback
    const dashUrl = media.dash_manifest_url;
    const fallbackAvailable = !!dashUrl;

    // Check 1: Is direct play/stream even possible server-side?
    if (!media.direct_play_supported && !media.direct_stream_supported) {
      return {
        mode: 'dash',
        url: dashUrl,
        reason: 'Direct play not supported for this media',
        fallbackAvailable: false
      };
    }

    // Check 2: Can the browser decode the codecs?
    const capability = await this.capabilityService.checkCodecSupport(media);

    if (!capability.canDirectPlay && !capability.canDirectStream) {
      return {
        mode: 'dash',
        url: dashUrl,
        reason: capability.reason,
        fallbackAvailable: false
      };
    }

    // Check 3: Bandwidth check
    // First try cached measurement, otherwise measure fresh
    let measurement = this.bandwidthService.getCached();
    if (!measurement) {
      try {
        measurement = await this.bandwidthService.measure();
      } catch (err) {
        console.warn('Bandwidth measurement failed, falling back to DASH:', err);
        return {
          mode: 'dash',
          url: dashUrl,
          reason: 'Could not measure bandwidth',
          fallbackAvailable: false
        };
      }
    }

    const bandwidthMbps = measurement.bitsPerSecond / 1_000_000;

    if (!this.bandwidthService.isSufficient(media.bitrate, measurement)) {
      const requiredMbps = (media.bitrate * 1.3) / 1_000_000;
      return {
        mode: 'dash',
        url: dashUrl,
        reason: `Bandwidth insufficient (${bandwidthMbps.toFixed(1)} Mbps available, ${requiredMbps.toFixed(1)} Mbps required)`,
        fallbackAvailable: false,
        bandwidthMbps
      };
    }

    // All checks passed - determine mode based on container
    if (capability.canDirectPlay && media.direct_play_url) {
      return {
        mode: 'direct',
        url: media.direct_play_url,
        reason: 'Direct play: codec and bandwidth supported',
        fallbackAvailable,
        bandwidthMbps
      };
    }

    if (capability.canDirectStream && media.direct_stream_url) {
      return {
        mode: 'direct-stream',
        url: media.direct_stream_url,
        reason: 'Direct stream: remuxing container',
        fallbackAvailable,
        bandwidthMbps,
        selectedAudioTrackIndex: capability.selectedAudioTrackIndex
      };
    }

    // Shouldn't reach here, but fallback to DASH
    return {
      mode: 'dash',
      url: dashUrl,
      reason: 'No direct play URL available',
      fallbackAvailable: false,
      bandwidthMbps
    };
  }

  /**
   * Quick check without bandwidth measurement (uses cached)
   * Useful for UI indicators before full decision
   */
  async quickCheck(media: MediaDetail): Promise<{
    directPlayPossible: boolean;
    directStreamPossible: boolean;
    reason: string;
  }> {
    if (!media.direct_play_supported && !media.direct_stream_supported) {
      return {
        directPlayPossible: false,
        directStreamPossible: false,
        reason: 'Direct play not supported for this media'
      };
    }

    const capability = await this.capabilityService.checkCodecSupport(media);

    return {
      directPlayPossible: capability.canDirectPlay,
      directStreamPossible: capability.canDirectStream,
      reason: capability.reason
    };
  }
}
