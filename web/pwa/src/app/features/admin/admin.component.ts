import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ApiService, User, UserRole } from '../../core/services/api.service';
import { AuthService } from '../../core/services/auth.service';

type ModalMode = 'create' | 'edit' | 'reset-password' | 'delete' | null;

@Component({
  selector: 'app-admin',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="min-h-screen bg-slate-900 py-8 px-4">
      <div class="max-w-6xl mx-auto">
        <!-- Header -->
        <div class="flex items-center justify-between mb-8">
          <div>
            <h1 class="text-3xl font-bold text-white">User Management</h1>
            <p class="text-slate-400 mt-1">Manage user accounts and permissions</p>
          </div>
          <div class="flex gap-3">
            <button
              (click)="goBack()"
              class="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
            >
              Back
            </button>
            <button
              (click)="openCreateModal()"
              class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors flex items-center gap-2"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/>
              </svg>
              Add User
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
                  <th class="px-6 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Username</th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Role</th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Status</th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Created</th>
                  <th class="px-6 py-3 text-right text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
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
                            <span class="ml-2 text-xs text-blue-400">(you)</span>
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
                          Must change password
                        </span>
                      } @else {
                        <span class="text-green-400 text-sm">Active</span>
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
                      No users found
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
              <h2 class="text-xl font-bold text-white mb-4">Create User</h2>
              <form (ngSubmit)="createUser()">
                <div class="space-y-4">
                  <div>
                    <label class="block text-sm font-medium text-slate-300 mb-1">Username</label>
                    <input
                      type="text"
                      [(ngModel)]="formUsername"
                      name="username"
                      required
                      class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="Enter username"
                    />
                  </div>
                  <div>
                    <label class="block text-sm font-medium text-slate-300 mb-1">Password</label>
                    <input
                      type="password"
                      [(ngModel)]="formPassword"
                      name="password"
                      required
                      minlength="8"
                      class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="Enter password (min 8 characters)"
                    />
                  </div>
                  <div>
                    <label class="block text-sm font-medium text-slate-300 mb-1">Role</label>
                    <select
                      [(ngModel)]="formRole"
                      name="role"
                      class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                    >
                      <option value="user">User</option>
                      <option value="manager">Manager</option>
                      <option value="admin">Admin</option>
                    </select>
                  </div>
                </div>
                <div class="flex gap-3 mt-6">
                  <button
                    type="button"
                    (click)="closeModal()"
                    class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    [disabled]="modalLoading()"
                    class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50"
                  >
                    @if (modalLoading()) {
                      Creating...
                    } @else {
                      Create User
                    }
                  </button>
                </div>
              </form>
            </div>
          }

          <!-- Edit User Modal -->
          @if (modalMode() === 'edit') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">Edit User Role</h2>
              <p class="text-slate-400 mb-4">Editing: <span class="text-white font-medium">{{ selectedUser()?.username }}</span></p>
              <form (ngSubmit)="updateUser()">
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1">Role</label>
                  <select
                    [(ngModel)]="formRole"
                    name="role"
                    class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="user">User</option>
                    <option value="manager">Manager</option>
                    <option value="admin">Admin</option>
                  </select>
                </div>
                <div class="flex gap-3 mt-6">
                  <button
                    type="button"
                    (click)="closeModal()"
                    class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    [disabled]="modalLoading()"
                    class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50"
                  >
                    @if (modalLoading()) {
                      Saving...
                    } @else {
                      Save Changes
                    }
                  </button>
                </div>
              </form>
            </div>
          }

          <!-- Reset Password Modal -->
          @if (modalMode() === 'reset-password') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">Reset Password</h2>
              <p class="text-slate-400 mb-4">Reset password for: <span class="text-white font-medium">{{ selectedUser()?.username }}</span></p>
              <form (ngSubmit)="resetPassword()">
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-1">New Password</label>
                  <input
                    type="password"
                    [(ngModel)]="formPassword"
                    name="password"
                    required
                    minlength="8"
                    class="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="Enter new password (min 8 characters)"
                  />
                </div>
                <p class="mt-2 text-sm text-yellow-400">
                  The user will be required to change this password on next login.
                </p>
                <div class="flex gap-3 mt-6">
                  <button
                    type="button"
                    (click)="closeModal()"
                    class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    [disabled]="modalLoading()"
                    class="flex-1 px-4 py-2 bg-yellow-600 text-white rounded-md hover:bg-yellow-700 transition-colors disabled:opacity-50"
                  >
                    @if (modalLoading()) {
                      Resetting...
                    } @else {
                      Reset Password
                    }
                  </button>
                </div>
              </form>
            </div>
          }

          <!-- Delete User Modal -->
          @if (modalMode() === 'delete') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">Delete User</h2>
              <p class="text-slate-400 mb-4">
                Are you sure you want to delete <span class="text-white font-medium">{{ selectedUser()?.username }}</span>?
              </p>
              <p class="text-red-400 text-sm mb-6">This action cannot be undone.</p>
              <div class="flex gap-3">
                <button
                  (click)="closeModal()"
                  class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                >
                  Cancel
                </button>
                <button
                  (click)="deleteUser()"
                  [disabled]="modalLoading()"
                  class="flex-1 px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 transition-colors disabled:opacity-50"
                >
                  @if (modalLoading()) {
                    Deleting...
                  } @else {
                    Delete User
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
      this.error.set('Please fill in all fields');
      return;
    }

    if (this.formPassword.length < 8) {
      this.error.set('Password must be at least 8 characters');
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
        this.success.set(`User "${this.formUsername}" created successfully`);
        this.closeModal();
        this.loadUsers();
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || 'Failed to create user');
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
        this.success.set(`User "${user.username}" updated successfully`);
        this.closeModal();
        this.loadUsers();
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || 'Failed to update user');
      }
    });
  }

  resetPassword(): void {
    const user = this.selectedUser();
    if (!user || !this.formPassword) return;

    if (this.formPassword.length < 8) {
      this.error.set('Password must be at least 8 characters');
      return;
    }

    this.modalLoading.set(true);
    this.api.resetUserPassword(user.id, { new_password: this.formPassword }).subscribe({
      next: () => {
        this.modalLoading.set(false);
        this.success.set(`Password reset for "${user.username}"`);
        this.closeModal();
        this.loadUsers();
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || 'Failed to reset password');
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
        this.success.set(`User "${user.username}" deleted successfully`);
        this.closeModal();
        this.loadUsers();
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || 'Failed to delete user');
      }
    });
  }
}
