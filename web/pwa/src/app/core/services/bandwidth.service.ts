import { Injectable, inject } from '@angular/core';
import { ApiService } from './api.service';
import { firstValueFrom } from 'rxjs';

export interface BandwidthMeasurement {
  bitsPerSecond: number;
  timestamp: number;
  bytesTransferred: number;
  durationMs: number;
}

const STORAGE_KEY = 'vime_bandwidth_measurement';
const MEASUREMENT_TTL_MS = 5 * 60 * 1000; // 5 minutes
const SAFETY_MARGIN = 1.3; // Require 30% headroom over bitrate

@Injectable({
  providedIn: 'root'
})
export class BandwidthService {
  private readonly api = inject(ApiService);

  /**
   * Measure current bandwidth by fetching probe data from the server
   * @param bytes Number of bytes to download for measurement (default 2MB)
   * @returns Measurement result with bits per second
   */
  async measure(bytes: number = 2_000_000): Promise<BandwidthMeasurement> {
    const startTime = performance.now();

    try {
      const buffer = await firstValueFrom(this.api.probe(bytes));
      const endTime = performance.now();
      const durationMs = endTime - startTime;
      const bytesTransferred = buffer.byteLength;

      // Calculate bits per second
      const bitsPerSecond = (bytesTransferred * 8) / (durationMs / 1000);

      const measurement: BandwidthMeasurement = {
        bitsPerSecond,
        timestamp: Date.now(),
        bytesTransferred,
        durationMs
      };

      // Cache the measurement
      this.saveMeasurement(measurement);

      return measurement;
    } catch (err) {
      console.error('Bandwidth measurement failed:', err);
      throw err;
    }
  }

  /**
   * Get cached measurement if it's still valid (within TTL)
   * @returns Cached measurement or null if expired/not found
   */
  getCached(): BandwidthMeasurement | null {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (!stored) return null;

      const measurement: BandwidthMeasurement = JSON.parse(stored);

      // Check if measurement is still valid
      if (Date.now() - measurement.timestamp > MEASUREMENT_TTL_MS) {
        localStorage.removeItem(STORAGE_KEY);
        return null;
      }

      return measurement;
    } catch {
      return null;
    }
  }

  /**
   * Check if bandwidth is sufficient for the given bitrate
   * Uses a 1.3x safety margin (30% headroom)
   * @param bitrate Required bitrate in bits per second
   * @param measurement Optional measurement to use (defaults to cached)
   * @returns true if bandwidth is sufficient
   */
  isSufficient(bitrate: number, measurement?: BandwidthMeasurement): boolean {
    const m = measurement || this.getCached();
    if (!m) return false;

    const requiredBandwidth = bitrate * SAFETY_MARGIN;
    return m.bitsPerSecond >= requiredBandwidth;
  }

  /**
   * Get human-readable bandwidth string
   */
  formatBandwidth(bitsPerSecond: number): string {
    if (bitsPerSecond >= 1_000_000_000) {
      return `${(bitsPerSecond / 1_000_000_000).toFixed(1)} Gbps`;
    }
    if (bitsPerSecond >= 1_000_000) {
      return `${(bitsPerSecond / 1_000_000).toFixed(1)} Mbps`;
    }
    if (bitsPerSecond >= 1_000) {
      return `${(bitsPerSecond / 1_000).toFixed(0)} Kbps`;
    }
    return `${bitsPerSecond.toFixed(0)} bps`;
  }

  /**
   * Clear cached measurement
   */
  clearCache(): void {
    localStorage.removeItem(STORAGE_KEY);
  }

  private saveMeasurement(measurement: BandwidthMeasurement): void {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(measurement));
    } catch {
      // localStorage might be unavailable or full
    }
  }
}
