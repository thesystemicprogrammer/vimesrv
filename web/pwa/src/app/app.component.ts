import { Component, inject, effect } from '@angular/core';
import { RouterOutlet, Router, NavigationEnd } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { NavbarComponent } from './shared/components/navbar.component';
import { FooterComponent } from './shared/components/footer.component';
import { AuthService } from './core/services/auth.service';
import { filter, map } from 'rxjs/operators';
import { toSignal } from '@angular/core/rxjs-interop';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, NavbarComponent, FooterComponent],
  template: `
    <div class="min-h-screen flex flex-col">
      <app-navbar />
      <main class="flex-1 max-w-screen-2xl mx-auto w-full">
        <router-outlet />
      </main>
      @if (!isPlayerRoute()) {
        <app-footer />
      }
    </div>
  `,
  styles: [`
    :host {
      display: block;
    }
  `]
})
export class AppComponent {
  private readonly router = inject(Router);
  private readonly translate = inject(TranslateService);
  private readonly auth = inject(AuthService);

  isPlayerRoute = toSignal(
    this.router.events.pipe(
      filter((event): event is NavigationEnd => event instanceof NavigationEnd),
      map((event) => event.urlAfterRedirects.startsWith('/play/'))
    ),
    { initialValue: false }
  );

  constructor() {
    // Set initial language from AuthService
    this.translate.use(this.auth.language());
    
    // Sync TranslateService with AuthService language changes
    effect(() => {
      const lang = this.auth.language();
      this.translate.use(lang);
    });
  }
}
