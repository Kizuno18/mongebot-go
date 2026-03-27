// IPC service layer for WebSocket JSON-RPC communication with Go backend.

type RPCCallback = (params: unknown) => void;

interface RPCRequest {
  jsonrpc: "2.0";
  method: string;
  params?: unknown;
  id: number;
}

interface RPCResponse {
  jsonrpc: "2.0";
  result?: { method?: string; params?: unknown } | unknown;
  error?: { code: number; message: string };
  id?: number;
}

interface PendingRequest {
  resolve: (value: unknown) => void;
  reject: (reason: Error) => void;
}

class IPCService {
  private ws: WebSocket | null = null;
  private requestId = 0;
  private pending = new Map<number, PendingRequest>();
  private eventHandlers = new Map<string, Set<RPCCallback>>();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;
  private reconnectDelay = 1000;
  private url: string;

  constructor(port: number = 9800) {
    this.url = `ws://127.0.0.1:${port}/ws`;
  }

  // Connect to the backend WebSocket server.
  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(this.url);

      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        console.log("[IPC] Connected to backend");
        resolve();
      };

      this.ws.onmessage = (event) => {
        this.handleMessage(JSON.parse(event.data));
      };

      this.ws.onclose = () => {
        console.log("[IPC] Disconnected");
        this.attemptReconnect();
      };

      this.ws.onerror = (err) => {
        console.error("[IPC] WebSocket error", err);
        reject(new Error("WebSocket connection failed"));
      };
    });
  }

  // Send a JSON-RPC request and wait for the response.
  async call<T = unknown>(method: string, params?: unknown): Promise<T> {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      throw new Error("Not connected to backend");
    }

    const id = ++this.requestId;
    const request: RPCRequest = {
      jsonrpc: "2.0",
      method,
      params,
      id,
    };

    return new Promise<T>((resolve, reject) => {
      this.pending.set(id, {
        resolve: resolve as (value: unknown) => void,
        reject,
      });
      this.ws!.send(JSON.stringify(request));

      // Timeout after 30 seconds
      setTimeout(() => {
        if (this.pending.has(id)) {
          this.pending.delete(id);
          reject(new Error(`Request ${method} timed out`));
        }
      }, 30000);
    });
  }

  // Subscribe to server-pushed events.
  on(event: string, callback: RPCCallback): () => void {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, new Set());
    }
    this.eventHandlers.get(event)!.add(callback);

    // Return unsubscribe function
    return () => {
      this.eventHandlers.get(event)?.delete(callback);
    };
  }

  // Disconnect from the backend.
  disconnect(): void {
    this.maxReconnectAttempts = 0; // Prevent reconnect
    this.ws?.close();
    this.ws = null;
  }

  private handleMessage(msg: RPCResponse): void {
    // Check if it's a response to a pending request
    if (msg.id !== undefined && this.pending.has(msg.id)) {
      const { resolve, reject } = this.pending.get(msg.id)!;
      this.pending.delete(msg.id);

      if (msg.error) {
        reject(new Error(msg.error.message));
      } else {
        resolve(msg.result);
      }
      return;
    }

    // It's a server-pushed event
    const result = msg.result as { method?: string; params?: unknown };
    if (result?.method) {
      const handlers = this.eventHandlers.get(result.method);
      if (handlers) {
        for (const handler of handlers) {
          handler(result.params);
        }
      }
    }
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return;

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.min(this.reconnectAttempts, 5);

    console.log(`[IPC] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);

    setTimeout(() => {
      this.connect().catch(() => {
        // Will retry via onclose handler
      });
    }, delay);
  }
}

// Singleton IPC instance
export const ipc = new IPCService();
