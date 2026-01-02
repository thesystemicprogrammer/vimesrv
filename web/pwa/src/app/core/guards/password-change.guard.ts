import { inject } from '@angular/core';
import { Router, CanActivateFn } from '@angular/router';
import { AuthService } from '../services/auth.service';

/**
 * Guard that redirects users to change-password page if they have must_change_password flag set.
 * This should be used on protected routes AFTER authGuard.
 */
export const passwordChangeGuard: CanActivateFn = (route) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  // If not authenticated, let authGuard handle it
  if (!auth.isAuthenticated()) {
    return true;
  }

  // If user must change password and not already on change-password page, redirect
  if (auth.mustChangePassword()) {
    router.navigate(['/change-password']);
    return false;
  }

  return true;
};
