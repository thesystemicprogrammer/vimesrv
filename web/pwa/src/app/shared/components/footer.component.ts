import { Component } from '@angular/core';
import { TranslateModule } from '@ngx-translate/core';

@Component({
  selector: 'app-footer',
  standalone: true,
  imports: [TranslateModule],
  template: `
    <footer class="bg-zinc-900 border-t border-zinc-800 px-4 py-3">
      <div class="max-w-screen-2xl mx-auto flex items-center justify-end gap-2 text-zinc-400 text-sm">
        <span>{{ 'footer.metadataProvidedBy' | translate }}</span>
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
