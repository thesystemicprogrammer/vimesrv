import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';
import { ApiService } from '../../core/services/api.service';

interface LanguageOption {
  code: string;
  name: string;
  flag: string;
}

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
            <!-- Language Selector -->
            <div class="relative">
              <button
                (click)="toggleLanguageMenu()"
                class="flex items-center gap-2 px-3 py-1.5 text-sm text-zinc-400 hover:text-white hover:bg-zinc-800 rounded transition-colors"
              >
                <span class="text-base">{{ getCurrentLanguageFlag() }}</span>
                <span class="hidden sm:inline">{{ getCurrentLanguageName() }}</span>
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
                </svg>
              </button>

              @if (languageMenuOpen()) {
                <!-- Backdrop -->
                <div
                  class="fixed inset-0 z-10"
                  (click)="closeLanguageMenu()"
                ></div>

                <!-- Dropdown -->
                <div class="absolute right-0 mt-2 w-40 bg-zinc-800 border border-zinc-700 rounded-lg shadow-lg z-20 py-1">
                  @for (lang of languages; track lang.code) {
                    <button
                      (click)="selectLanguage(lang.code)"
                      class="w-full px-4 py-2 text-left text-sm flex items-center gap-2 hover:bg-zinc-700 transition-colors"
                      [class.text-blue-400]="auth.language() === lang.code"
                      [class.text-zinc-300]="auth.language() !== lang.code"
                    >
                      <span class="text-base">{{ lang.flag }}</span>
                      <span>{{ lang.name }}</span>
                      @if (auth.language() === lang.code) {
                        <svg class="w-4 h-4 ml-auto" fill="currentColor" viewBox="0 0 20 20">
                          <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
                        </svg>
                      }
                    </button>
                  }
                </div>
              }
            </div>

            <!-- User Menu Dropdown -->
            <div class="relative">
              <button
                (click)="toggleUserMenu()"
                class="flex items-center gap-2 px-3 py-1.5 text-sm text-zinc-400 hover:text-white hover:bg-zinc-800 rounded transition-colors"
              >
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                </svg>
                <span class="hidden sm:inline">{{ auth.username() }}</span>
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
                </svg>
              </button>

              @if (userMenuOpen()) {
                <!-- Backdrop -->
                <div
                  class="fixed inset-0 z-10"
                  (click)="closeUserMenu()"
                ></div>

                <!-- User Dropdown Menu -->
                <div class="absolute right-0 mt-2 w-48 bg-zinc-800 border border-zinc-700 rounded-lg shadow-lg z-20 py-1">
                  <!-- Username header -->
                  <div class="px-4 py-2 border-b border-zinc-700">
                    <p class="text-sm font-medium text-white">{{ auth.username() }}</p>
                    <p class="text-xs text-zinc-400 capitalize">{{ auth.role() }}</p>
                  </div>

                  <!-- Change Password -->
                  <button
                    (click)="navigateToChangePassword()"
                    class="w-full px-4 py-2 text-left text-sm text-zinc-300 hover:bg-zinc-700 hover:text-white transition-colors flex items-center gap-2"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                    </svg>
                    Change Password
                  </button>

                  <!-- Admin Panel (only for admins) -->
                  @if (auth.isAdmin()) {
                    <button
                      (click)="navigateToAdmin()"
                      class="w-full px-4 py-2 text-left text-sm text-zinc-300 hover:bg-zinc-700 hover:text-white transition-colors flex items-center gap-2"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                      </svg>
                      Admin Panel
                    </button>
                  }

                  <div class="border-t border-zinc-700 my-1"></div>

                  <!-- Logout -->
                  <button
                    (click)="logout()"
                    class="w-full px-4 py-2 text-left text-sm text-red-400 hover:bg-zinc-700 hover:text-red-300 transition-colors flex items-center gap-2"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                    </svg>
                    Logout
                  </button>
                </div>
              }
            </div>
          </div>
        </div>
      </nav>
    }

    <!-- Toast notification -->
    @if (toastMessage()) {
      <div class="fixed bottom-4 right-4 z-50 animate-slide-up">
        <div 
          class="px-4 py-3 rounded-lg shadow-lg flex items-center gap-3"
          [class.bg-green-600]="toastType() === 'success'"
          [class.bg-blue-600]="toastType() === 'info'"
          [class.bg-red-600]="toastType() === 'error'"
        >
          @if (toastType() === 'info') {
            <svg class="w-5 h-5 text-white animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          }
          @if (toastType() === 'success') {
            <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
            </svg>
          }
          @if (toastType() === 'error') {
            <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          }
          <span class="text-white text-sm">{{ toastMessage() }}</span>
        </div>
      </div>
    }
  `,
  styles: [`
    :host {
      display: block;
    }
    
    @keyframes slide-up {
      from {
        opacity: 0;
        transform: translateY(1rem);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }
    
    .animate-slide-up {
      animation: slide-up 0.2s ease-out;
    }
  `]
})
export class NavbarComponent {
  readonly auth = inject(AuthService);
  private readonly api = inject(ApiService);
  private readonly router = inject(Router);

  languageMenuOpen = signal(false);
  userMenuOpen = signal(false);
  toastMessage = signal<string | null>(null);
  toastType = signal<'success' | 'info' | 'error'>('info');

  languages: LanguageOption[] = [
    { code: 'en', name: 'English', flag: '🇬🇧' },
    { code: 'es', name: 'Español', flag: '🇪🇸' },
    { code: 'fr', name: 'Français', flag: '🇫🇷' },
    { code: 'de', name: 'Deutsch', flag: '🇩🇪' },
    { code: 'it', name: 'Italiano', flag: '🇮🇹' },
    { code: 'pt', name: 'Português', flag: '🇵🇹' },
    { code: 'nl', name: 'Nederlands', flag: '🇳🇱' },
    { code: 'ja', name: '日本語', flag: '🇯🇵' },
    { code: 'ko', name: '한국어', flag: '🇰🇷' },
    { code: 'zh', name: '中文', flag: '🇨🇳' }
  ];

  toggleLanguageMenu(): void {
    this.languageMenuOpen.update(open => !open);
    // Close user menu if open
    if (this.languageMenuOpen()) {
      this.userMenuOpen.set(false);
    }
  }

  closeLanguageMenu(): void {
    this.languageMenuOpen.set(false);
  }

  toggleUserMenu(): void {
    this.userMenuOpen.update(open => !open);
    // Close language menu if open
    if (this.userMenuOpen()) {
      this.languageMenuOpen.set(false);
    }
  }

  closeUserMenu(): void {
    this.userMenuOpen.set(false);
  }

  navigateToChangePassword(): void {
    this.closeUserMenu();
    this.router.navigate(['/change-password']);
  }

  navigateToAdmin(): void {
    this.closeUserMenu();
    this.router.navigate(['/admin']);
  }

  selectLanguage(code: string): void {
    const previousLanguage = this.auth.language();
    
    // Don't do anything if selecting the same language
    if (code === previousLanguage) {
      this.closeLanguageMenu();
      return;
    }

    this.auth.setLanguage(code);
    this.closeLanguageMenu();

    // Get the language name for the toast
    const langName = this.languages.find(l => l.code === code)?.name || code;

    // Show toast and trigger translation fetch
    this.showToast(`Fetching ${langName} translations...`, 'info');

    this.api.fetchTranslations(code).subscribe({
      next: (response) => {
        if (response.data.queued) {
          this.showToast(`${langName} translations queued`, 'success');
        } else {
          this.showToast(`${langName} translations already in progress`, 'info');
        }
        // Auto-hide after 3 seconds
        setTimeout(() => this.hideToast(), 3000);
      },
      error: (err) => {
        console.error('Failed to fetch translations:', err);
        this.showToast(`Failed to queue ${langName} translations`, 'error');
        setTimeout(() => this.hideToast(), 5000);
      }
    });
  }

  private showToast(message: string, type: 'success' | 'info' | 'error'): void {
    this.toastMessage.set(message);
    this.toastType.set(type);
  }

  private hideToast(): void {
    this.toastMessage.set(null);
  }

  getCurrentLanguageFlag(): string {
    const lang = this.languages.find(l => l.code === this.auth.language());
    return lang?.flag || '🌐';
  }

  getCurrentLanguageName(): string {
    const lang = this.languages.find(l => l.code === this.auth.language());
    return lang?.name || 'Language';
  }

  logout(): void {
    this.closeUserMenu();
    this.auth.logout();
    this.router.navigate(['/login']);
  }
}
