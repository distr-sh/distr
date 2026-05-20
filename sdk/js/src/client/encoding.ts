/**
 * Thrown by {@link toBase64} when the input contains characters that the
 * platform's `btoa` cannot encode (codepoints outside the Latin-1 range,
 * i.e. > 0xFF).
 *
 * Common culprits: the emdash "—" (U+2014), smart quotes (" " ' '), and
 * other typographic punctuation that text editors silently insert.
 */
export class Base64EncodeError extends Error {
  constructor(public readonly field: string) {
    super(
      `Cannot base64-encode field "${field}": value contains characters outside the Latin-1 range ` +
        `(codepoint > 0xFF). Distr's deployment configuration only supports Latin-1 characters. ` +
        `Please replace non-Latin-1 characters such as the emdash "—" or smart quotes with Latin-1 equivalents ` +
        `(ASCII works).`
    );
    this.name = 'Base64EncodeError';
  }
}

/**
 * Encodes a string as base64. Pre-validates the input against the Latin-1
 * range that `btoa` accepts; if the input is out of range, throws a typed
 * {@link Base64EncodeError} naming the field. Any other errors thrown by
 * the platform's `btoa` (e.g. it being unavailable) propagate unchanged.
 */
export function toBase64(value: string, field: string): string {
  for (let i = 0; i < value.length; i++) {
    if (value.charCodeAt(i) > 0xff) {
      throw new Base64EncodeError(field);
    }
  }
  return btoa(value);
}
