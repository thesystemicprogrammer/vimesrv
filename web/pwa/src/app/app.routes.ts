import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';

export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () => import('./features/auth/login.component').then(m => m.LoginComponent)
  },
  {
    path: '',
    redirectTo: 'library',
    pathMatch: 'full'
  },
  {
    path: 'library',
    canActivate: [authGuard],
    loadComponent: () => import('./features/library/library.component').then(m => m.LibraryComponent)
  },
  {
    path: 'movie/:id',
    canActivate: [authGuard],
    loadComponent: () => import('./features/library/movie-detail.component').then(m => m.MovieDetailComponent)
  },
  {
    path: 'movie/:id/cast',
    canActivate: [authGuard],
    loadComponent: () => import('./features/library/movie-cast.component').then(m => m.MovieCastComponent)
  },
  {
    path: 'series/:id',
    canActivate: [authGuard],
    loadComponent: () => import('./features/library/series-detail.component').then(m => m.SeriesDetailComponent)
  },
  {
    path: 'play/:id',
    canActivate: [authGuard],
    loadComponent: () => import('./features/player/player.component').then(m => m.PlayerComponent)
  },
  {
    path: '**',
    redirectTo: ''
  }
];
