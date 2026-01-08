import { Component, inject, signal, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { TranslateModule, TranslateService } from '@ngx-translate/core';
import { Subject, takeUntil, interval } from 'rxjs';
import {
  ApiService,
  WorkerConfig,
  ListWorkersResponse,
} from '../../core/services/api.service';

type ModalMode = 'delete-worker' | 'set-count' | null;

@Component({
  selector: 'app-workers-admin',
  standalone: true,
  imports: [CommonModule, FormsModule, TranslateModule],
  template: `
    <div class="min-h-screen bg-slate-900 py-8 px-4">
      <div class="max-w-6xl mx-auto">
        <!-- Header -->
        <div class="flex items-center justify-between mb-8">
          <div>
            <h1 class="text-3xl font-bold text-white">{{ 'workersAdmin.title' | translate }}</h1>
            <p class="text-slate-400 mt-1">{{ 'workersAdmin.subtitle' | translate }}</p>
          </div>
          <button
            (click)="goBack()"
            class="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
          >
            {{ 'common.back' | translate }}
          </button>
        </div>

        <!-- Toast Messages -->
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
          <!-- Local Workers Section -->
          <div class="bg-slate-800 rounded-lg border border-slate-700 overflow-hidden mb-6">
            <div class="p-6 border-b border-slate-700">
              <div class="flex items-center justify-between">
                <div>
                  <h2 class="text-xl font-bold text-white">{{ 'workersAdmin.localWorkers' | translate }}</h2>
                  <p class="text-slate-400 text-sm mt-1">{{ 'workersAdmin.localWorkersDesc' | translate }}</p>
                </div>
                <button
                  (click)="openSetCountModal()"
                  class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors flex items-center gap-2"
                >
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/>
                  </svg>
                  {{ 'workersAdmin.setMaxParallelJobs' | translate }} ({{ localWorkerCount() }})
                </button>
              </div>
            </div>

            <!-- Local Workers List -->
            @if (workersData()?.local_workers?.length) {
              <div class="overflow-x-auto">
                <table class="w-full">
                  <thead class="bg-slate-700/50">
                    <tr>
                      <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'workersAdmin.name' | translate }}</th>
                      <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'workersAdmin.acceptsVideo' | translate }}</th>
                      <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'workersAdmin.acceptsAudio' | translate }}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-slate-700">
                    @for (worker of workersData()?.local_workers; track worker.id) {
                      <tr class="hover:bg-slate-700/30">
                        <td class="px-4 py-3 text-white font-medium">{{ worker.name }}</td>
                        <td class="px-4 py-3">
                          <label class="relative inline-flex items-center cursor-pointer">
                            <input
                              type="checkbox"
                              [checked]="worker.accepts_video"
                              (change)="toggleAcceptsVideo(worker)"
                              [disabled]="updatingWorker() === worker.name"
                              class="sr-only peer"
                            />
                            <div class="w-11 h-6 bg-slate-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600 peer-disabled:opacity-50"></div>
                          </label>
                        </td>
                        <td class="px-4 py-3">
                          <label class="relative inline-flex items-center cursor-pointer">
                            <input
                              type="checkbox"
                              [checked]="worker.accepts_audio"
                              (change)="toggleAcceptsAudio(worker)"
                              [disabled]="updatingWorker() === worker.name"
                              class="sr-only peer"
                            />
                            <div class="w-11 h-6 bg-slate-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600 peer-disabled:opacity-50"></div>
                          </label>
                        </td>
                      </tr>
                    }
                  </tbody>
                </table>
              </div>
            } @else {
              <div class="text-center py-8 text-slate-400">
                {{ 'workersAdmin.noLocalWorkers' | translate }}
              </div>
            }
          </div>

          <!-- Distributed Workers Section -->
          <div class="bg-slate-800 rounded-lg border border-slate-700 overflow-hidden">
            <div class="p-6 border-b border-slate-700">
              <h2 class="text-xl font-bold text-white">{{ 'workersAdmin.distributedWorkers' | translate }}</h2>
              <p class="text-slate-400 text-sm mt-1">{{ 'workersAdmin.distributedWorkersDesc' | translate }}</p>
            </div>

            <!-- Distributed Workers List -->
            @if (workersData()?.distributed_workers?.length) {
              <div class="overflow-x-auto">
                <table class="w-full">
                  <thead class="bg-slate-700/50">
                    <tr>
                      <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'workersAdmin.status' | translate }}</th>
                      <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'workersAdmin.name' | translate }}</th>
                      <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'workersAdmin.acceptsVideo' | translate }}</th>
                      <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'workersAdmin.acceptsAudio' | translate }}</th>
                      <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'workersAdmin.activeJobs' | translate }}</th>
                      <th class="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">{{ 'workersAdmin.lastSeen' | translate }}</th>
                      <th class="px-4 py-3 text-right text-xs font-medium text-slate-400 uppercase">{{ 'workersAdmin.actions' | translate }}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-slate-700">
                    @for (worker of workersData()?.distributed_workers; track worker.id) {
                      <tr class="hover:bg-slate-700/30">
                        <td class="px-4 py-3">
                          @if (worker.online) {
                            <span class="inline-flex items-center gap-1.5">
                              <span class="w-2 h-2 rounded-full bg-green-400 animate-pulse"></span>
                              <span class="text-green-400 text-sm">{{ 'workersAdmin.online' | translate }}</span>
                            </span>
                          } @else {
                            <span class="inline-flex items-center gap-1.5">
                              <span class="w-2 h-2 rounded-full bg-slate-500"></span>
                              <span class="text-slate-500 text-sm">{{ 'workersAdmin.offline' | translate }}</span>
                            </span>
                          }
                        </td>
                        <td class="px-4 py-3 text-white font-medium">{{ worker.name }}</td>
                        <td class="px-4 py-3">
                          <label class="relative inline-flex items-center cursor-pointer">
                            <input
                              type="checkbox"
                              [checked]="worker.accepts_video"
                              (change)="toggleAcceptsVideo(worker)"
                              [disabled]="updatingWorker() === worker.name"
                              class="sr-only peer"
                            />
                            <div class="w-11 h-6 bg-slate-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600 peer-disabled:opacity-50"></div>
                          </label>
                        </td>
                        <td class="px-4 py-3">
                          <label class="relative inline-flex items-center cursor-pointer">
                            <input
                              type="checkbox"
                              [checked]="worker.accepts_audio"
                              (change)="toggleAcceptsAudio(worker)"
                              [disabled]="updatingWorker() === worker.name"
                              class="sr-only peer"
                            />
                            <div class="w-11 h-6 bg-slate-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-500 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600 peer-disabled:opacity-50"></div>
                          </label>
                        </td>
                        <td class="px-4 py-3 text-white">{{ worker.active_jobs }}</td>
                        <td class="px-4 py-3 text-slate-400 text-sm">
                          @if (worker.last_seen) {
                            {{ formatLastSeen(worker.last_seen) }}
                          } @else {
                            -
                          }
                        </td>
                        <td class="px-4 py-3 text-right">
                          <button
                            (click)="openDeleteModal(worker)"
                            class="p-1.5 text-slate-400 hover:text-red-400 hover:bg-slate-600 rounded transition-colors"
                            [title]="'workersAdmin.delete' | translate"
                          >
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                            </svg>
                          </button>
                        </td>
                      </tr>
                    }
                  </tbody>
                </table>
              </div>
            } @else {
              <div class="text-center py-8 text-slate-400">
                {{ 'workersAdmin.noDistributedWorkers' | translate }}
              </div>
            }
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
        <div 
          class="bg-slate-800 rounded-lg border border-slate-700 w-full max-w-md shadow-xl"
          (click)="$event.stopPropagation()"
        >
          <!-- Set Count Modal -->
          @if (modalMode() === 'set-count') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">{{ 'workersAdmin.setMaxParallelJobs' | translate }}</h2>
              <form (ngSubmit)="setLocalWorkerCount()">
                <div>
                  <label class="block text-sm font-medium text-slate-300 mb-2">{{ 'workersAdmin.workerCount' | translate }}</label>
                  <input
                    type="number"
                    [(ngModel)]="newWorkerCount"
                    name="count"
                    min="1"
                    max="16"
                    class="w-full px-4 py-3 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  <p class="text-slate-400 text-sm mt-2">{{ 'workersAdmin.countHint' | translate }}</p>
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
                    [disabled]="modalLoading() || newWorkerCount < 1 || newWorkerCount > 16"
                    class="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50"
                  >
                    @if (modalLoading()) {
                      {{ 'common.processing' | translate }}
                    } @else {
                      {{ 'common.save' | translate }}
                    }
                  </button>
                </div>
              </form>
            </div>
          }

          <!-- Delete Worker Modal -->
          @if (modalMode() === 'delete-worker') {
            <div class="p-6">
              <h2 class="text-xl font-bold text-white mb-4">{{ 'workersAdmin.deleteWorker' | translate }}</h2>
              <p class="text-slate-400 mb-4">{{ 'workersAdmin.deleteConfirm' | translate:{ name: selectedWorker()?.name } }}</p>
              <p class="text-yellow-400 text-sm mb-6">{{ 'workersAdmin.deleteHint' | translate }}</p>
              <div class="flex gap-3">
                <button
                  (click)="closeModal()"
                  class="flex-1 px-4 py-2 border border-slate-600 text-slate-300 rounded-md hover:bg-slate-700 transition-colors"
                >
                  {{ 'common.cancel' | translate }}
                </button>
                <button
                  (click)="deleteWorkerConfig()"
                  [disabled]="modalLoading()"
                  class="flex-1 px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 transition-colors disabled:opacity-50"
                >
                  @if (modalLoading()) {
                    {{ 'common.processing' | translate }}
                  } @else {
                    {{ 'common.delete' | translate }}
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
export class WorkersAdminComponent implements OnInit, OnDestroy {
  private readonly api = inject(ApiService);
  private readonly router = inject(Router);
  private readonly translate = inject(TranslateService);
  private readonly destroy$ = new Subject<void>();

  // Data state
  loading = signal(true);
  workersData = signal<ListWorkersResponse | null>(null);
  localWorkerCount = signal(1);

  // Toast messages
  error = signal<string | null>(null);
  success = signal<string | null>(null);

  // Modal state
  modalMode = signal<ModalMode>(null);
  modalLoading = signal(false);
  selectedWorker = signal<WorkerConfig | null>(null);

  // Inline toggle state
  updatingWorker = signal<string | null>(null);

  // Form values
  newWorkerCount = 1;

  ngOnInit(): void {
    this.loadData();

    // Auto-refresh every 30 seconds
    interval(30000)
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => {
        this.refreshData();
      });
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  goBack(): void {
    this.router.navigate(['/']);
  }

  loadData(): void {
    this.loading.set(true);
    this.api.listWorkers().subscribe({
      next: (response) => {
        this.workersData.set(response);
        this.localWorkerCount.set(response.local_workers?.length || 1);
        this.newWorkerCount = response.local_workers?.length || 1;
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
        this.loading.set(false);
      }
    });
  }

  refreshData(): void {
    this.api.listWorkers().subscribe({
      next: (response) => {
        this.workersData.set(response);
        this.localWorkerCount.set(response.local_workers?.length || 1);
      },
      error: () => {
        // Silent failure on refresh
      }
    });
  }

  formatLastSeen(dateStr: string): string {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffSec = Math.floor(diffMs / 1000);

    if (diffSec < 60) {
      return this.translate.instant('workersAdmin.justNow');
    } else if (diffSec < 3600) {
      const mins = Math.floor(diffSec / 60);
      return this.translate.instant('workersAdmin.minutesAgo', { count: mins });
    } else if (diffSec < 86400) {
      const hours = Math.floor(diffSec / 3600);
      return this.translate.instant('workersAdmin.hoursAgo', { count: hours });
    } else {
      return date.toLocaleDateString();
    }
  }

  // Inline toggle handlers
  toggleAcceptsVideo(worker: WorkerConfig): void {
    this.updatingWorker.set(worker.name);
    this.api.updateWorkerConfig(worker.name, {
      accepts_video: !worker.accepts_video
    }).subscribe({
      next: () => {
        this.updatingWorker.set(null);
        this.refreshData();
      },
      error: (err) => {
        this.updatingWorker.set(null);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }

  toggleAcceptsAudio(worker: WorkerConfig): void {
    this.updatingWorker.set(worker.name);
    this.api.updateWorkerConfig(worker.name, {
      accepts_audio: !worker.accepts_audio
    }).subscribe({
      next: () => {
        this.updatingWorker.set(null);
        this.refreshData();
      },
      error: (err) => {
        this.updatingWorker.set(null);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }

  // Modal handlers
  openSetCountModal(): void {
    this.newWorkerCount = this.localWorkerCount();
    this.modalMode.set('set-count');
  }

  openDeleteModal(worker: WorkerConfig): void {
    this.selectedWorker.set(worker);
    this.modalMode.set('delete-worker');
  }

  closeModal(): void {
    this.modalMode.set(null);
    this.selectedWorker.set(null);
  }

  // Actions
  setLocalWorkerCount(): void {
    if (this.newWorkerCount < 1 || this.newWorkerCount > 16) return;

    this.modalLoading.set(true);
    this.api.setLocalWorkerCount(this.newWorkerCount).subscribe({
      next: (response) => {
        this.modalLoading.set(false);
        this.localWorkerCount.set(this.newWorkerCount);
        this.success.set(response.message);
        if (response.warning) {
          this.error.set(response.warning);
        }
        this.closeModal();
        this.loadData();
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }

  deleteWorkerConfig(): void {
    const worker = this.selectedWorker();
    if (!worker) return;

    this.modalLoading.set(true);
    this.api.deleteWorkerConfig(worker.name).subscribe({
      next: () => {
        this.modalLoading.set(false);
        this.success.set(this.translate.instant('workersAdmin.deleteSuccess'));
        this.closeModal();
        this.loadData();
      },
      error: (err) => {
        this.modalLoading.set(false);
        this.error.set(err.error?.error?.message || this.translate.instant('common.error'));
      }
    });
  }
}
