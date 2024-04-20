const CRLF = '\r\n';

const encoder = new TextEncoder();

export async function serializeRequest(req: Request): Promise<ArrayBuffer> {
  const requestLine = `${req.method} ${req.url} HTTP/1.1`;
  const headerFields: string[] = [];
  req.headers.forEach((v, k) => {
    headerFields.push(`${k}: ${v}`);
  });
  const headerStr = requestLine + CRLF + headerFields.join(CRLF) + CRLF + CRLF;
  const header = encoder.encode(headerStr);

  if (req.body === null) {
    return header.buffer;
  }

  const body = await req.arrayBuffer();

  const out = new Uint8Array(header.byteLength + body.byteLength);
  out.set(header, 0);
  out.set(new Uint8Array(body), header.byteLength);

  return out.buffer;
}
