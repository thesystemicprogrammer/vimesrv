import { Injectable, signal } from '@angular/core';

interface ScrollState {
  scrollY: number;
  activeTab: string;
  moviesPage: number;
  seriesPage: number;
  moviesCount: number;
  seriesCount: number;
}

/**
 * Service to preserve scroll position and pagination state when navigating
 * away from the library and returning back.
 */
@Injectable({
  providedIn: 'root'
})
export class ScrollStateService {
  private readonly STORAGE_KEY = 'library_scroll_state';

  // Signal to track if we should restore state on next library load
  private shouldRestore = signal(false);

  /**
   * Save the current scroll and pagination state before navigating away
   */
  saveState(state: ScrollState): void {
    try {
      sessionStorage.setItem(this.STORAGE_KEY, JSON.stringify(state));
      this.shouldRestore.set(true);
    } catch {
      // SessionStorage may not be available
    }
  }

  /**
   * Get the saved state if available and restoration is pending
   */
  getState(): ScrollState | null {
    if (!this.shouldRestore()) {
      return null;
    }

    try {
      const stored = sessionStorage.getItem(this.STORAGE_KEY);
      if (stored) {
        return JSON.parse(stored) as ScrollState;
      }
    } catch {
      // Parse error or storage not available
    }
    return null;
  }

  /**
   * Clear the saved state (called after restoration or on full reload)
   */
  clearState(): void {
    try {
      sessionStorage.removeItem(this.STORAGE_KEY);
    } catch {
      // SessionStorage may not be available
    }
    this.shouldRestore.set(false);
  }

  /**
   * Mark that we're navigating away and should restore on return
   */
  markForRestoration(): void {
    this.shouldRestore.set(true);
  }

  /**
   * Check if restoration is pending
   */
  isPendingRestore(): boolean {
    return this.shouldRestore();
  }
}
