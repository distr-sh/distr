/**
 * Thrown by {@link toBase64} when the input contains characters that the
 * platform's `btoa` cannot encode (codepoints outside the Latin-1 range).
 *
 * Common culprits: the emdash "—" (U+2014), smart quotes ("" '' "" ''), and
 * other typographic punctuation that text editors silently insert.
 */
export class Base64EncodeError extends Error {
  constructor(
    public readonly field: string,
    cause: unknown
  ) {
    super(
      `Cannot base64-encode field "${field}": value contains characters outside the Latin-1 range ` +
        `(codepoint > 0xFF). Distr's deployment configuration only supports Latin-1 characters. ` +
        `Please replace non-ASCII characters such as the emdash "—" or smart quotes with plain ASCII equivalents.`
    );
    this.name = 'Base64EncodeError';
    this.cause = cause;
  }
}

/**
 * Wraps `btoa` and rethrows the InvalidCharacterError as a typed
 * {@link Base64EncodeError} with a descriptive message identifying the field
 * that failed to encode.
 */
export function toBase64(value: string, field: string): string {
  try {
    return btoa(value);
  } catch (e) {
    throw new Base64EncodeError(field, e);
  }
}
