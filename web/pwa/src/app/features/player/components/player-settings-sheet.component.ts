import { Component, Input, Output, EventEmitter } from '@angular/core';
import { CommonModule } from '@angular/common';
import { AudioStream, SubtitleStream } from '../../../core/services/api.service';
import { QualityLevel } from '../services/playback-engine.service';

@Component({
  selector: 'app-player-settings-sheet',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './player-settings-sheet.component.html',
  styleUrl: './player-settings-sheet.component.css'
})
export class PlayerSettingsSheetComponent {
  @Input() isOpen = false;
  
  // Track inputs
  @Input() audioTracks: AudioStream[] = [];
  @Input() currentAudioIndex = 0;
  @Input() subtitleTracks: SubtitleStream[] = [];
  @Input() currentSubtitleIndex = -1;
  @Input() qualityLevels: QualityLevel[] = [];
  @Input() currentQualityIndex = -1;
  
  // Outputs
  @Output() closed = new EventEmitter<void>();
  @Output() audioTrackChanged = new EventEmitter<number>();
  @Output() subtitleTrackChanged = new EventEmitter<number>();
  @Output() qualityChanged = new EventEmitter<number>();
  
  // Track which section is expanded (for mobile accordion style)
  expandedSection: 'audio' | 'subtitles' | 'quality' | null = null;
  
  onBackdropClick(event: MouseEvent): void {
    // Only close if clicking the backdrop itself, not the sheet content
    if ((event.target as HTMLElement).classList.contains('settings-backdrop')) {
      this.closed.emit();
    }
  }
  
  onClose(): void {
    this.closed.emit();
  }
  
  toggleSection(section: 'audio' | 'subtitles' | 'quality'): void {
    this.expandedSection = this.expandedSection === section ? null : section;
  }
  
  onAudioSelect(index: number): void {
    this.audioTrackChanged.emit(index);
  }
  
  onSubtitleSelect(index: number): void {
    this.subtitleTrackChanged.emit(index);
  }
  
  onQualitySelect(index: number): void {
    this.qualityChanged.emit(index);
  }
  
  getAudioTrackLabel(track: AudioStream, index: number): string {
    return track.title || track.language || `Track ${index + 1}`;
  }
  
  getSubtitleTrackLabel(track: SubtitleStream, index: number): string {
    return track.title || track.language || `Track ${index + 1}`;
  }
}
