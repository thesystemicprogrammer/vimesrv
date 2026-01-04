import { Directive, ElementRef, Input, OnDestroy, OnInit, Renderer2 } from '@angular/core';

/**
 * Directive for lazy loading images with IntersectionObserver.
 * Provides better control over loading behavior and adds a fade-in effect.
 *
 * Usage:
 * <img appLazyLoad [lazySrc]="imageUrl" [lazyAlt]="altText" />
 */
@Directive({
  selector: '[appLazyLoad]',
  standalone: true
})
export class LazyLoadDirective implements OnInit, OnDestroy {
  @Input() lazySrc = '';
  @Input() lazyAlt = '';
  @Input() lazyPlaceholder = '';

  private observer: IntersectionObserver | null = null;
  private hasLoaded = false;

  constructor(
    private readonly el: ElementRef<HTMLImageElement>,
    private readonly renderer: Renderer2
  ) {}

  ngOnInit(): void {
    // Set up initial state
    this.renderer.setStyle(this.el.nativeElement, 'opacity', '0');
    this.renderer.setStyle(this.el.nativeElement, 'transition', 'opacity 0.3s ease-in-out');

    if (this.lazyAlt) {
      this.renderer.setAttribute(this.el.nativeElement, 'alt', this.lazyAlt);
    }

    // Use placeholder if provided
    if (this.lazyPlaceholder) {
      this.renderer.setAttribute(this.el.nativeElement, 'src', this.lazyPlaceholder);
      this.renderer.setStyle(this.el.nativeElement, 'opacity', '1');
    }

    // Check for IntersectionObserver support
    if ('IntersectionObserver' in window) {
      this.setupIntersectionObserver();
    } else {
      // Fallback: load immediately for older browsers
      this.loadImage();
    }
  }

  ngOnDestroy(): void {
    this.disconnectObserver();
  }

  private setupIntersectionObserver(): void {
    const options: IntersectionObserverInit = {
      root: null, // viewport
      rootMargin: '100px', // Start loading 100px before element comes into view
      threshold: 0
    };

    this.observer = new IntersectionObserver((entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting && !this.hasLoaded) {
          this.loadImage();
          this.disconnectObserver();
        }
      });
    }, options);

    this.observer.observe(this.el.nativeElement);
  }

  private loadImage(): void {
    if (this.hasLoaded || !this.lazySrc) {
      return;
    }

    this.hasLoaded = true;

    // Create a new image to preload
    const img = new Image();

    img.onload = () => {
      this.renderer.setAttribute(this.el.nativeElement, 'src', this.lazySrc);
      // Fade in
      this.renderer.setStyle(this.el.nativeElement, 'opacity', '1');
    };

    img.onerror = () => {
      // Keep placeholder or show nothing
      if (!this.lazyPlaceholder) {
        this.renderer.setStyle(this.el.nativeElement, 'opacity', '0');
      }
    };

    img.src = this.lazySrc;
  }

  private disconnectObserver(): void {
    if (this.observer) {
      this.observer.disconnect();
      this.observer = null;
    }
  }
}
