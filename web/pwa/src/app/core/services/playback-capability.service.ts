import { Injectable } from '@angular/core';
import { MediaDetail, SubtitleStream } from './api.service';

export interface PlaybackCapability {
  canDirectPlay: boolean;
  canDirectStream: boolean;
  reason: string;
}

// Codec string mappings for MediaCapabilities API
const VIDEO_CODEC_STRINGS: Record<string, string> = {
  'h264': 'avc1.640028',        // H.264 High Profile Level 4.0
  'avc1': 'avc1.640028',
  'avc': 'avc1.640028',
  'hevc': 'hev1.1.6.L93.B0',    // HEVC Main Profile Level 3.1
  'h265': 'hev1.1.6.L93.B0',
  'av1': 'av01.0.08M.08',       // AV1 Main Profile Level 4.0
  'vp9': 'vp09.00.10.08',       // VP9 Profile 0 Level 1.0
  'vp8': 'vp8',
};

const AUDIO_CODEC_STRINGS: Record<string, string> = {
  'aac': 'mp4a.40.2',           // AAC-LC
  'mp3': 'mp3',
  'ac3': 'ac-3',                // Dolby Digital
  'eac3': 'ec-3',               // Dolby Digital Plus
  'opus': 'opus',
  'vorbis': 'vorbis',
  'flac': 'flac',
  // Note: DTS and TrueHD are not supported by browsers
};

// Image-based subtitle formats (cannot be rendered by browser)
const IMAGE_BASED_SUBTITLE_FORMATS = ['pgs', 'hdmv_pgs_subtitle', 'dvdsub', 'vobsub', 'dvd_subtitle'];

@Injectable({
  providedIn: 'root'
})
export class PlaybackCapabilityService {

  /**
   * Check if the browser can decode the media for direct play
   */
  async checkCodecSupport(media: MediaDetail): Promise<PlaybackCapability> {
    // Check 1: Container format
    if (!this.isDirectPlayableContainer(media.format) &&
        !this.isRemuxableContainer(media.format)) {
      return {
        canDirectPlay: false,
        canDirectStream: false,
        reason: `Container format "${media.format}" not supported for direct play`
      };
    }

    // Check 2: Video codec support
    const videoSupported = await this.canDecodeVideo(
      media.video_codec,
      media.width,
      media.height,
      media.bitrate
    );
    if (!videoSupported) {
      return {
        canDirectPlay: false,
        canDirectStream: false,
        reason: `Video codec "${media.video_codec}" not supported by browser`
      };
    }

    // Check 3: Audio codec support (at least one must be supported)
    const audioSupported = await this.canDecodeAnyAudio(media.audio_codecs || []);
    if (!audioSupported) {
      return {
        canDirectPlay: false,
        canDirectStream: false,
        reason: `No supported audio codec found (available: ${(media.audio_codecs || []).join(', ')})`
      };
    }

    // Check 4: Subtitle format (image-based subs require transcoding)
    if (this.hasImageBasedSubtitles(media.subtitle_streams)) {
      return {
        canDirectPlay: false,
        canDirectStream: false,
        reason: 'Media contains image-based subtitles (PGS/VOBSUB) which require transcoding'
      };
    }

    // All checks passed
    const canDirectPlay = this.isDirectPlayableContainer(media.format);
    const canDirectStream = this.isRemuxableContainer(media.format);

    return {
      canDirectPlay,
      canDirectStream,
      reason: canDirectPlay ? 'Direct play supported' : 'Remux required for container'
    };
  }

  /**
   * Check if container supports direct play (no remux needed)
   */
  private isDirectPlayableContainer(format: string): boolean {
    if (!format) return false;
    const normalized = format.toLowerCase();
    return normalized === 'mp4' || normalized === 'mov' || normalized === 'm4v';
  }

  /**
   * Check if container can be remuxed for direct stream
   */
  private isRemuxableContainer(format: string): boolean {
    if (!format) return false;
    const normalized = format.toLowerCase();
    return normalized === 'mkv' || normalized === 'matroska' ||
           normalized === 'avi' || normalized === 'webm';
  }

  /**
   * Check video codec support using MediaCapabilities API
   */
  private async canDecodeVideo(
    codec: string,
    width: number,
    height: number,
    bitrate: number
  ): Promise<boolean> {
    if (!codec) return false;

    const codecString = VIDEO_CODEC_STRINGS[codec.toLowerCase()];
    if (!codecString) {
      console.warn(`Unknown video codec: ${codec}`);
      return false;
    }

    // Use MediaCapabilities API if available
    if ('mediaCapabilities' in navigator) {
      try {
        const config: MediaDecodingConfiguration = {
          type: 'file',
          video: {
            contentType: `video/mp4; codecs="${codecString}"`,
            width: width || 1920,
            height: height || 1080,
            bitrate: bitrate || 5_000_000,
            framerate: 30
          }
        };

        const result = await navigator.mediaCapabilities.decodingInfo(config);
        return result.supported && result.smooth;
      } catch (err) {
        console.warn('MediaCapabilities check failed:', err);
      }
    }

    // Fallback to canPlayType
    const video = document.createElement('video');
    const canPlay = video.canPlayType(`video/mp4; codecs="${codecString}"`);
    return canPlay === 'probably' || canPlay === 'maybe';
  }

  /**
   * Check if at least one audio codec is supported
   */
  private async canDecodeAnyAudio(codecs: string[]): Promise<boolean> {
    if (!codecs || codecs.length === 0) {
      // If no audio codecs specified, assume it's fine
      return true;
    }

    for (const codec of codecs) {
      const codecString = AUDIO_CODEC_STRINGS[codec.toLowerCase()];
      if (!codecString) {
        continue; // Unknown codec, skip
      }

      // Use MediaCapabilities API if available
      if ('mediaCapabilities' in navigator) {
        try {
          const config: MediaDecodingConfiguration = {
            type: 'file',
            audio: {
              contentType: `audio/mp4; codecs="${codecString}"`,
              channels: '2',
              bitrate: 128000,
              samplerate: 48000
            }
          };

          const result = await navigator.mediaCapabilities.decodingInfo(config);
          if (result.supported) {
            return true;
          }
        } catch (err) {
          console.warn('MediaCapabilities audio check failed:', err);
        }
      }

      // Fallback to canPlayType
      const audio = document.createElement('audio');
      const canPlay = audio.canPlayType(`audio/mp4; codecs="${codecString}"`);
      if (canPlay === 'probably' || canPlay === 'maybe') {
        return true;
      }
    }

    return false;
  }

  /**
   * Check if media has image-based subtitles
   * Note: Currently we don't have format info in SubtitleStream, so this
   * always returns false. When format info is added, implement actual check.
   */
  private hasImageBasedSubtitles(subtitles: SubtitleStream[]): boolean {
    // TODO: Add 'format' or 'codec' field to SubtitleStream and check here
    // For now, assume all subtitles are text-based (VTT/SRT/ASS)
    return false;
  }
}
