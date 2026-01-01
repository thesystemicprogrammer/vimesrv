import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';

export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () => import('./features/auth/login.component').then(m => m.LoginComponent)
  },
  {
    path: '',
    canActivate: [authGuard],
    loadComponent: () => import('./features/library/library.component').then(m => m.LibraryComponent)
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
