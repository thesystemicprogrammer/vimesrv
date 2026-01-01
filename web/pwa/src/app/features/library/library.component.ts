import { Component, inject, OnInit, signal } from '@angular/core';
import { Router } from '@angular/router';
import { ApiService, MediaListItem } from '../../core/services/api.service';
import { MediaCardComponent } from './media-card.component';

@Component({
  selector: 'app-library',
  standalone: true,
  imports: [MediaCardComponent],
  template: `
    <div class="container mx-auto px-4 py-8">
      <div class="flex justify-between items-center mb-8">
        <h1 class="text-3xl font-bold text-white">Library</h1>
        <button
          (click)="scanLibrary()"
          [disabled]="scanning()"
          class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md transition disabled:opacity-50"
        >
          @if (scanning()) {
            Scanning...
          } @else {
            Scan Library
          }
        </button>
      </div>

      @if (loading()) {
        <div class="flex justify-center items-center h-64">
          <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
        </div>
      } @else if (error()) {
        <div class="bg-red-500/10 border border-red-500 text-red-400 px-4 py-3 rounded">
          {{ error() }}
        </div>
      } @else if (media().length === 0) {
        <div class="text-center text-slate-400 py-16">
          <p class="text-xl mb-4">No media found</p>
          <p>Click "Scan Library" to discover media files</p>
        </div>
      } @else {
        <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6">
          @for (item of media(); track item.id) {
            <app-media-card [media]="item" (click)="playMedia(item)" />
          }
        </div>
      }
    </div>
  `
})
export class LibraryComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly router = inject(Router);

  media = signal<MediaListItem[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);
  scanning = signal(false);

  ngOnInit(): void {
    this.loadMedia();
  }

  loadMedia(): void {
    this.loading.set(true);
    this.error.set(null);

    this.api.listMedia(1, 100).subscribe({
      next: (response) => {
        this.media.set(response.data.items || []);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Failed to load media');
        this.loading.set(false);
      }
    });
  }

  playMedia(item: MediaListItem): void {
    this.router.navigate(['/play', item.id]);
  }

  scanLibrary(): void {
    this.scanning.set(true);
    this.api.scanLibrary().subscribe({
      next: () => {
        setTimeout(() => {
          this.scanning.set(false);
          this.loadMedia();
        }, 2000);
      },
      error: () => {
        this.scanning.set(false);
      }
    });
  }
}
