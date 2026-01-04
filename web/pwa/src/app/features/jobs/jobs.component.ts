import { Component, inject, signal, OnInit, OnDestroy, computed } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { TranslateModule, TranslateService } from '@ngx-translate/core';
import { ApiService, Job, JobStatus, JobType } from '../../core/services/api.service';
import { WebSocketService, JobStartedPayload, JobProgressPayload, JobCompletedPayload, JobFailedPayload, JobsQueuedPayload } from '../../core/services/websocket.service';

type FilterTab = 'all' | 'running' | 'queued' | 'completed' | 'failed';

interface JobWithProgress extends Job {
  progress?: {
    percentage?: number;
    frame?: number;
    fps?: number;
    bitrate?: string;
    time?: string;
    speed?: string;
    message?: string;
  };
}

@Component({
  selector: 'app-jobs',
  standalone: true,
  imports: [CommonModule, TranslateModule],
  template: `
    <div class="min-h-screen bg-slate-900 py-8 px-4">
      <div class="max-w-6xl mx-auto">
        <!-- Header -->
        <div class="flex items-center justify-between mb-8">
          <div>
            <h1 class="text-3xl font-bold text-white">{{ 'jobs.title' | translate }}</h1>
            <p class="text-slate-400 mt-1">{{ 'jobs.subtitle' | translate }}</p>
          </div>
          <div class="flex gap-3">
            <button
              (click)="goBack()"
              class="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
            >
              {{ 'admin.back' | translate }}
            </button>
            <button
              (click)="loadJobs()"
              [disabled]="loading()"
              class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors flex items-center gap-2 disabled:opacity-50"
            >
              <svg class="w-5 h-5" [class.animate-spin]="loading()" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
              </svg>
              {{ 'jobs.refresh' | translate }}
            </button>
          </div>
        </div>

        <!-- WebSocket Status -->
        <div class="mb-4 flex items-center gap-2">
          <div 
            class="w-2 h-2 rounded-full"
            [class.bg-green-500]="wsService.isConnected()"
            [class.bg-yellow-500]="wsService.connectionState() === 'connecting' || wsService.connectionState() === 'reconnecting'"
            [class.bg-red-500]="wsService.connectionState() === 'disconnected'"
          ></div>
          <span class="text-sm text-slate-400">
            @if (wsService.isConnected()) {
              {{ 'jobs.wsConnected' | translate }}
            } @else if (wsService.connectionState() === 'connecting' || wsService.connectionState() === 'reconnecting') {
              {{ 'jobs.wsConnecting' | translate }}
            } @else {
              {{ 'jobs.wsDisconnected' | translate }}
            }
          </span>
        </div>

        <!-- Error Message -->
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

        <!-- Filter Tabs -->
        <div class="flex gap-1 mb-6 bg-slate-800 p-1 rounded-lg w-fit">
          @for (tab of tabs; track tab.key) {
            <button
              (click)="activeTab.set(tab.key)"
              class="px-4 py-2 text-sm font-medium rounded-md transition-colors"
              [class.bg-blue-600]="activeTab() === tab.key"
              [class.text-white]="activeTab() === tab.key"
              [class.text-slate-400]="activeTab() !== tab.key"
              [class.hover:text-white]="activeTab() !== tab.key"
            >
              {{ tab.label | translate }}
              <span class="ml-1.5 px-1.5 py-0.5 text-xs rounded-full bg-slate-700">
                {{ getTabCount(tab.key) }}
              </span>
            </button>
          }
        </div>

        <!-- Loading State -->
        @if (loading() && jobs().length === 0) {
          <div class="flex justify-center py-12">
            <svg class="animate-spin h-8 w-8 text-blue-500" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          </div>
        } @else {
          <!-- Jobs List -->
          <div class="space-y-4">
            @for (job of filteredJobs(); track job.id) {
              <div class="bg-slate-800 rounded-lg border border-slate-700 p-4">
                <!-- Job Header -->
                <div class="flex items-start justify-between mb-3">
                  <div class="flex items-center gap-3">
                    <!-- Job Type Icon -->
                    <div [ngClass]="getJobIconClass(job.type)">
                      <ng-container [ngSwitch]="job.type">
                        <svg *ngSwitchCase="'transcode_video'" class="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z"/>
                        </svg>
                        <svg *ngSwitchCase="'transcode_audio'" class="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 4v16M17 4v16M3 8h4m10 0h4M3 12h18M3 16h4m10 0h4M4 20h16a1 1 0 001-1V5a1 1 0 00-1-1H4a1 1 0 00-1 1v14a1 1 0 001 1z"/>
                        </svg>
                        <svg *ngSwitchCase="'scan_library'" class="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/>
                        </svg>
                        <svg *ngSwitchCase="'enrich_metadata'" class="w-5 h-5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"/>
                        </svg>
                        <svg *ngSwitchDefault class="w-5 h-5 text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 5h12M9 3v2m1.048 9.5A18.022 18.022 0 016.412 9m6.088 9h7M11 21l5-10 5 10M12.751 5C11.783 10.77 8.07 15.61 3 18.129"/>
                        </svg>
                      </ng-container>
                    </div>
                    <div>
                      <h3 class="text-white font-medium">
                        {{ getJobTypeLabel(job.type) }}
                        @if (getTranscodeInfo(job); as info) {
                          <span class="text-slate-400 text-sm ml-1">({{ info }})</span>
                        }
                      </h3>
                      <p class="text-sm text-slate-400">{{ 'jobs.jobId' | translate }}: {{ job.id }}</p>
                    </div>
                  </div>
                  <!-- Status Badge -->
                  <span [ngClass]="getStatusClass(job.status)">
                    {{ getStatusLabel(job.status) }}
                  </span>
                </div>

                <!-- Progress Bar (for running transcode jobs) -->
                @if (job.status === 'running' && isTranscodeJob(job.type)) {
                  <div class="mb-3">
                    <!-- Progress message (contains filename) -->
                    @if (job.progress?.message) {
                      <p class="text-sm text-slate-300 mb-2">{{ job.progress?.message }}</p>
                    }
                    <div class="flex justify-between text-sm text-slate-400 mb-1">
                      <span>{{ 'jobs.progress' | translate }}</span>
                      <span>{{ job.progress?.percentage?.toFixed(1) || 0 }}%</span>
                    </div>
                    <div class="w-full h-2 bg-slate-700 rounded-full overflow-hidden">
                      <div class="h-full bg-blue-500 transition-all duration-300" [style.width.%]="job.progress?.percentage || 0"></div>
                    </div>
                    <!-- Progress Details -->
                    @if (job.progress?.fps || job.progress?.speed || job.progress?.bitrate) {
                      <div class="flex flex-wrap gap-4 mt-2 text-xs text-slate-500">
                        @if (job.progress?.time) {
                          <span>{{ 'jobs.time' | translate }}: {{ job.progress?.time }}</span>
                        }
                        @if (job.progress?.fps) {
                          <span>{{ 'jobs.fps' | translate }}: {{ job.progress?.fps }}</span>
                        }
                        @if (job.progress?.speed) {
                          <span>{{ 'jobs.speed' | translate }}: {{ job.progress?.speed }}</span>
                        }
                        @if (job.progress?.bitrate) {
                          <span>{{ 'jobs.bitrate' | translate }}: {{ job.progress?.bitrate }}</span>
                        }
                      </div>
                    }
                  </div>
                }

                <!-- Progress for scan_library job -->
                @if (job.status === 'running' && job.type === 'scan_library') {
                  <div class="mb-3">
                    @if (job.progress?.message) {
                      <p class="text-sm text-slate-300 mb-2">{{ job.progress?.message }}</p>
                    }
                    @if (job.progress?.percentage !== undefined && (job.progress?.percentage ?? -1) >= 0) {
                      <!-- Determinate progress (copying phase) -->
                      <div class="flex justify-between text-sm text-slate-400 mb-1">
                        <span>{{ 'jobs.progress' | translate }}</span>
                        <span>{{ job.progress?.percentage?.toFixed(1) || 0 }}%</span>
                      </div>
                      <div class="w-full h-2 bg-slate-700 rounded-full overflow-hidden">
                        <div class="h-full bg-green-500 transition-all duration-300" [style.width.%]="job.progress?.percentage || 0"></div>
                      </div>
                    } @else {
                      <!-- Indeterminate progress (analyzing phase) -->
                      <div class="flex items-center gap-2">
                        <svg class="animate-spin h-4 w-4 text-green-400" fill="none" viewBox="0 0 24 24">
                          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                        </svg>
                        @if (!job.progress?.message) {
                          <span class="text-sm text-slate-400">{{ 'jobs.processing' | translate }}</span>
                        }
                      </div>
                    }
                  </div>
                }

                <!-- Indeterminate spinner (for running non-transcode, non-scan_library jobs) -->
                @if (job.status === 'running' && !isTranscodeJob(job.type) && job.type !== 'scan_library') {
                  <div class="mb-3 flex items-center gap-2">
                    <svg class="animate-spin h-4 w-4 text-blue-400" fill="none" viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    <span class="text-sm text-slate-400">{{ 'jobs.processing' | translate }}</span>
                  </div>
                }

                <!-- Job Payload Details -->
                @if (getJobDetails(job); as details) {
                  <div class="mb-3 text-sm text-slate-300">
                    @switch (job.type) {
                      @case ('enrich_metadata') {
                        <span class="text-slate-500">{{ 'jobs.file' | translate }}:</span> {{ details }}
                      }
                      @case ('fetch_translations') {
                        <span class="text-slate-500">{{ 'jobs.language' | translate }}:</span> {{ details }}
                      }
                      @case ('transcode_video') {
                        <span class="text-slate-500">{{ 'jobs.transcodeId' | translate }}:</span> {{ details }}
                      }
                      @case ('transcode_audio') {
                        <span class="text-slate-500">{{ 'jobs.transcodeId' | translate }}:</span> {{ details }}
                      }
                    }
                  </div>
                }

                <!-- Timestamps -->
                <div class="flex flex-wrap gap-4 text-sm text-slate-400">
                  <!-- Created (for queued jobs) -->
                  @if (job.status === 'queued') {
                    <span class="flex items-center gap-1">
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
                      </svg>
                      {{ 'jobs.created' | translate }}: {{ formatDate(job.created_at) }}
                    </span>
                  }

                  <!-- Started (for running jobs) -->
                  @if (job.status === 'running' && job.started_at) {
                    <span class="flex items-center gap-1">
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/>
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                      </svg>
                      {{ 'jobs.started' | translate }}: {{ formatDate(job.started_at) }}
                    </span>
                  }

                  <!-- Started + Ended + Duration (for completed/failed jobs) -->
                  @if ((job.status === 'succeeded' || job.status === 'dead') && job.started_at) {
                    <span class="flex items-center gap-1">
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"/>
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                      </svg>
                      {{ 'jobs.started' | translate }}: {{ formatDate(job.started_at) }}
                    </span>
                    @if (job.finished_at) {
                      <span class="flex items-center gap-1">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/>
                        </svg>
                        {{ 'jobs.ended' | translate }}: {{ formatDate(job.finished_at) }}
                      </span>
                      <span class="flex items-center gap-1 text-slate-500">
                        {{ 'jobs.duration' | translate }}: {{ formatDuration(job.started_at, job.finished_at) }}
                      </span>
                    }
                  }

                  <!-- Attempt count (if > 1) -->
                  @if (job.attempts > 1) {
                    <span class="flex items-center gap-1 text-yellow-400">
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/>
                      </svg>
                      {{ 'jobs.attempt' | translate }}: {{ job.attempts }}/{{ job.max_attempts }}
                    </span>
                  }
                </div>

                <!-- Error Message (for failed jobs) -->
                @if (job.status === 'dead' && job.last_error) {
                  <div class="mt-3 p-3 bg-red-500/10 border border-red-500/30 rounded-lg">
                    <p class="text-sm text-red-400 font-mono">{{ job.last_error }}</p>
                  </div>
                }
              </div>
            } @empty {
              <div class="bg-slate-800 rounded-lg border border-slate-700 p-12 text-center">
                <svg class="w-12 h-12 text-slate-600 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"/>
                </svg>
                <p class="text-slate-400">{{ 'jobs.noJobs' | translate }}</p>
              </div>
            }
          </div>
        }
      </div>
    </div>
  `
})
export class JobsComponent implements OnInit, OnDestroy {
  private readonly api = inject(ApiService);
  readonly wsService = inject(WebSocketService);
  private readonly router = inject(Router);
  private readonly translate = inject(TranslateService);

  jobs = signal<JobWithProgress[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);
  activeTab = signal<FilterTab>('all');

  private unsubscribers: (() => void)[] = [];

  tabs: { key: FilterTab; label: string }[] = [
    { key: 'all', label: 'jobs.tabAll' },
    { key: 'running', label: 'jobs.tabRunning' },
    { key: 'queued', label: 'jobs.tabQueued' },
    { key: 'completed', label: 'jobs.tabCompleted' },
    { key: 'failed', label: 'jobs.tabFailed' }
  ];

  // Status priority for sorting: running (0) -> queued (1) -> dead (2) -> succeeded (3)
  private readonly statusPriority: Record<string, number> = {
    'running': 0,
    'queued': 1,
    'dead': 2,
    'succeeded': 3
  };

  private sortJobsByStatus(jobs: JobWithProgress[]): JobWithProgress[] {
    return [...jobs].sort((a, b) => {
      const priorityA = this.statusPriority[a.status] ?? 4;
      const priorityB = this.statusPriority[b.status] ?? 4;
      if (priorityA !== priorityB) {
        return priorityA - priorityB;
      }
      // Within same status, sort by created_at descending (newest first)
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
    });
  }

  filteredJobs = computed(() => {
    const tab = this.activeTab();
    const allJobs = this.jobs();
    
    let filtered: JobWithProgress[];
    switch (tab) {
      case 'running':
        filtered = allJobs.filter(j => j.status === 'running');
        break;
      case 'queued':
        filtered = allJobs.filter(j => j.status === 'queued');
        break;
      case 'completed':
        filtered = allJobs.filter(j => j.status === 'succeeded');
        break;
      case 'failed':
        filtered = allJobs.filter(j => j.status === 'dead');
        break;
      default:
        filtered = allJobs;
    }
    
    return this.sortJobsByStatus(filtered);
  });

  ngOnInit(): void {
    this.loadJobs();
    this.setupWebSocketListeners();
  }

  ngOnDestroy(): void {
    this.unsubscribers.forEach(unsub => unsub());
  }

  private setupWebSocketListeners(): void {
    // Job started - add to list or update status
    this.unsubscribers.push(
      this.wsService.onJobStarted((payload: JobStartedPayload) => {
        this.jobs.update(jobs => {
          const existing = jobs.find(j => j.id === payload.job_id);
          if (existing) {
            return jobs.map(j => 
              j.id === payload.job_id 
                ? { ...j, status: 'running' as JobStatus, attempts: payload.attempt }
                : j
            );
          }
          // Job not in list, add it
          const newJob: JobWithProgress = {
            id: payload.job_id,
            type: payload.job_type as JobType,
            status: 'running' as JobStatus,
            priority: 0,
            attempts: payload.attempt,
            max_attempts: payload.max_attempts,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            started_at: new Date().toISOString()
          };
          return [newJob, ...jobs];
        });
      })
    );

    // Job progress - update progress info
    this.unsubscribers.push(
      this.wsService.onJobProgress((payload: JobProgressPayload) => {
        this.jobs.update(jobs => 
          jobs.map(j => 
            j.id === payload.job_id 
              ? { 
                  ...j, 
                  progress: {
                    percentage: payload.percentage,
                    frame: payload.frame,
                    fps: payload.fps,
                    bitrate: payload.bitrate,
                    time: payload.time,
                    speed: payload.speed,
                    message: payload.message
                  }
                }
              : j
          )
        );
      })
    );

    // Job completed - update status
    this.unsubscribers.push(
      this.wsService.onJobCompleted((payload: JobCompletedPayload) => {
        this.jobs.update(jobs => 
          jobs.map(j => 
            j.id === payload.job_id 
              ? { ...j, status: 'succeeded' as JobStatus, progress: undefined }
              : j
          )
        );
      })
    );

    // Job failed - update status and error
    this.unsubscribers.push(
      this.wsService.onJobFailed((payload: JobFailedPayload) => {
        this.jobs.update(jobs => 
          jobs.map(j => 
            j.id === payload.job_id 
              ? { ...j, status: 'dead' as JobStatus, last_error: payload.error_message, progress: undefined }
              : j
          )
        );
      })
    );

    // Job retrying - update attempt count
    this.unsubscribers.push(
      this.wsService.onJobRetrying((payload) => {
        this.jobs.update(jobs => 
          jobs.map(j => 
            j.id === payload.job_id 
              ? { ...j, status: 'queued' as JobStatus, attempts: payload.attempt, progress: undefined }
              : j
          )
        );
      })
    );

    // Jobs queued - add new jobs to the list
    this.unsubscribers.push(
      this.wsService.onJobsQueued((payload: JobsQueuedPayload) => {
        this.jobs.update(jobs => {
          const newJobs: JobWithProgress[] = payload.jobs
            .filter(qj => !jobs.some(j => j.id === qj.job_id))
            .map(qj => ({
              id: qj.job_id,
              type: qj.job_type as JobType,
              status: 'queued' as JobStatus,
              priority: qj.priority,
              attempts: 0,
              max_attempts: qj.max_attempts,
              created_at: qj.created_at,
              updated_at: qj.created_at
            }));
          return [...newJobs, ...jobs];
        });
      })
    );
  }

  loadJobs(): void {
    this.loading.set(true);
    this.api.listJobs({ includeOld: true }).subscribe({
      next: (response) => {
        // Preserve progress from currently running jobs
        const existingJobs = this.jobs();
        const progressMap = new Map<number, JobWithProgress['progress']>();
        existingJobs.forEach(job => {
          if (job.status === 'running' && job.progress) {
            progressMap.set(job.id, job.progress);
          }
        });

        // Merge new jobs with existing progress for running jobs
        const newJobs: JobWithProgress[] = (response.data.jobs || []).map(job => ({
          ...job,
          progress: job.status === 'running' ? progressMap.get(job.id) : undefined
        }));

        this.jobs.set(newJobs);
        this.loading.set(false);
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Failed to load jobs');
        this.loading.set(false);
      }
    });
  }

  goBack(): void {
    this.router.navigate(['/']);
  }

  getTabCount(tab: FilterTab): number {
    const allJobs = this.jobs();
    switch (tab) {
      case 'running':
        return allJobs.filter(j => j.status === 'running').length;
      case 'queued':
        return allJobs.filter(j => j.status === 'queued').length;
      case 'completed':
        return allJobs.filter(j => j.status === 'succeeded').length;
      case 'failed':
        return allJobs.filter(j => j.status === 'dead').length;
      default:
        return allJobs.length;
    }
  }

  getJobIconClass(type: string): string {
    const baseClass = 'w-10 h-10 rounded-lg flex items-center justify-center';
    switch (type) {
      case 'transcode_video':
      case 'transcode_audio':
        return `${baseClass} bg-blue-500/20`;
      case 'scan_library':
        return `${baseClass} bg-green-500/20`;
      case 'enrich_metadata':
        return `${baseClass} bg-purple-500/20`;
      case 'fetch_translations':
        return `${baseClass} bg-orange-500/20`;
      default:
        return `${baseClass} bg-slate-500/20`;
    }
  }

  getStatusClass(status: string): string {
    const baseClass = 'px-2.5 py-1 text-xs font-medium rounded-full';
    switch (status) {
      case 'queued':
        return `${baseClass} bg-yellow-500/20 text-yellow-400`;
      case 'running':
        return `${baseClass} bg-blue-500/20 text-blue-400`;
      case 'succeeded':
        return `${baseClass} bg-green-500/20 text-green-400`;
      case 'dead':
        return `${baseClass} bg-red-500/20 text-red-400`;
      default:
        return `${baseClass} bg-slate-500/20 text-slate-400`;
    }
  }

  isTranscodeJob(type: string): boolean {
    return type === 'transcode_video' || type === 'transcode_audio';
  }

  getTranscodeInfo(job: JobWithProgress): string | null {
    const transcodeId = job.payload?.['transcode_id'] as string;
    if (!transcodeId) return null;

    if (job.type === 'transcode_video') {
      // Format: "{mediaID}-video-{quality}" e.g., "abc123-video-720p"
      const match = transcodeId.match(/-video-(\d+p)$/);
      return match ? match[1] : null;
    }

    if (job.type === 'transcode_audio') {
      const parts: string[] = [];

      // Get language from payload and convert to display name
      const language = job.payload?.['language'] as string;
      if (language) {
        parts.push(this.getLanguageName(language));
      }

      // Get channel layout from payload, clean up format
      const channelLayout = job.payload?.['channel_layout'] as string;
      if (channelLayout) {
        // Convert "5.1(side)" to "5.1", "stereo" stays as-is
        parts.push(channelLayout.replace(/\(.*\)$/, ''));
      }

      return parts.length > 0 ? parts.join(', ') : null;
    }

    return null;
  }

  getJobTypeLabel(type: string): string {
    const labels: Record<string, string> = {
      'transcode_video': this.translate.instant('jobs.typeTranscodeVideo'),
      'transcode_audio': this.translate.instant('jobs.typeTranscodeAudio'),
      'scan_library': this.translate.instant('jobs.typeScanLibrary'),
      'enrich_metadata': this.translate.instant('jobs.typeEnrichMetadata'),
      'fetch_translations': this.translate.instant('jobs.typeFetchTranslations')
    };
    return labels[type] || type;
  }

  getStatusLabel(status: string): string {
    const labels: Record<string, string> = {
      'queued': this.translate.instant('jobs.statusPending'),
      'running': this.translate.instant('jobs.statusRunning'),
      'succeeded': this.translate.instant('jobs.statusCompleted'),
      'dead': this.translate.instant('jobs.statusFailed')
    };
    return labels[status] || status;
  }

  formatDate(dateStr: string): string {
    const date = new Date(dateStr);
    const locale = this.translate.currentLang === 'de' ? 'de-DE' : 'en-US';
    return date.toLocaleString(locale, {
      dateStyle: 'medium',
      timeStyle: 'short'
    });
  }

  formatDuration(startedAt: string, finishedAt: string): string {
    const start = new Date(startedAt).getTime();
    const end = new Date(finishedAt).getTime();
    const diffMs = end - start;

    if (diffMs < 0) return '0s';

    const hours = Math.floor(diffMs / 3600000);
    const minutes = Math.floor((diffMs % 3600000) / 60000);
    const seconds = Math.floor((diffMs % 60000) / 1000);

    if (hours > 0) return `${hours}h ${minutes}m ${seconds}s`;
    if (minutes > 0) return `${minutes}m ${seconds}s`;
    return `${seconds}s`;
  }

  getJobDetails(job: JobWithProgress): string | null {
    const payload = job.payload;
    if (!payload) return null;

    switch (job.type) {
      case 'enrich_metadata':
        return (payload['filename'] as string) || null;
      case 'fetch_translations':
        return this.getLanguageName(payload['language'] as string);
      case 'transcode_video':
      case 'transcode_audio':
        return (payload['transcode_id'] as string) || null;
      default:
        return null;
    }
  }

  getLanguageName(code: string): string {
    if (!code) return '';
    const locale = this.translate.currentLang === 'de' ? 'de' : 'en';
    try {
      return new Intl.DisplayNames([locale], { type: 'language' }).of(code) || code;
    } catch {
      return code;
    }
  }
}
