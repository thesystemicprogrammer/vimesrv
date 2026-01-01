import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';

@Component({
  selector: 'app-navbar',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    @if (auth.isAuthenticated()) {
      <nav class="bg-zinc-900 border-b border-zinc-800 px-4 py-3">
        <div class="max-w-7xl mx-auto flex items-center justify-between">
          <!-- Logo / Title -->
          <a routerLink="/" class="flex items-center gap-2 text-white hover:text-zinc-300 transition-colors">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-blue-500" viewBox="0 0 24 24" fill="currentColor">
              <path d="M4 4h16a2 2 0 012 2v12a2 2 0 01-2 2H4a2 2 0 01-2-2V6a2 2 0 012-2zm0 2v12h16V6H4zm6 3l6 3-6 3V9z"/>
            </svg>
            <span class="text-xl font-semibold">VimeSrv</span>
          </a>

          <!-- User Info & Actions -->
          <div class="flex items-center gap-4">
            @if (auth.username()) {
              <span class="text-zinc-400 text-sm hidden sm:inline">
                {{ auth.username() }}
              </span>
            }
            
            <button
              (click)="logout()"
              class="flex items-center gap-2 px-3 py-1.5 text-sm text-zinc-400 hover:text-white hover:bg-zinc-800 rounded transition-colors"
            >
              <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
              </svg>
              <span class="hidden sm:inline">Logout</span>
            </button>
          </div>
        </div>
      </nav>
    }
  `,
  styles: [`
    :host {
      display: block;
    }
  `]
})
export class NavbarComponent {
  readonly auth = inject(AuthService);

  logout(): void {
    this.auth.logout();
    // Router navigation will be handled by the auth guard
    window.location.href = '/login';
  }
}
