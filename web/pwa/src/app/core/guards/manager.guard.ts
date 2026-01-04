import { inject } from '@angular/core';
import { Router, CanActivateFn } from '@angular/router';
import { AuthService } from '../services/auth.service';

/**
 * Guard that allows access for admin or manager roles.
 * Redirects to login if not authenticated, or to home if not authorized.
 */
export const managerGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  if (!auth.isAuthenticated()) {
    router.navigate(['/login']);
    return false;
  }

  if (!auth.canManageLibrary()) {
    // Redirect non-managers to home page
    router.navigate(['/']);
    return false;
  }

  return true;
};
