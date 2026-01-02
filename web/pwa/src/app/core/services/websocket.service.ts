import { Injectable, signal, computed, inject, effect, DestroyRef } from '@angular/core';
import { AuthService } from './auth.service';

// WebSocket message types
export type WebSocketMessageType = 
  | 'job_completed'
  | 'job_failed'
  | 'job_retrying'
  | 'translations_available';

export interface WebSocketMessage<T = unknown> {
  type: WebSocketMessageType;
  payload: T;
}

export interface JobCompletedPayload {
  job_id: number;
  job_type: string;
  payload?: unknown;
}

export interface JobFailedPayload {
  job_id: number;
  job_type: string;
  payload?: unknown;
  error_message: string;
}

export interface JobRetryingPayload {
  job_id: number;
  job_type: string;
  payload?: unknown;
  attempt: number;
  max_attempts: number;
}

export type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';

// Configuration
const RECONNECT_BASE_DELAY_MS = 1000;
const RECONNECT_MAX_DELAY_MS = 30000;
const RECONNECT_MAX_ATTEMPTS = 10;

@Injectable({
  providedIn: 'root'
})
export class WebSocketService {
  private readonly authService = inject(AuthService);
  private readonly destroyRef = inject(DestroyRef);
  
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
  private manualDisconnect = false;

  // Signals for reactive state
  private connectionStateSignal = signal<ConnectionState>('disconnected');
  private lastMessageSignal = signal<WebSocketMessage | null>(null);
  private lastJobCompletedSignal = signal<JobCompletedPayload | null>(null);
  private lastJobFailedSignal = signal<JobFailedPayload | null>(null);
  private lastJobRetryingSignal = signal<JobRetryingPayload | null>(null);

  // Public readonly signals
  readonly connectionState = computed(() => this.connectionStateSignal());
  readonly isConnected = computed(() => this.connectionStateSignal() === 'connected');
  readonly lastMessage = computed(() => this.lastMessageSignal());
  readonly lastJobCompleted = computed(() => this.lastJobCompletedSignal());
  readonly lastJobFailed = computed(() => this.lastJobFailedSignal());
  readonly lastJobRetrying = computed(() => this.lastJobRetryingSignal());

  // Message listeners for subscribers
  private messageCallbacks: Map<WebSocketMessageType, Set<(payload: unknown) => void>> = new Map();

  constructor() {
    // Auto-connect when authenticated, disconnect on logout
    effect(() => {
      const isAuth = this.authService.isAuthenticated();
      const token = this.authService.token();
      
      if (isAuth && token) {
        this.connect();
      } else {
        this.disconnect();
      }
    });

    // Cleanup on destroy
    this.destroyRef.onDestroy(() => {
      this.disconnect();
    });
  }

  /**
   * Connect to the WebSocket server
   */
  connect(): void {
    // Don't reconnect if already connected or connecting
    if (this.ws?.readyState === WebSocket.OPEN || this.ws?.readyState === WebSocket.CONNECTING) {
      return;
    }

    const token = this.authService.token();
    if (!token) {
      console.warn('[WebSocket] Cannot connect: no auth token');
      return;
    }

    this.manualDisconnect = false;
    this.connectionStateSignal.set('connecting');

    // Build WebSocket URL with token
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws?token=${encodeURIComponent(token)}`;

    try {
      this.ws = new WebSocket(wsUrl);
      this.setupWebSocketHandlers();
    } catch (error) {
      console.error('[WebSocket] Failed to create WebSocket:', error);
      this.connectionStateSignal.set('disconnected');
      this.scheduleReconnect();
    }
  }

  /**
   * Disconnect from the WebSocket server
   */
  disconnect(): void {
    this.manualDisconnect = true;
    this.clearReconnectTimeout();
    this.reconnectAttempts = 0;

    if (this.ws) {
      this.ws.close(1000, 'Client disconnecting');
      this.ws = null;
    }

    this.connectionStateSignal.set('disconnected');
  }

  /**
   * Subscribe to a specific message type
   */
  onMessage<T>(type: WebSocketMessageType, callback: (payload: T) => void): () => void {
    if (!this.messageCallbacks.has(type)) {
      this.messageCallbacks.set(type, new Set());
    }
    
    const typedCallback = callback as (payload: unknown) => void;
    this.messageCallbacks.get(type)!.add(typedCallback);

    // Return unsubscribe function
    return () => {
      this.messageCallbacks.get(type)?.delete(typedCallback);
    };
  }

  /**
   * Subscribe to job completed events
   */
  onJobCompleted(callback: (payload: JobCompletedPayload) => void): () => void {
    return this.onMessage('job_completed', callback);
  }

  /**
   * Subscribe to job failed events
   */
  onJobFailed(callback: (payload: JobFailedPayload) => void): () => void {
    return this.onMessage('job_failed', callback);
  }

  /**
   * Subscribe to job retrying events
   */
  onJobRetrying(callback: (payload: JobRetryingPayload) => void): () => void {
    return this.onMessage('job_retrying', callback);
  }

  private setupWebSocketHandlers(): void {
    if (!this.ws) return;

    this.ws.onopen = () => {
      console.log('[WebSocket] Connected');
      this.connectionStateSignal.set('connected');
      this.reconnectAttempts = 0;
    };

    this.ws.onclose = (event) => {
      console.log(`[WebSocket] Disconnected: code=${event.code}, reason=${event.reason}`);
      this.ws = null;
      
      if (!this.manualDisconnect) {
        this.connectionStateSignal.set('reconnecting');
        this.scheduleReconnect();
      } else {
        this.connectionStateSignal.set('disconnected');
      }
    };

    this.ws.onerror = (error) => {
      console.error('[WebSocket] Error:', error);
      // The onclose handler will be called after this
    };

    this.ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as WebSocketMessage;
        this.handleMessage(message);
      } catch (error) {
        console.error('[WebSocket] Failed to parse message:', error);
      }
    };
  }

  private handleMessage(message: WebSocketMessage): void {
    console.log('[WebSocket] Received:', message.type, message.payload);
    
    this.lastMessageSignal.set(message);

    // Update type-specific signals
    switch (message.type) {
      case 'job_completed':
        this.lastJobCompletedSignal.set(message.payload as JobCompletedPayload);
        break;
      case 'job_failed':
        this.lastJobFailedSignal.set(message.payload as JobFailedPayload);
        break;
      case 'job_retrying':
        this.lastJobRetryingSignal.set(message.payload as JobRetryingPayload);
        break;
    }

    // Notify subscribers
    const callbacks = this.messageCallbacks.get(message.type);
    if (callbacks) {
      callbacks.forEach(callback => {
        try {
          callback(message.payload);
        } catch (error) {
          console.error(`[WebSocket] Error in message callback for ${message.type}:`, error);
        }
      });
    }
  }

  private scheduleReconnect(): void {
    if (this.manualDisconnect) return;
    if (this.reconnectAttempts >= RECONNECT_MAX_ATTEMPTS) {
      console.warn('[WebSocket] Max reconnect attempts reached');
      this.connectionStateSignal.set('disconnected');
      return;
    }

    // Exponential backoff with jitter
    const delay = Math.min(
      RECONNECT_BASE_DELAY_MS * Math.pow(2, this.reconnectAttempts) + Math.random() * 1000,
      RECONNECT_MAX_DELAY_MS
    );

    console.log(`[WebSocket] Scheduling reconnect in ${delay}ms (attempt ${this.reconnectAttempts + 1}/${RECONNECT_MAX_ATTEMPTS})`);

    this.reconnectTimeout = setTimeout(() => {
      this.reconnectAttempts++;
      this.connect();
    }, delay);
  }

  private clearReconnectTimeout(): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
  }
}
