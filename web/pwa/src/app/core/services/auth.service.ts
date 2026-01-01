import { Injectable, signal, computed } from '@angular/core';

const TOKEN_KEY = 'vimesrv_token';
const STREAM_TOKEN_KEY = 'vimesrv_stream_token';

export interface AuthState {
  token: string | null;
  streamToken: string | null;
  username: string | null;
  isAuthenticated: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private tokenSignal = signal<string | null>(this.getStoredToken());
  private streamTokenSignal = signal<string | null>(null);
  private usernameSignal = signal<string | null>(null);

  readonly isAuthenticated = computed(() => !!this.tokenSignal());
  readonly token = computed(() => this.tokenSignal());
  readonly streamToken = computed(() => this.streamTokenSignal());
  readonly username = computed(() => this.usernameSignal());

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
