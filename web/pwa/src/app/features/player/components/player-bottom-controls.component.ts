import { Component, Input, Output, EventEmitter, ElementRef, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AudioStream, SubtitleStream } from '../../../core/services/api.service';
import { QualityLevel } from '../services/playback-engine.service';

@Component({
  selector: 'app-player-bottom-controls',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './player-bottom-controls.component.html',
  styleUrl: './player-bottom-controls.component.css'
})
export class PlayerBottomControlsComponent {
  @ViewChild('progressBar') progressBar!: ElementRef<HTMLDivElement>;
  
  // Playback state inputs
  @Input({ required: true }) currentTime!: number;
  @Input({ required: true }) duration!: number;
  @Input({ required: true }) progress!: number;
  @Input({ required: true }) volume!: number;
  @Input({ required: true }) isMuted!: boolean;
  @Input({ required: true }) isPlaying!: boolean;
  @Input({ required: true }) isFullscreen!: boolean;
  @Input({ required: true }) visible!: boolean;
  
  // Track inputs
  @Input() audioTracks: AudioStream[] = [];
  @Input() currentAudioIndex = 0;
  @Input() subtitleTracks: SubtitleStream[] = [];
  @Input() currentSubtitleIndex = -1;
  @Input() qualityLevels: QualityLevel[] = [];
  @Input() currentQualityIndex = -1;
  
  // Mode label
  @Input() playbackModeLabel = '';
  
  // Outputs
  @Output() seekRequested = new EventEmitter<number>();
  @Output() volumeChanged = new EventEmitter<number>();
  @Output() muteToggled = new EventEmitter<void>();
  @Output() playPauseToggled = new EventEmitter<void>();
  @Output() fullscreenToggled = new EventEmitter<void>();
  @Output() audioTrackChanged = new EventEmitter<number>();
  @Output() subtitleTrackChanged = new EventEmitter<number>();
  @Output() qualityChanged = new EventEmitter<number>();
  @Output() settingsRequested = new EventEmitter<void>();
  
  onSeek(event: MouseEvent): void {
    const progressBar = this.progressBar.nativeElement;
    const rect = progressBar.getBoundingClientRect();
    const percent = (event.clientX - rect.left) / rect.width;
    this.seekRequested.emit(percent * 100);
  }
  
  onVolumeSliderChange(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.volumeChanged.emit(parseFloat(input.value));
  }
  
  onToggleMute(event: Event): void {
    event.stopPropagation();
    this.muteToggled.emit();
  }
  
  onTogglePlay(event: Event): void {
    event.stopPropagation();
    this.playPauseToggled.emit();
  }
  
  onToggleFullscreen(): void {
    this.fullscreenToggled.emit();
  }
  
  onAudioChange(event: Event): void {
    const select = event.target as HTMLSelectElement;
    this.audioTrackChanged.emit(parseInt(select.value, 10));
  }
  
  onSubtitleChange(event: Event): void {
    const select = event.target as HTMLSelectElement;
    this.subtitleTrackChanged.emit(parseInt(select.value, 10));
  }
  
  onQualityChange(event: Event): void {
    const select = event.target as HTMLSelectElement;
    this.qualityChanged.emit(parseInt(select.value, 10));
  }
  
  onSettingsClick(event: Event): void {
    event.stopPropagation();
    this.settingsRequested.emit();
  }
  
  hasTrackOptions(): boolean {
    return this.audioTracks.length > 1 || this.subtitleTracks.length > 0 || this.qualityLevels.length > 1;
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
}
