import { Injectable, signal, computed } from '@angular/core';

const TOKEN_KEY = 'vimesrv_token';
const STREAM_TOKEN_KEY = 'vimesrv_stream_token';
const LANGUAGE_KEY = 'vimesrv_language';
const DEFAULT_LANGUAGE = 'en';

export type UserRole = 'admin' | 'manager' | 'user';

export interface JwtClaims {
  sub: string;           // username
  user_id: string;
  role: UserRole;
  must_change_password: boolean;
  exp: number;
  iat: number;
}

export interface AuthState {
  token: string | null;
  streamToken: string | null;
  username: string | null;
  isAuthenticated: boolean;
  language: string;
  role: UserRole | null;
  userId: string | null;
  mustChangePassword: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private tokenSignal = signal<string | null>(this.getStoredToken());
  private streamTokenSignal = signal<string | null>(null);
  private usernameSignal = signal<string | null>(null);
  private languageSignal = signal<string>(this.getStoredLanguage());
  private roleSignal = signal<UserRole | null>(null);
  private userIdSignal = signal<string | null>(null);
  private mustChangePasswordSignal = signal<boolean>(false);

  readonly isAuthenticated = computed(() => !!this.tokenSignal());
  readonly token = computed(() => this.tokenSignal());
  readonly streamToken = computed(() => this.streamTokenSignal());
  readonly username = computed(() => this.usernameSignal());
  readonly language = computed(() => this.languageSignal());
  readonly role = computed(() => this.roleSignal());
  readonly userId = computed(() => this.userIdSignal());
  readonly mustChangePassword = computed(() => this.mustChangePasswordSignal());
  readonly isAdmin = computed(() => this.roleSignal() === 'admin');
  readonly isManager = computed(() => this.roleSignal() === 'manager');
  readonly canManageUsers = computed(() => this.roleSignal() === 'admin');
  readonly canManageLibrary = computed(() => this.roleSignal() === 'admin' || this.roleSignal() === 'manager');

  constructor() {
    // Restore token from storage on init
    const storedToken = this.getStoredToken();
    if (storedToken) {
      this.tokenSignal.set(storedToken);
      this.parseAndSetClaims(storedToken);
    }
  }

  private parseJwt(token: string): JwtClaims | null {
    try {
      const parts = token.split('.');
      if (parts.length !== 3) {
        return null;
      }
      const payload = parts[1];
      const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
      return JSON.parse(decoded) as JwtClaims;
    } catch {
      return null;
    }
  }

  private parseAndSetClaims(token: string): void {
    const claims = this.parseJwt(token);
    if (claims) {
      this.usernameSignal.set(claims.sub);
      this.userIdSignal.set(claims.user_id);
      this.roleSignal.set(claims.role);
      this.mustChangePasswordSignal.set(claims.must_change_password);
    }
  }

  getClaims(): JwtClaims | null {
    const token = this.tokenSignal();
    if (!token) return null;
    return this.parseJwt(token);
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
    this.parseAndSetClaims(token);
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
    this.roleSignal.set(null);
    this.userIdSignal.set(null);
    this.mustChangePasswordSignal.set(false);
    try {
      localStorage.removeItem(TOKEN_KEY);
      localStorage.removeItem(STREAM_TOKEN_KEY);
    } catch {
      // Storage not available
    }
  }

  clearMustChangePassword(): void {
    this.mustChangePasswordSignal.set(false);
  }

  getAuthHeader(): string | null {
    const token = this.tokenSignal();
    return token ? `Bearer ${token}` : null;
  }
}
