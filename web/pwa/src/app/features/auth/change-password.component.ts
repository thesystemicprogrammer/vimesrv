import { Component, inject, signal, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { TranslateModule, TranslateService } from '@ngx-translate/core';
import { ApiService } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';

@Component({
  selector: 'app-change-password',
  standalone: true,
  imports: [FormsModule, TranslateModule],
  template: `
    <div class="min-h-screen flex items-center justify-center bg-slate-900 px-4">
      <div class="max-w-md w-full space-y-8">
        <div class="text-center">
          <h1 class="text-3xl font-bold text-white">{{ 'auth.changePassword' | translate }}</h1>
          @if (isForced()) {
            <p class="mt-2 text-yellow-400">{{ 'auth.mustChangePassword' | translate }}</p>
          } @else {
            <p class="mt-2 text-slate-400">{{ 'auth.updatePassword' | translate }}</p>
          }
        </div>

        <form (ngSubmit)="changePassword()" class="mt-8 space-y-6 bg-slate-800 p-8 rounded-lg shadow-xl">
          @if (error()) {
            <div class="bg-red-500/10 border border-red-500 text-red-400 px-4 py-3 rounded">
              {{ error() }}
            </div>
          }

          @if (success()) {
            <div class="bg-green-500/10 border border-green-500 text-green-400 px-4 py-3 rounded">
              {{ success() }}
            </div>
          }

          <div class="space-y-4">
            <div>
              <label for="currentPassword" class="block text-sm font-medium text-slate-300">{{ 'auth.currentPassword' | translate }}</label>
              <input
                id="currentPassword"
                name="currentPassword"
                type="password"
                [(ngModel)]="currentPassword"
                required
                class="mt-1 block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                [placeholder]="'auth.enterCurrentPassword' | translate"
              />
            </div>

            <div>
              <label for="newPassword" class="block text-sm font-medium text-slate-300">{{ 'auth.newPassword' | translate }}</label>
              <input
                id="newPassword"
                name="newPassword"
                type="password"
                [(ngModel)]="newPassword"
                required
                minlength="8"
                class="mt-1 block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                [placeholder]="'auth.enterNewPassword' | translate"
              />
            </div>

            <div>
              <label for="confirmPassword" class="block text-sm font-medium text-slate-300">{{ 'auth.confirmPassword' | translate }}</label>
              <input
                id="confirmPassword"
                name="confirmPassword"
                type="password"
                [(ngModel)]="confirmPassword"
                required
                class="mt-1 block w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                [placeholder]="'auth.confirmNewPassword' | translate"
              />
            </div>
          </div>

          <div class="flex gap-3">
            @if (!isForced()) {
              <button
                type="button"
                (click)="cancel()"
                class="flex-1 py-2 px-4 border border-slate-600 rounded-md shadow-sm text-sm font-medium text-slate-300 bg-transparent hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-slate-500"
              >
                {{ 'common.cancel' | translate }}
              </button>
            }
            <button
              type="submit"
              [disabled]="loading()"
              class="flex-1 flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              @if (loading()) {
                <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                {{ 'auth.changingPassword' | translate }}
              } @else {
                {{ 'auth.changePassword' | translate }}
              }
            </button>
          </div>
        </form>
      </div>
    </div>
  `
})
export class ChangePasswordComponent implements OnInit {
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly translate = inject(TranslateService);

  currentPassword = '';
  newPassword = '';
  confirmPassword = '';
  loading = signal(false);
  error = signal<string | null>(null);
  success = signal<string | null>(null);
  isForced = signal(false);

  ngOnInit(): void {
    // Check if this is a forced password change
    this.isForced.set(this.auth.mustChangePassword());
  }

  changePassword(): void {
    this.error.set(null);
    this.success.set(null);

    // Validate inputs
    if (!this.currentPassword || !this.newPassword || !this.confirmPassword) {
      this.error.set(this.translate.instant('auth.fillAllFields'));
      return;
    }

    if (this.newPassword.length < 8) {
      this.error.set(this.translate.instant('auth.passwordMinLength'));
      return;
    }

    if (this.newPassword !== this.confirmPassword) {
      this.error.set(this.translate.instant('auth.passwordMismatch'));
      return;
    }

    if (this.currentPassword === this.newPassword) {
      this.error.set(this.translate.instant('auth.passwordMustBeDifferent'));
      return;
    }

    this.loading.set(true);

    this.api.changePassword({
      current_password: this.currentPassword,
      new_password: this.newPassword
    }).subscribe({
      next: () => {
        this.loading.set(false);
        this.success.set(this.translate.instant('auth.passwordChanged'));
        
        // Clear the must change password flag locally
        this.auth.clearMustChangePassword();
        
        // Redirect after a short delay
        setTimeout(() => {
          this.router.navigate(['/']);
        }, 1500);
      },
      error: (err) => {
        this.loading.set(false);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }

  cancel(): void {
    this.router.navigate(['/']);
  }
}
