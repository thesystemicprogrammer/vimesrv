import { Component, inject, OnInit, signal, computed } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { ApiService, SeriesDetail, SeriesCreditPerson } from '../../core/services/api.service';

const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p';

// Department order for crew grouping
const DEPARTMENT_ORDER = [
  'Directing',
  'Writing',
  'Production',
  'Sound',
  'Camera',
  'Art',
  'Editing',
  'Visual Effects',
  'Costume & Make-Up',
  'Lighting',
  'Crew'
];

interface CrewDepartment {
  name: string;
  members: SeriesCreditPerson[];
}

interface ParsedRole {
  character: string;
  episode_count: number;
}

interface ParsedJob {
  job: string;
  episode_count: number;
}

@Component({
  selector: 'app-series-cast',
  standalone: true,
  template: `
    @if (loading()) {
      <div class="flex justify-center items-center h-screen">
        <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-blue-500"></div>
      </div>
    } @else if (error()) {
      <div class="container mx-auto px-4 py-8">
        <div class="bg-red-500/10 border border-red-500 text-red-400 px-4 py-3 rounded">
          {{ error() }}
        </div>
        <button
          (click)="goBack()"
          class="mt-4 text-blue-400 hover:text-blue-300 transition"
        >
          &larr; Back
        </button>
      </div>
    } @else if (series()) {
      <!-- Backdrop hero section -->
      <div class="relative min-h-[30vh]">
        @if (backdropUrl()) {
          <div
            class="absolute inset-0 bg-cover bg-center"
            [style.background-image]="'url(' + backdropUrl() + ')'"
          ></div>
        }
        <div class="absolute inset-0 bg-gradient-to-t from-slate-900 via-slate-900/80 to-slate-900/30"></div>

        <!-- Back button -->
        <div class="absolute top-4 left-4 z-10">
          <button
            (click)="goBack()"
            class="flex items-center gap-2 px-3 py-2 bg-black/50 hover:bg-black/70 text-white rounded-lg transition"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7"/>
            </svg>
            <span class="text-sm">Back</span>
          </button>
        </div>

        <!-- Title overlay -->
        <div class="absolute bottom-0 left-0 right-0 p-4 md:p-8">
          <div class="container mx-auto">
            <h1 class="text-2xl md:text-3xl font-bold text-white">Cast & Crew</h1>
            <p class="text-slate-300">{{ series()!.name }} ({{ series()!.year }})</p>
          </div>
        </div>
      </div>

      <!-- Main content -->
      <div class="container mx-auto px-4 py-8">
        <!-- Cast -->
        @if (cast().length > 0) {
          <section class="mb-10">
            <h2 class="text-xl font-semibold text-white mb-4 border-b border-slate-700 pb-2">
              Series Cast ({{ cast().length }})
            </h2>
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
              @for (person of cast(); track person.id) {
                <a
                  [href]="getTmdbPersonUrl(person.tmdb_person_id)"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="bg-slate-800 rounded-lg overflow-hidden hover:bg-slate-700 transition group"
                >
                  <div class="aspect-[2/3] bg-slate-700">
                    @if (person.profile_path) {
                      <img
                        [src]="getProfileUrl(person.profile_path)"
                        [alt]="person.name"
                        class="w-full h-full object-cover"
                      />
                    } @else {
                      <div class="w-full h-full flex items-center justify-center text-slate-500">
                        <svg class="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                        </svg>
                      </div>
                    }
                  </div>
                  <div class="p-3">
                    <p class="font-medium text-white text-sm group-hover:text-blue-400 transition">{{ person.name }}</p>
                    <p class="text-slate-400 text-xs truncate">{{ parseRolesDisplay(person.roles) }}</p>
                    <p class="text-slate-500 text-xs mt-1">
                      {{ person.total_episode_count }} {{ person.total_episode_count === 1 ? 'episode' : 'episodes' }}
                    </p>
                  </div>
                </a>
              }
            </div>
          </section>
        }

        <!-- Crew by department -->
        @for (dept of crewByDepartment(); track dept.name) {
          <section class="mb-10">
            <h2 class="text-xl font-semibold text-white mb-4 border-b border-slate-700 pb-2">
              {{ dept.name }} ({{ dept.members.length }})
            </h2>
            <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
              @for (person of dept.members; track person.id) {
                <a
                  [href]="getTmdbPersonUrl(person.tmdb_person_id)"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="bg-slate-800 rounded-lg overflow-hidden hover:bg-slate-700 transition group"
                >
                  <div class="aspect-[2/3] bg-slate-700">
                    @if (person.profile_path) {
                      <img
                        [src]="getProfileUrl(person.profile_path)"
                        [alt]="person.name"
                        class="w-full h-full object-cover"
                      />
                    } @else {
                      <div class="w-full h-full flex items-center justify-center text-slate-500">
                        <svg class="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                        </svg>
                      </div>
                    }
                  </div>
                  <div class="p-3">
                    <p class="font-medium text-white text-sm group-hover:text-blue-400 transition">{{ person.name }}</p>
                    <p class="text-slate-400 text-xs truncate">{{ parseJobsDisplay(person.jobs) }}</p>
                    <p class="text-slate-500 text-xs mt-1">
                      {{ person.total_episode_count }} {{ person.total_episode_count === 1 ? 'episode' : 'episodes' }}
                    </p>
                  </div>
                </a>
              }
            </div>
          </section>
        }
      </div>
    }
  `
})
export class SeriesCastComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly api = inject(ApiService);

  series = signal<SeriesDetail | null>(null);
  cast = signal<SeriesCreditPerson[]>([]);
  crew = signal<SeriesCreditPerson[]>([]);
  loading = signal(true);
  error = signal<string | null>(null);
  backdropUrl = signal<string | null>(null);

  // Group crew by department
  crewByDepartment = computed(() => {
    const crewList = this.crew();
    if (crewList.length === 0) return [];

    // Group by department
    const deptMap = new Map<string, SeriesCreditPerson[]>();

    for (const person of crewList) {
      const dept = person.department || this.inferDepartment(this.parseJobsDisplay(person.jobs));
      if (!deptMap.has(dept)) {
        deptMap.set(dept, []);
      }
      deptMap.get(dept)!.push(person);
    }

    // Sort departments by predefined order
    const result: CrewDepartment[] = [];
    for (const deptName of DEPARTMENT_ORDER) {
      const members = deptMap.get(deptName);
      if (members && members.length > 0) {
        result.push({ name: deptName, members });
        deptMap.delete(deptName);
      }
    }

    // Add any remaining departments not in our list
    for (const [name, members] of deptMap) {
      if (members.length > 0) {
        result.push({ name, members });
      }
    }

    return result;
  });

  ngOnInit(): void {
    const seriesIdParam = this.route.snapshot.paramMap.get('id');
    if (!seriesIdParam || isNaN(Number(seriesIdParam))) {
      this.error.set('Invalid series ID');
      this.loading.set(false);
      return;
    }

    this.loadSeriesAndCredits(Number(seriesIdParam));
  }

  loadSeriesAndCredits(seriesId: number): void {
    this.loading.set(true);
    this.error.set(null);

    // First get the series details
    this.api.getSeriesDetail(seriesId).subscribe({
      next: (response) => {
        const seriesData = response.data;
        this.series.set(seriesData);
        if (seriesData.backdrop_path) {
          this.backdropUrl.set(`${TMDB_IMAGE_BASE}/w1280${seriesData.backdrop_path}`);
        }

        // Then fetch full credits
        this.api.getSeriesCredits(seriesId).subscribe({
          next: (creditsResponse) => {
            this.cast.set(creditsResponse.data.cast || []);
            this.crew.set(creditsResponse.data.crew || []);
            this.loading.set(false);
          },
          error: (err) => {
            // Still show the page, just without credits
            this.cast.set([]);
            this.crew.set([]);
            this.loading.set(false);
          }
        });
      },
      error: (err) => {
        this.error.set(err.error?.error?.message || 'Failed to load series');
        this.loading.set(false);
      }
    });
  }

  // Infer department from job title (fallback when department not provided)
  inferDepartment(job: string): string {
    const jobLower = job.toLowerCase();

    if (jobLower.includes('director') && !jobLower.includes('art') && !jobLower.includes('photography')) {
      return 'Directing';
    }
    if (jobLower.includes('writer') || jobLower.includes('screenplay') || jobLower.includes('story') || jobLower.includes('script')) {
      return 'Writing';
    }
    if (jobLower.includes('producer') || jobLower.includes('production manager') || jobLower.includes('executive')) {
      return 'Production';
    }
    if (jobLower.includes('sound') || jobLower.includes('music') || jobLower.includes('composer') || jobLower.includes('audio')) {
      return 'Sound';
    }
    if (jobLower.includes('camera') || jobLower.includes('cinematograph') || jobLower.includes('photography') || jobLower.includes('gaffer')) {
      return 'Camera';
    }
    if (jobLower.includes('art') || jobLower.includes('design') || jobLower.includes('set ')) {
      return 'Art';
    }
    if (jobLower.includes('edit')) {
      return 'Editing';
    }
    if (jobLower.includes('visual effect') || jobLower.includes('vfx') || jobLower.includes('cgi') || jobLower.includes('animation')) {
      return 'Visual Effects';
    }
    if (jobLower.includes('costume') || jobLower.includes('makeup') || jobLower.includes('make-up') || jobLower.includes('hair') || jobLower.includes('wardrobe')) {
      return 'Costume & Make-Up';
    }
    if (jobLower.includes('light') || jobLower.includes('grip') || jobLower.includes('electric')) {
      return 'Lighting';
    }

    return 'Crew';
  }

  goBack(): void {
    const s = this.series();
    if (s) {
      this.router.navigate(['/series', s.series_metadata_id]);
    } else {
      this.router.navigate(['/library']);
    }
  }

  getProfileUrl(profilePath: string): string {
    return `${TMDB_IMAGE_BASE}/w185${profilePath}`;
  }

  getTmdbPersonUrl(tmdbPersonId: number): string {
    return `https://www.themoviedb.org/person/${tmdbPersonId}`;
  }

  parseRolesDisplay(rolesJson: string | undefined): string {
    if (!rolesJson) return '';
    try {
      const roles = JSON.parse(rolesJson) as ParsedRole[];
      if (roles.length === 0) return '';
      // Show first character, or combine if multiple
      if (roles.length === 1) {
        return roles[0].character || '';
      }
      return roles.map(r => r.character).filter(c => c).join(' / ');
    } catch {
      return '';
    }
  }

  parseJobsDisplay(jobsJson: string | undefined): string {
    if (!jobsJson) return '';
    try {
      const jobs = JSON.parse(jobsJson) as ParsedJob[];
      if (jobs.length === 0) return '';
      // Show first job, or combine if multiple
      if (jobs.length === 1) {
        return jobs[0].job || '';
      }
      return jobs.map(j => j.job).filter(j => j).join(' / ');
    } catch {
      return '';
    }
  }
}
