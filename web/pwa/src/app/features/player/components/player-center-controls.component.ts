import { Component, Input, Output, EventEmitter, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-player-center-controls',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './player-center-controls.component.html',
  styleUrl: './player-center-controls.component.css'
})
export class PlayerCenterControlsComponent implements OnDestroy {
  @Input({ required: true }) isPlaying!: boolean;
  @Input({ required: true }) visible!: boolean;
  @Input() skipDuration = 30;
  
  @Output() playPauseClicked = new EventEmitter<void>();
  @Output() skipRequested = new EventEmitter<number>();
  
  private pendingSkipSeconds = 0;
  private skipAccumulateTimeout: any = null;
  private readonly ACCUMULATE_WINDOW_MS = 400;
  
  onSkipBack(event: Event): void {
    event.stopPropagation();
    this.accumulateSkip(-this.skipDuration);
  }
  
  onSkipForward(event: Event): void {
    event.stopPropagation();
    this.accumulateSkip(this.skipDuration);
  }
  
  private accumulateSkip(seconds: number): void {
    this.pendingSkipSeconds += seconds;
    
    if (this.skipAccumulateTimeout) {
      clearTimeout(this.skipAccumulateTimeout);
    }
    
    this.skipAccumulateTimeout = setTimeout(() => {
      this.skipRequested.emit(this.pendingSkipSeconds);
      this.pendingSkipSeconds = 0;
      this.skipAccumulateTimeout = null;
    }, this.ACCUMULATE_WINDOW_MS);
  }
  
  onPlayPause(event: Event): void {
    event.stopPropagation();
    this.playPauseClicked.emit();
  }
  
  ngOnDestroy(): void {
    if (this.skipAccumulateTimeout) {
      clearTimeout(this.skipAccumulateTimeout);
    }
  }
}
