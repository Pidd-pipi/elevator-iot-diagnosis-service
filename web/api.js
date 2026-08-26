// API 客户端：统一请求后端接口并解析统一响应格式
// { code, message, request_id, data }。

export class ApiError extends Error {
  constructor(status, code, message, requestId) {
    super(message);
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

export async function api(path, options = {}) {
  const opts = {
    method: options.method || 'GET',
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
  };
  if (options.body !== undefined) {
    opts.body = typeof options.body === 'string' ? options.body : JSON.stringify(options.body);
  }
  const resp = await fetch(path, opts);
  let payload = null;
  try {
    payload = await resp.json();
  } catch {
    // 非 JSON 响应（如静态资源/纯文本错误）。
    if (!resp.ok) {
      throw new ApiError(resp.status, resp.status, `HTTP ${resp.status}`, resp.headers.get('X-Request-Id') || '');
    }
    return null;
  }
  if (!resp.ok || (payload && payload.code !== 0)) {
    throw new ApiError(
      resp.status,
      payload ? payload.code : resp.status,
      (payload && payload.message) || `HTTP ${resp.status}`,
      payload ? payload.request_id : '',
    );
  }
  return payload.data;
}
