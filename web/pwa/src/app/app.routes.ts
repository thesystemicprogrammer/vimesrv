import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';
import { adminGuard } from './core/guards/admin.guard';
import { managerGuard } from './core/guards/manager.guard';
import { passwordChangeGuard } from './core/guards/password-change.guard';

export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () => import('./features/auth/login.component').then(m => m.LoginComponent)
  },
  {
    path: 'change-password',
    canActivate: [authGuard],
    loadComponent: () => import('./features/auth/change-password.component').then(m => m.ChangePasswordComponent)
  },
  {
    path: '',
    redirectTo: 'library',
    pathMatch: 'full'
  },
  {
    path: 'library',
    canActivate: [authGuard, passwordChangeGuard],
    loadComponent: () => import('./features/library/library.component').then(m => m.LibraryComponent)
  },
  {
    path: 'search',
    canActivate: [authGuard, passwordChangeGuard],
    loadComponent: () => import('./features/search/search-results.component').then(m => m.SearchResultsComponent)
  },
  {
    path: 'movie/:id',
    canActivate: [authGuard, passwordChangeGuard],
    loadComponent: () => import('./features/library/movie-detail.component').then(m => m.MovieDetailComponent)
  },
  {
    path: 'movie/:id/cast',
    canActivate: [authGuard, passwordChangeGuard],
    loadComponent: () => import('./features/library/movie-cast.component').then(m => m.MovieCastComponent)
  },
  {
    path: 'series/:id',
    canActivate: [authGuard, passwordChangeGuard],
    loadComponent: () => import('./features/library/series-detail.component').then(m => m.SeriesDetailComponent)
  },
  {
    path: 'series/:id/cast',
    canActivate: [authGuard, passwordChangeGuard],
    loadComponent: () => import('./features/library/series-cast.component').then(m => m.SeriesCastComponent)
  },
  {
    path: 'play/:id',
    canActivate: [authGuard, passwordChangeGuard],
    loadComponent: () => import('./features/player/player.component').then(m => m.PlayerComponent)
  },
  {
    path: 'admin',
    canActivate: [adminGuard, passwordChangeGuard],
    loadComponent: () => import('./features/admin/admin.component').then(m => m.AdminComponent)
  },
  {
    path: 'jobs',
    canActivate: [managerGuard, passwordChangeGuard],
    loadComponent: () => import('./features/jobs/jobs.component').then(m => m.JobsComponent)
  },
  {
    path: '**',
    redirectTo: ''
  }
];
