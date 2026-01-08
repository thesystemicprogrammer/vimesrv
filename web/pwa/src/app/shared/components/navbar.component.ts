import { Component, inject, signal, ElementRef, ViewChild, HostListener } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { TranslateModule, TranslateService } from '@ngx-translate/core';
import { AuthService } from '../../core/services/auth.service';
import { ApiService, LibrarySearchResult } from '../../core/services/api.service';
import { debounceTime, Subject, switchMap, of, catchError } from 'rxjs';

interface LanguageOption {
  code: string;
  name: string;
  flag: string;
}

@Component({
  selector: 'app-navbar',
  standalone: true,
  imports: [CommonModule, RouterModule, FormsModule, TranslateModule],
  template: `
    @if (auth.isAuthenticated()) {
      <nav class="bg-zinc-900 border-b border-zinc-800 px-4 py-3">
        <div class="max-w-screen-2xl mx-auto flex items-center justify-between">
          <!-- Logo / Title -->
          <a routerLink="/" class="flex items-center gap-2 text-white hover:text-zinc-300 transition-colors">
            <img src="assets/logo.svg" alt="VimeSrv" class="h-8 w-8" />
            <span class="text-xl font-semibold" [class.hidden]="searchExpanded() && isMobile()">VimeSrv</span>
          </a>

          <!-- Search Bar (expandable) -->
          <div class="flex-1 flex justify-center px-4" [class.hidden]="!searchExpanded() && isMobile()">
            <div class="relative w-full max-w-md" #searchContainer>
              <div class="relative">
                <input
                  #searchInput
                  type="text"
                  [(ngModel)]="searchQuery"
                  (ngModelChange)="onSearchChange($event)"
                  (focus)="onSearchFocus()"
                  (keydown.escape)="closeSearch()"
                  (keydown.enter)="goToSearchResults()"
                  [placeholder]="'common.search' | translate"
                  class="w-full pl-10 pr-4 py-2 bg-zinc-800 border border-zinc-700 rounded-lg text-white placeholder-zinc-400 focus:outline-none focus:border-blue-500 text-sm"
                />
                <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                </svg>
                @if (searchQuery) {
                  <button
                    (click)="clearSearch()"
                    class="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-white"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                    </svg>
                  </button>
                }
              </div>

              <!-- Search Results Dropdown -->
              @if (showSearchResults() && (searchResults().length > 0 || searchLoading() || searchQuery.length >= 2)) {
                <div class="absolute top-full left-0 right-0 mt-2 bg-zinc-800 border border-zinc-700 rounded-lg shadow-xl z-30 max-h-96 overflow-y-auto">
                  @if (searchLoading()) {
                    <div class="flex items-center justify-center py-8">
                      <div class="animate-spin rounded-full h-6 w-6 border-t-2 border-b-2 border-blue-500"></div>
                    </div>
                  } @else if (searchResults().length === 0 && searchQuery.length >= 2) {
                    <div class="py-8 text-center text-zinc-400 text-sm">
                      {{ 'common.noResults' | translate }}
                    </div>
                  } @else {
                    @for (result of searchResults().slice(0, 8); track result.media_id || result.series_metadata_id) {
                      <button
                        (click)="selectSearchResult(result)"
                        class="w-full flex items-center gap-3 p-3 hover:bg-zinc-700 transition text-left"
                      >
                        <!-- Poster thumbnail -->
                        <div class="w-10 h-14 bg-zinc-700 rounded overflow-hidden flex-shrink-0">
                          @if (result.poster_path) {
                            <img
                              [src]="'https://image.tmdb.org/t/p/w92' + result.poster_path"
                              [alt]="result.title"
                              class="w-full h-full object-cover"
                            />
                          } @else {
                            <div class="w-full h-full flex items-center justify-center text-zinc-500">
                              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z"/>
                              </svg>
                            </div>
                          }
                        </div>
                        <div class="flex-1 min-w-0">
                          <p class="text-white text-sm font-medium truncate">{{ result.title }}</p>
                          <div class="flex items-center gap-2 text-xs text-zinc-400">
                            <span class="px-1.5 py-0.5 bg-zinc-700 rounded">{{ result.type === 'movie' ? ('metadataMatch.movie' | translate) : ('metadataMatch.tvSeries' | translate) }}</span>
                            @if (result.year) {
                              <span>{{ result.year }}</span>
                            }
                            @if (result.vote_average) {
                              <span class="flex items-center gap-0.5">
                                <svg class="w-3 h-3 text-yellow-500" fill="currentColor" viewBox="0 0 20 20">
                                  <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"/>
                                </svg>
                                {{ result.vote_average.toFixed(1) }}
                              </span>
                            }
                          </div>
                        </div>
                      </button>
                    }
                    @if (searchResults().length > 8 || searchQuery.length >= 2) {
                      <button
                        (click)="goToSearchResults()"
                        class="w-full py-3 text-center text-sm text-blue-400 hover:text-blue-300 hover:bg-zinc-700 transition border-t border-zinc-700"
                      >
                        {{ 'nav.viewAllResults' | translate }}
                      </button>
                    }
                  }
                </div>
              }
            </div>
          </div>

          <!-- User Info & Actions -->
          <div class="flex items-center gap-2 sm:gap-4">
            <!-- Search Icon (mobile toggle) -->
            <button
              (click)="toggleSearch()"
              class="sm:hidden p-2 text-zinc-400 hover:text-white hover:bg-zinc-800 rounded transition-colors"
              [class.text-blue-400]="searchExpanded()"
            >
              @if (searchExpanded()) {
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                </svg>
              } @else {
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                </svg>
              }
            </button>

            <!-- Language Selector -->
            <div class="relative" [class.hidden]="searchExpanded() && isMobile()">
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
            <div class="relative" [class.hidden]="searchExpanded() && isMobile()">
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
                    {{ 'nav.changePassword' | translate }}
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
                      {{ 'nav.adminPanel' | translate }}
                    </button>
                  }

                  <!-- Jobs (for admin and manager) -->
                  @if (auth.canManageLibrary()) {
                    <button
                      (click)="navigateToJobs()"
                      class="w-full px-4 py-2 text-left text-sm text-zinc-300 hover:bg-zinc-700 hover:text-white transition-colors flex items-center gap-2"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
                      </svg>
                      {{ 'nav.jobs' | translate }}
                    </button>

                    <!-- Transcodings (for admin and manager) -->
                    <button
                      (click)="navigateToTranscodings()"
                      class="w-full px-4 py-2 text-left text-sm text-zinc-300 hover:bg-zinc-700 hover:text-white transition-colors flex items-center gap-2"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
                      </svg>
                      {{ 'nav.transcodings' | translate }}
                    </button>
                  }

                  <!-- Workers (admin only) -->
                  @if (auth.isAdmin()) {
                    <button
                      (click)="navigateToWorkers()"
                      class="w-full px-4 py-2 text-left text-sm text-zinc-300 hover:bg-zinc-700 hover:text-white transition-colors flex items-center gap-2"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
                      </svg>
                      {{ 'nav.workers' | translate }}
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
                    {{ 'nav.logout' | translate }}
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
  private readonly translate = inject(TranslateService);

  @ViewChild('searchInput') searchInput!: ElementRef<HTMLInputElement>;
  @ViewChild('searchContainer') searchContainer!: ElementRef<HTMLDivElement>;

  languageMenuOpen = signal(false);
  userMenuOpen = signal(false);
  toastMessage = signal<string | null>(null);
  toastType = signal<'success' | 'info' | 'error'>('info');

  // Search state
  searchExpanded = signal(false);
  showSearchResults = signal(false);
  searchLoading = signal(false);
  searchResults = signal<LibrarySearchResult[]>([]);
  searchQuery = '';

  private searchSubject = new Subject<string>();

  languages: LanguageOption[] = [
    { code: 'en', name: 'English', flag: '🇬🇧' },
    { code: 'de', name: 'Deutsch', flag: '🇩🇪' }
  ];

  constructor() {
    // Set up debounced search
    this.searchSubject.pipe(
      debounceTime(300),
      switchMap(query => {
        if (query.length < 2) {
          return of({ data: { results: [] } });
        }
        this.searchLoading.set(true);
        return this.api.searchLibrary(query, 10).pipe(
          catchError(() => of({ data: { results: [] } }))
        );
      })
    ).subscribe(response => {
      this.searchResults.set(response.data.results || []);
      this.searchLoading.set(false);
    });
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    // Close search results if clicking outside
    if (this.searchContainer && !this.searchContainer.nativeElement.contains(event.target as Node)) {
      this.showSearchResults.set(false);
    }
  }

  isMobile(): boolean {
    return window.innerWidth < 640; // sm breakpoint
  }

  toggleSearch(): void {
    this.searchExpanded.update(v => !v);
    if (this.searchExpanded()) {
      setTimeout(() => {
        this.searchInput?.nativeElement?.focus();
      }, 0);
    } else {
      this.closeSearch();
    }
  }

  onSearchChange(query: string): void {
    this.searchSubject.next(query);
    this.showSearchResults.set(true);
  }

  onSearchFocus(): void {
    if (this.searchQuery.length >= 2 || this.searchResults().length > 0) {
      this.showSearchResults.set(true);
    }
  }

  closeSearch(): void {
    this.showSearchResults.set(false);
    if (this.isMobile()) {
      this.searchExpanded.set(false);
    }
  }

  clearSearch(): void {
    this.searchQuery = '';
    this.searchResults.set([]);
    this.showSearchResults.set(false);
  }

  selectSearchResult(result: LibrarySearchResult): void {
    this.closeSearch();
    this.clearSearch();
    
    if (result.type === 'movie' && result.media_id) {
      this.router.navigate(['/movie', result.media_id]);
    } else if (result.type === 'series' && result.series_metadata_id) {
      this.router.navigate(['/series', result.series_metadata_id]);
    }
  }

  goToSearchResults(): void {
    if (this.searchQuery.length >= 2) {
      this.closeSearch();
      this.router.navigate(['/search'], { queryParams: { q: this.searchQuery } });
      this.clearSearch();
    }
  }

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

  navigateToJobs(): void {
    this.closeUserMenu();
    this.router.navigate(['/jobs']);
  }

  navigateToTranscodings(): void {
    this.closeUserMenu();
    this.router.navigate(['/transcodings']);
  }

  navigateToWorkers(): void {
    this.closeUserMenu();
    this.router.navigate(['/admin/workers']);
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
    this.showToast(this.translate.instant('toast.fetchingTranslations', { language: langName }), 'info');

    this.api.fetchTranslations(code).subscribe({
      next: (response) => {
        if (response.data.queued) {
          this.showToast(this.translate.instant('toast.translationsQueued', { language: langName }), 'success');
        } else {
          this.showToast(this.translate.instant('toast.translationsInProgress', { language: langName }), 'info');
        }
        // Auto-hide after 3 seconds
        setTimeout(() => this.hideToast(), 3000);
      },
      error: (err) => {
        console.error('Failed to fetch translations:', err);
        this.showToast(this.translate.instant('toast.translationsFailed', { language: langName }), 'error');
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
