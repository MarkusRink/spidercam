import type { IncomingMessage, ServerResponse } from "node:http";

export function sendJson(
  res: ServerResponse,
  status: number,
  body: unknown,
): void {
  const payload = JSON.stringify(body);
  res.writeHead(status, {
    "content-type": "application/json",
    "content-length": Buffer.byteLength(payload),
  });
  res.end(payload);
}

export async function readJsonBody<T>(req: IncomingMessage): Promise<T | null> {
  const chunks: Buffer[] = [];
  for await (const chunk of req) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  if (chunks.length === 0) {
    return {} as T;
  }
  try {
    return JSON.parse(Buffer.concat(chunks).toString("utf8")) as T;
  } catch {
    return null;
  }
}

export function notFound(res: ServerResponse): void {
  sendJson(res, 404, { error: "not found" });
}

export function methodNotAllowed(res: ServerResponse): void {
  sendJson(res, 405, { error: "method not allowed" });
}

export function serveSpaStub(res: ServerResponse): void {
  const html = `<!DOCTYPE html><html><head><title>Spidercam</title></head><body><p>Spidercam UI (mock)</p></body></html>`;
  res.writeHead(200, { "content-type": "text/html" });
  res.end(html);
}
