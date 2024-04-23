const CRLF = '\r\n';
const CRLF_ENCODED = CRLF.split('').map((s) => s.charCodeAt(0));

const encoder = new TextEncoder();
const decoder = new TextDecoder();

export async function serializeRequest(req: Request): Promise<ArrayBuffer> {
  const url = new URL(req.url);
  url.hash = '';

  const requestLine = `${req.method} ${url.toString()} HTTP/1.1`;
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

export function deserializeResponse(serialized: ArrayBuffer): Response {
  const arr = new Uint8Array(serialized);
  const headerEndIndex = findHeaderEndIndex(arr);
  if (headerEndIndex === -1) {
    return Response.error();
  }

  const header = arr.subarray(0, headerEndIndex);
  const body = arr.subarray(headerEndIndex + 4);

  const headerStr = decoder.decode(header);
  const { status, statusText, headersList } = parseHeader(headerStr);

  return new Response(body, {
    status,
    statusText,
    headers: new Headers(headersList),
  });
}

function findHeaderEndIndex(arr: Uint8Array): number {
  const [cr, lf] = CRLF_ENCODED;
  for (let i = 0; i < arr.length - 3; i++) {
    const isHeaderEnd =
      arr[i] === cr && arr[i + 1] === lf && arr[i + 2] === cr && arr[i + 3] === lf;
    if (isHeaderEnd) {
      return i;
    }
  }

  return -1;
}

function parseHeader(header: string) {
  const [requestLine, ...headerFieldLines] = header.split(CRLF);

  const [_, statusStr, ...statusTextParts] = requestLine.split(' ');
  const status = parseInt(statusStr);
  const statusText = statusTextParts.join(' ');

  const headersList = headerFieldLines.map<[string, string]>((s) => {
    const [name, ...valueParts] = s.split(':');
    return [name, valueParts.join(':').slice(1)];
  });

  return { status, statusText, headersList };
}
