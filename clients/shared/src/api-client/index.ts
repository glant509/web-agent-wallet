import { randomHex } from "../wallet-core/index";

export interface RequestTrace {
  traceId: string;
  spanId: string;
}

export class APIError extends Error {
  constructor(public readonly status: number, message: string, public readonly body?: unknown) {
    super(message);
    this.name = "APIError";
  }
}

export class AgentWalletAPIClient {
  constructor(private readonly resolveURL: (resource: string) => string = (resource) => resource) {}

  createTrace(traceId = ""): RequestTrace {
    return {
      traceId: /^[0-9a-f]{32}$/i.test(traceId) ? traceId.toLowerCase() : randomHex(16),
      spanId: randomHex(8)
    };
  }

  request(resource: string, options: RequestInit = {}, traceId = ""): Promise<Response> {
    const trace = this.createTrace(traceId);
    const headers = new Headers(options.headers || {});
    headers.set("traceparent", `00-${trace.traceId}-${trace.spanId}-01`);
    headers.set("X-Trace-ID", trace.traceId);
    headers.set("X-Span-ID", trace.spanId);
    return fetch(this.resolveURL(resource), { ...options, headers });
  }

  async json<T>(resource: string, options: RequestInit = {}, traceId = ""): Promise<T> {
    const response = await this.request(resource, options, traceId);
    const body = await response.json().catch(() => null);
    if (!response.ok) {
      const message = body && typeof body === "object" && "error" in body
        ? String((body as { error: unknown }).error)
        : `Request failed with status ${response.status}`;
      throw new APIError(response.status, message, body);
    }
    return body as T;
  }
}

export function createAPIClient(): AgentWalletAPIClient {
  return new AgentWalletAPIClient((resource) => {
    const platform = (window as Window & { AgentWalletPlatform?: { resolveAPIURL(value: string): string } }).AgentWalletPlatform;
    return platform ? platform.resolveAPIURL(resource) : resource;
  });
}
