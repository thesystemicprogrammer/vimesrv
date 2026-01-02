import { Component } from '@angular/core';

@Component({
  selector: 'app-footer',
  standalone: true,
  template: `
    <footer class="bg-zinc-900 border-t border-zinc-800 px-4 py-3">
      <div class="max-w-7xl mx-auto flex items-center justify-center gap-2 text-zinc-400 text-sm">
        <span>Metadata provided by</span>
        <a 
          href="https://www.themoviedb.org" 
          target="_blank" 
          rel="noopener noreferrer"
          class="inline-flex items-center hover:opacity-80 transition-opacity"
        >
          <img 
            src="tmdb-logo.svg" 
            alt="TMDB" 
            class="h-4"
          />
        </a>
      </div>
    </footer>
  `
})
export class FooterComponent {}
