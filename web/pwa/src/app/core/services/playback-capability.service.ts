import { Injectable } from '@angular/core';
import { MediaDetail, SubtitleStream, AudioStream } from './api.service';

export interface PlaybackCapability {
  canDirectPlay: boolean;
  canDirectStream: boolean;
  reason: string;
  // For direct stream: which audio track to use (index in audio_streams array)
  selectedAudioTrackIndex?: number;
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

// Browser-compatible audio codecs
const BROWSER_COMPATIBLE_AUDIO = ['aac', 'mp3', 'opus', 'vorbis', 'flac'];

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
    console.log('Checking codec support for:', {
      format: media.format,
      video_codec: media.video_codec,
      audio_codecs: media.audio_codecs,
      audio_streams: media.audio_streams,
      width: media.width,
      height: media.height,
      bitrate: media.bitrate,
      direct_play_supported: media.direct_play_supported,
      direct_stream_supported: media.direct_stream_supported
    });

    // Check 1: Container format
    if (!this.isDirectPlayableContainer(media.format) &&
        !this.isRemuxableContainer(media.format)) {
      const result: PlaybackCapability = {
        canDirectPlay: false,
        canDirectStream: false,
        reason: `Container format "${media.format}" not supported for direct play`
      };
      console.log('Capability check result:', result);
      return result;
    }

    // Check 2: Video codec support
    const videoSupported = await this.canDecodeVideo(
      media.video_codec,
      media.width,
      media.height,
      media.bitrate
    );
    if (!videoSupported) {
      const result: PlaybackCapability = {
        canDirectPlay: false,
        canDirectStream: false,
        reason: `Video codec "${media.video_codec}" not supported by browser`
      };
      console.log('Capability check result:', result);
      return result;
    }

    // Check 3: Audio codec support
    // Find the first browser-compatible audio track
    const audioAnalysis = await this.findCompatibleAudioTrack(media.audio_streams || []);
    console.log('Audio analysis:', audioAnalysis);

    const canDirectPlay = this.isDirectPlayableContainer(media.format);
    const canDirectStream = this.isRemuxableContainer(media.format);

    // For both direct play and direct stream, we need at least one browser-compatible audio track
    if (audioAnalysis.bestCompatibleIndex === -1 && (media.audio_streams?.length ?? 0) > 0) {
      const result: PlaybackCapability = {
        canDirectPlay: false,
        canDirectStream: false,
        reason: `No browser-compatible audio codec found (available: ${(media.audio_codecs || []).join(', ')})`
      };
      console.log('Capability check result:', result);
      return result;
    }

    // Check 4: Subtitle format (image-based subs require transcoding)
    if (this.hasImageBasedSubtitles(media.subtitle_streams)) {
      const result: PlaybackCapability = {
        canDirectPlay: false,
        canDirectStream: false,
        reason: 'Media contains image-based subtitles (PGS/VOBSUB) which require transcoding'
      };
      console.log('Capability check result:', result);
      return result;
    }

    // Determine selected audio track for direct stream
    const selectedAudioTrackIndex = audioAnalysis.bestCompatibleIndex !== -1 ? audioAnalysis.bestCompatibleIndex : 0;

    const result: PlaybackCapability = {
      canDirectPlay,
      canDirectStream,
      reason: canDirectPlay
        ? 'Direct play supported'
        : 'Remux required for container',
      selectedAudioTrackIndex
    };
    console.log('Capability check result:', result);
    return result;
  }

  /**
   * Find the first browser-compatible audio track.
   * Returns the index in the audio_streams array, or -1 if none found.
   */
  private async findCompatibleAudioTrack(audioStreams: AudioStream[]): Promise<{
    bestCompatibleIndex: number;
    compatibleIndices: number[];
  }> {
    if (!audioStreams || audioStreams.length === 0) {
      return { bestCompatibleIndex: -1, compatibleIndices: [] };
    }

    const compatibleIndices: number[] = [];

    for (let i = 0; i < audioStreams.length; i++) {
      const stream = audioStreams[i];
      
      if (stream.codec && this.isBrowserCompatibleAudio(stream.codec)) {
        // Verify browser can actually decode it
        const isSupported = await this.canDecodeAudioCodec(stream.codec);
        if (isSupported) {
          compatibleIndices.push(i);
        }
      }
    }

    return {
      bestCompatibleIndex: compatibleIndices.length > 0 ? compatibleIndices[0] : -1,
      compatibleIndices
    };
  }

  /**
   * Check if an audio codec is known to be browser-compatible
   */
  private isBrowserCompatibleAudio(codec: string): boolean {
    return BROWSER_COMPATIBLE_AUDIO.includes(codec.toLowerCase());
  }

  /**
   * Test if a specific audio codec can be decoded by the browser
   */
  private async canDecodeAudioCodec(codec: string): Promise<boolean> {
    const codecString = AUDIO_CODEC_STRINGS[codec.toLowerCase()];
    if (!codecString) {
      return false;
    }

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
        return result.supported;
      } catch {
        // Fall through to canPlayType
      }
    }

    const audio = document.createElement('audio');
    const canPlay = audio.canPlayType(`audio/mp4; codecs="${codecString}"`);
    return canPlay === 'probably' || canPlay === 'maybe';
  }

  /**
   * Check if container supports direct play (no remux needed)
   * Note: ffprobe may return comma-separated formats like "matroska,webm"
   */
  private isDirectPlayableContainer(format: string): boolean {
    if (!format) return false;
    const normalized = format.toLowerCase();
    return normalized.includes('mp4') || normalized.includes('mov') || normalized.includes('m4v');
  }

  /**
   * Check if container can be remuxed for direct stream
   * Note: ffprobe may return comma-separated formats like "matroska,webm"
   */
  private isRemuxableContainer(format: string): boolean {
    if (!format) return false;
    const normalized = format.toLowerCase();
    return normalized.includes('mkv') || normalized.includes('matroska') ||
           normalized.includes('avi') || normalized.includes('webm');
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
        console.log(`MediaCapabilities video check: codec=${codec}, supported=${result.supported}, smooth=${result.smooth}`);
        // Only require supported, not smooth - the smooth check is too conservative
        // and may reject playable content on capable hardware
        return result.supported;
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
