import {HttpContext, HttpContextToken, HttpInterceptorFn} from '@angular/common/http';

const UNTRIMMED_KEYS = new HttpContextToken<readonly string[]>(() => []);

// skipTrim exempts the named body keys from trimming, for a value whose surrounding whitespace is
// significant (a secret value, a certificate, a file). It is the counterpart of the Go `trim:"-"`
// tag, so it names single keys rather than exempting the whole body.
export function skipTrim(key: string, ...more: string[]): HttpContext {
  return new HttpContext().set(UNTRIMMED_KEYS, [key, ...more]);
}

function trimBody(value: unknown, untrimmedKeys: readonly string[]): unknown {
  if (typeof value === 'string') {
    return value.trim();
  }
  if (Array.isArray(value)) {
    return value.map((entry) => trimBody(entry, untrimmedKeys));
  }
  if (isPlainObject(value)) {
    return Object.fromEntries(
      Object.entries(value).map(([key, entry]) => [
        key,
        untrimmedKeys.includes(key) ? entry : trimBody(entry, untrimmedKeys),
      ])
    );
  }
  return value;
}

// Only a plain object or array is safe to rebuild: a FormData, Blob or ArrayBuffer body must reach
// the request untouched, and a Date has to keep its class to serialize correctly.
function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (typeof value !== 'object' || value === null) {
    return false;
  }
  const prototype = Object.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}

export const trimInterceptor: HttpInterceptorFn = (req, next) => {
  if (!Array.isArray(req.body) && !isPlainObject(req.body)) {
    return next(req);
  }
  return next(req.clone({body: trimBody(req.body, req.context.get(UNTRIMMED_KEYS))}));
};
