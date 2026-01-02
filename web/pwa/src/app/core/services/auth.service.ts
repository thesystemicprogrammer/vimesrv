import { Injectable, signal, computed } from '@angular/core';

const TOKEN_KEY = 'vimesrv_token';
const STREAM_TOKEN_KEY = 'vimesrv_stream_token';
const LANGUAGE_KEY = 'vimesrv_language';
const DEFAULT_LANGUAGE = 'en';

export interface AuthState {
  token: string | null;
  streamToken: string | null;
  username: string | null;
  isAuthenticated: boolean;
  language: string;
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private tokenSignal = signal<string | null>(this.getStoredToken());
  private streamTokenSignal = signal<string | null>(null);
  private usernameSignal = signal<string | null>(null);
  private languageSignal = signal<string>(this.getStoredLanguage());

  readonly isAuthenticated = computed(() => !!this.tokenSignal());
  readonly token = computed(() => this.tokenSignal());
  readonly streamToken = computed(() => this.streamTokenSignal());
  readonly username = computed(() => this.usernameSignal());
  readonly language = computed(() => this.languageSignal());

  constructor() {
    // Restore token from storage on init
    const storedToken = this.getStoredToken();
    if (storedToken) {
      this.tokenSignal.set(storedToken);
    }
  }

  private getStoredToken(): string | null {
    try {
      return localStorage.getItem(TOKEN_KEY);
    } catch {
      return null;
    }
  }

  private getStoredLanguage(): string {
    try {
      return localStorage.getItem(LANGUAGE_KEY) || DEFAULT_LANGUAGE;
    } catch {
      return DEFAULT_LANGUAGE;
    }
  }

  setToken(token: string): void {
    this.tokenSignal.set(token);
    try {
      localStorage.setItem(TOKEN_KEY, token);
    } catch {
      // Storage not available
    }
  }

  setStreamToken(token: string): void {
    this.streamTokenSignal.set(token);
  }

  setUsername(username: string): void {
    this.usernameSignal.set(username);
  }

  setLanguage(lang: string): void {
    this.languageSignal.set(lang);
    try {
      localStorage.setItem(LANGUAGE_KEY, lang);
    } catch {
      // Storage not available
    }
  }

  logout(): void {
    this.tokenSignal.set(null);
    this.streamTokenSignal.set(null);
    this.usernameSignal.set(null);
    try {
      localStorage.removeItem(TOKEN_KEY);
      localStorage.removeItem(STREAM_TOKEN_KEY);
    } catch {
      // Storage not available
    }
  }

  getAuthHeader(): string | null {
    const token = this.tokenSignal();
    return token ? `Bearer ${token}` : null;
  }
}
