import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { TranslateModule, TranslateService } from '@ngx-translate/core';
import { ApiService, User, UserRole } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';

type ModalMode = 'create' | 'edit' | 'reset-password' | 'delete' | null;

@Component({
  selector: 'app-admin',
  standalone: true,
  imports: [CommonModule, FormsModule, TranslateModule],
  template: `
    <div class="min-h-screen bg-slate-900 py-8 px-4">
      <div class="max-w-6xl mx-auto">
        <!-- Header -->
        <div class="flex items-center justify-between mb-8">
          <div>
            <h1 class="text-3xl font-bold text-white">{{ 'admin.title' | translate }}</h1>
            <p class="text-slate-400 mt-1">{{ 'admin.subtitle' | translate }}</p>
          </div>
          <div class="flex gap-3">
            <button
              (click)="goBack()"
              class="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
            >
              {{ 'admin.back' | translate }}
            </button>
            <button
              (click)="openCreateModal()"
              class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors flex items-center gap-2"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/>
              </svg>
              {{ 'admin.addUser' | translate }}
            </button>
          </div>
        </div>

        <!-- Error/Success Messages -->
        @if (error()) {
          <div class="mb-6 bg-red-500/10 border border-red-500 text-red-400 px-4 py-3 rounded-lg flex items-center justify-between">
            <span>{{ error() }}</span>
            <button (click)="error.set(null)" class="text-red-400 hover:text-red-300">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>
        }

        @if (success()) {
          <div class="mb-6 bg-green-500/10 border border-green-500 text-green-400 px-4 py-3 rounded-lg flex items-center justify-between">
            <span>{{ success() }}</span>
            <button (click)="success.set(null)" class="text-green-400 hover:text-green-300">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
              </svg>
            </button>
          </div>
        }

        <!-- Loading State -->
        @if (loading()) {
          <div class="flex justify-center py-12">
            <svg class="animate-spin h-8 w-8 text-blue-500" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          </div>
        } @else {
          <!-- Users Table -->
          <div class="bg-slate-800 rounded-lg border border-slate-700 overflow-hidden">
            <table class="w-full">
              <thead class="bg-slate-700/50">
                <tr>
                  <th class="px-6 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">{{ 'admin.username' | translate }}</th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">{{ 'admin.role' | translate }}</th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">{{ 'admin.status' | translate }}</th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">{{ 'admin.created' | translate }}</th>
                  <th class="px-6 py-3 text-right text-xs font-medium text-slate-400 uppercase tracking-wider">{{ 'admin.actions' | translate }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-700">
                @for (user of users(); track user.id) {
                  <tr class="hover:bg-slate-700/30 transition-colors">
                    <td class="px-6 py-4 whitespace-nowrap">
                      <div class="flex items-center gap-3">
                        <div class="w-8 h-8 rounded-full bg-slate-600 flex items-center justify-center">
                          <span class="text-sm font-medium text-white">{{ user.username.charAt(0).toUpperCase() }}</span>
                        </div>
                        <div>
                          <span class="text-white font-medium">{{ user.username }}</span>
                          @if (user.id === auth.userId()) {
                            <span class="ml-2 text-xs text-blue-400">({{ 'admin.you' | translate }})</span>
                          }
                        </div>
                      </div>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                      <span 
                        class="px-2 py-1 text-xs font-medium rounded-full"
                        [ngClass]="{
                          'bg-purple-500/20 text-purple-400': user.role === 'admin',
                          'bg-blue-500/20 text-blue-400': user.role === 'manager',
                          'bg-slate-500/20 text-slate-400': user.role === 'user'
                        }"
                      >
                        {{ user.role }}
                      </span>
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap">
                      @if (user.must_change_password) {
                        <span class="text-yellow-400 text-sm flex items-center gap-1">
                          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
                          </svg>
                          {{ 'admin.mustChangePassword' | translate }}
                        </span>
                      } @else {
                        <span class="text-green-400 text-sm">{{ 'admin.active' | translate }}</span>
                      }
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-slate-400 text-sm">
                      {{ formatDate(user.created_at) }}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-right">
                      <div class="flex items-center justify-end gap-2">
                        <button
                          (click)="openEditModal(user)"
                          class="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors"
                          title="Edit role"
                        >
                          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                          </svg>
                        </button>
                        <button
                          (click)="openResetPasswordModal(user)"
                          class="p-1.5 text-slate-400 hover:text-yellow-400 hover:bg-slate-600 rounded transition-colors"
                          title="Reset password"
                        >
                          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/>
                          </svg>
                        </button>
                        <button
                          (click)="openDeleteModal(user)"
                          [disabled]="user.id === auth.userId()"
                          class="p-1.5 text-slate-400 hover:text-red-400 hover:bg-slate-600 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                          title="Delete user"
                        >
                          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                          </svg>
                        </button>
                      </div>
                    </td>
                  </tr>
                } @empty {
                  <tr>
                    <td colspan="5" class="px-6 py-12 text-center text-slate-400">
                      {{ 'admin.noUsersFound' | translate }}
                    </td>
                  </tr>
                }
              </tbody>
            </table>
          </div>
        }
      </div>
    </div>

    <!-- Modal Backdrop -->
    @if (modalMode()) {
      <div 
        class="fixed inset-0 bg-black/50 z-40 flex items-center justify-center p-4"
        (click)="closeModal()"
      >
        <!-- Modal Content -->
        <div 
          class="bg-slate-800 rounded-lg border border-slate-700 w-full max-w-md shadow-xl"
          (click)="$event.stopPropagation()"
        >
          <!-- Create User Modal -->
          @if (modalMode() === 'create') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">{{ 'admin.createUser' | translate }}</h2>
              <form (ngSubmit)="createUser()">
                <div class="space-y-4">
                  <div>
                    <label class="block text-sm font-medium text-slate-300 mb-1">{{ 'admin.username' | translate }}</label>
                    <input
                      type="text"
                      [(ngModel)]="formUsername"
                      name="username"
                      required
                      class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                      [placeholder]="'admin.enterUsername' | translate"
                    />
                  </div>
                  <div>
                    <label class="block text-sm font-medium text-slate-300 mb-1">{{ 'admin.password' | translate }}</label>
                    <input
                      type="password"
                      [(ngModel)]="formPassword"
                      name="password"
                      required
                      minlength="8"
                      class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                      [placeholder]="'admin.enterPassword' | translate"
                    />
                  </div>
                  <div>
                    <label class="block text-sm font-medium text-slate-300 mb-1">{{ 'admin.role' | translate }}</label>
                    <select
                      [(ngModel)]="formRole"
                      name="role"
                      class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="user">{{ 'admin.roleUser' | translate }}</option>
                      <option value="manager">{{ 'admin.roleManager' | translate }}</option>
                      <option value="admin">{{ 'admin.roleAdmin' | translate }}</option>
                    </select>
                  </div>
                </div>
                <div class="flex gap-3 mt-6">
                  <button
                    type="button"
                    (click)="closeModal()"
                    class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                  >
                    {{ 'common.cancel' | translate }}
                  </button>
                  <button
                    type="submit"
                    [disabled]="modalLoading()"
                    class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50"
                  >
                    @if (modalLoading()) {
                      {{ 'admin.creating' | translate }}
                    } @else {
                      {{ 'admin.createUser' | translate }}
                    }
                  </button>
                </div>
              </form>
            </div>
          }

          <!-- Edit User Modal -->
          @if (modalMode() === 'edit') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">{{ 'admin.editUserRole' | translate }}</h2>
              <p class="text-slate-400 mb-4">{{ 'admin.editing' | translate }}: <span class="text-white font-medium">{{ selectedUser()?.username }}</span></p>
              <form (ngSubmit)="updateUser()">
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1">{{ 'admin.role' | translate }}</label>
                  <select
                    [(ngModel)]="formRole"
                    name="role"
                    class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="user">{{ 'admin.roleUser' | translate }}</option>
                    <option value="manager">{{ 'admin.roleManager' | translate }}</option>
                    <option value="admin">{{ 'admin.roleAdmin' | translate }}</option>
                  </select>
                </div>
                <div class="flex gap-3 mt-6">
                  <button
                    type="button"
                    (click)="closeModal()"
                    class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                  >
                    {{ 'common.cancel' | translate }}
                  </button>
                  <button
                    type="submit"
                    [disabled]="modalLoading()"
                    class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50"
                  >
                    @if (modalLoading()) {
                      {{ 'admin.saving' | translate }}
                    } @else {
                      {{ 'admin.saveChanges' | translate }}
                    }
                  </button>
                </div>
              </form>
            </div>
          }

          <!-- Reset Password Modal -->
          @if (modalMode() === 'reset-password') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">{{ 'admin.resetPassword' | translate }}</h2>
              <p class="text-slate-400 mb-4">{{ 'admin.resetPassword' | translate }}: <span class="text-white font-medium">{{ selectedUser()?.username }}</span></p>
              <form (ngSubmit)="resetPassword()">
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1">{{ 'admin.newPassword' | translate }}</label>
                  <input
                    type="password"
                    [(ngModel)]="formPassword"
                    name="password"
                    required
                    minlength="8"
                    class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    [placeholder]="'admin.enterNewPassword' | translate"
                  />
                </div>
                <p class="mt-2 text-sm text-yellow-400">
                  {{ 'admin.passwordMustChange' | translate }}
                </p>
                <div class="flex gap-3 mt-6">
                  <button
                    type="button"
                    (click)="closeModal()"
                    class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                  >
                    {{ 'common.cancel' | translate }}
                  </button>
                  <button
                    type="submit"
                    [disabled]="modalLoading()"
                    class="flex-1 px-4 py-2 bg-yellow-600 text-white rounded-md hover:bg-yellow-700 transition-colors disabled:opacity-50"
                  >
                    @if (modalLoading()) {
                      {{ 'admin.resetting' | translate }}
                    } @else {
                      {{ 'admin.resetPassword' | translate }}
                    }
                  </button>
                </div>
              </form>
            </div>
          }

          <!-- Delete User Modal -->
          @if (modalMode() === 'delete') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">{{ 'admin.deleteUser' | translate }}</h2>
              <p class="text-slate-400 mb-4">
                {{ 'admin.confirmDelete' | translate }} <span class="text-white font-medium">{{ selectedUser()?.username }}</span>?
              </p>
              <p class="text-red-400 text-sm mb-6">{{ 'admin.actionCannotBeUndone' | translate }}</p>
              <div class="flex gap-3">
                <button
                  (click)="closeModal()"
                  class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                >
                  {{ 'common.cancel' | translate }}
                </button>
                <button
                  (click)="deleteUser()"
                  [disabled]="modalLoading()"
                  class="flex-1 px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 transition-colors disabled:opacity-50"
                >
                  @if (modalLoading()) {
                    {{ 'admin.deleting' | translate }}
                  } @else {
                    {{ 'admin.deleteUser' | translate }}
                  }
                </button>
              </div>
            </div>
          }
        </div>
      </div>
    }
  `
})
export class AdminComponent implements OnInit {
  private readonly api = inject(ApiService);
  readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly translate = inject(TranslateService);

  users = signal<User[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);
  success = signal<string | null>(null);

  modalMode = signal<ModalMode>(null);
  modalLoading = signal(false);
  selectedUser = signal<User | null>(null);

  // Form fields
  formUsername = '';
  formPassword = '';
  formRole: UserRole = 'user';

  ngOnInit(): void {
    this.loadUsers();
  }

  loadUsers(): void {
    this.loading.set(true);
    this.api.listUsers().subscribe({
      next: (response) => {
        this.users.set(response.data);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Failed to load users');
        this.loading.set(false);
      }
    });
  }

  goBack(): void {
    this.router.navigate(['/']);
  }

  formatDate(dateStr: string): string {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    });
  }

  // Modal handlers
  openCreateModal(): void {
    this.formUsername = '';
    this.formPassword = '';
    this.formRole = 'user';
    this.modalMode.set('create');
  }

  openEditModal(user: User): void {
    this.selectedUser.set(user);
    this.formRole = user.role;
    this.modalMode.set('edit');
  }

  openResetPasswordModal(user: User): void {
    this.selectedUser.set(user);
    this.formPassword = '';
    this.modalMode.set('reset-password');
  }

  openDeleteModal(user: User): void {
    this.selectedUser.set(user);
    this.modalMode.set('delete');
  }

  closeModal(): void {
    this.modalMode.set(null);
    this.selectedUser.set(null);
    this.formUsername = '';
    this.formPassword = '';
    this.formRole = 'user';
  }

  // CRUD operations
  createUser(): void {
    if (!this.formUsername || !this.formPassword) {
      this.error.set(this.translate.instant('admin.fillAllFields'));
      return;
    }

    if (this.formPassword.length < 8) {
      this.error.set(this.translate.instant('admin.passwordMinLength'));
      return;
    }

    this.modalLoading.set(true);
    this.api.createUser({
      username: this.formUsername,
      password: this.formPassword,
      role: this.formRole
    }).subscribe({
      next: () => {
        this.modalLoading.set(false);
        this.success.set(this.translate.instant('admin.userCreated', { username: this.formUsername }));
        this.closeModal();
        this.loadUsers();
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }

  updateUser(): void {
    const user = this.selectedUser();
    if (!user) return;

    this.modalLoading.set(true);
    this.api.updateUser(user.id, { role: this.formRole }).subscribe({
      next: () => {
        this.modalLoading.set(false);
        this.success.set(this.translate.instant('admin.userUpdated', { username: user.username }));
        this.closeModal();
        this.loadUsers();
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }

  resetPassword(): void {
    const user = this.selectedUser();
    if (!user || !this.formPassword) return;

    if (this.formPassword.length < 8) {
      this.error.set(this.translate.instant('admin.passwordMinLength'));
      return;
    }

    this.modalLoading.set(true);
    this.api.resetUserPassword(user.id, { new_password: this.formPassword }).subscribe({
      next: () => {
        this.modalLoading.set(false);
        this.success.set(this.translate.instant('admin.passwordReset', { username: user.username }));
        this.closeModal();
        this.loadUsers();
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }

  deleteUser(): void {
    const user = this.selectedUser();
    if (!user) return;

    this.modalLoading.set(true);
    this.api.deleteUser(user.id).subscribe({
      next: () => {
        this.modalLoading.set(false);
        this.success.set(this.translate.instant('admin.userDeleted', { username: user.username }));
        this.closeModal();
        this.loadUsers();
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }
}
