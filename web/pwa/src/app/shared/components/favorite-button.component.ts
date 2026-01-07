import { Component, Input, Output, EventEmitter, inject, signal, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FavoritesService } from '../../core/services/favorites.service';

@Component({
  selector: 'app-favorite-button',
  standalone: true,
  imports: [CommonModule],
  template: `
    <button
      type="button"
      [class]="buttonClass"
      [disabled]="loading()"
      (click)="onToggleFavorite($event)"
      [attr.aria-label]="isFavorited() ? 'Remove from favorites' : 'Add to favorites'"
      [title]="isFavorited() ? 'Remove from favorites' : 'Add to favorites'"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        [attr.fill]="isFavorited() ? 'currentColor' : 'none'"
        viewBox="0 0 24 24"
        [attr.stroke]="isFavorited() ? 'none' : 'currentColor'"
        stroke-width="2"
        [class]="iconClass + (isFavorited() ? ' text-red-500' : ' text-white')"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M21 8.25c0-2.485-2.099-4.5-4.688-4.5-1.935 0-3.597 1.126-4.312 2.733-.715-1.607-2.377-2.733-4.313-2.733C5.1 3.75 3 5.765 3 8.25c0 7.22 9 12 9 12s9-4.78 9-12z"
        />
      </svg>
      @if (showLabel && !loading()) {
        <span class="ml-2">{{ isFavorited() ? labelRemove : labelAdd }}</span>
      }
      @if (loading()) {
        <span class="ml-2 animate-pulse">...</span>
      }
    </button>
  `,
  styles: [`
    :host {
      display: inline-block;
    }
    
    button {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      transition: all 0.2s ease-in-out;
    }
    
    button:hover:not(:disabled) {
      transform: scale(1.1);
    }
    
    button:active:not(:disabled) {
      transform: scale(0.95);
    }
    
    button:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
    
    svg {
      transition: all 0.2s ease-in-out;
    }
    
    button:hover:not(:disabled) svg {
      filter: drop-shadow(0 0 4px currentColor);
    }
  `]
})
export class FavoriteButtonComponent {
  private favoritesService = inject(FavoritesService);

  // Inputs
  @Input({ required: true }) mediaType!: 'movie' | 'series';
  @Input({ required: true }) metadataId!: number;
  @Input() showLabel = false;
  @Input() labelAdd = 'Add to Favorites';
  @Input() labelRemove = 'Remove from Favorites';
  @Input() buttonClass = 'btn btn-ghost btn-circle';
  @Input() iconClass = 'h-6 w-6';

  // Outputs
  @Output() favoriteToggled = new EventEmitter<boolean>();
  @Output() error = new EventEmitter<Error>();

  // State
  isFavorited = signal<boolean>(false);
  loading = signal<boolean>(false);

  constructor() {
    // Update favorited status when component initializes or inputs change
    effect(() => {
      if (this.mediaType && this.metadataId) {
        this.isFavorited.set(
          this.favoritesService.isFavorited(this.mediaType, this.metadataId)
        );
      }
    });
  }

  onToggleFavorite(event: Event): void {
    event.preventDefault();
    event.stopPropagation();

    if (this.loading()) return;

    this.loading.set(true);

    this.favoritesService.toggleFavorite(this.mediaType, this.metadataId).subscribe({
      next: (response) => {
        const newStatus = response.data.favorited;
        this.isFavorited.set(newStatus);
        this.loading.set(false);
        this.favoriteToggled.emit(newStatus);
      },
      error: (err) => {
        console.error('Failed to toggle favorite:', err);
        this.loading.set(false);
        this.error.emit(err);
      }
    });
  }
}
