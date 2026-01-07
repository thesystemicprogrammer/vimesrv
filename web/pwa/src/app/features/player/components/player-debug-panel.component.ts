import { Component, Input } from '@angular/core';
import { CommonModule, DecimalPipe } from '@angular/common';
import { DebugStats } from '../services/playback-engine.service';

@Component({
  selector: 'app-player-debug-panel',
  standalone: true,
  imports: [CommonModule, DecimalPipe],
  templateUrl: './player-debug-panel.component.html',
  styleUrl: './player-debug-panel.component.css'
})
export class PlayerDebugPanelComponent {
  @Input({ required: true }) visible!: boolean;
  @Input({ required: true }) stats!: DebugStats | null;
}
