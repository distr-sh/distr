/**
 * Base64-encodes a UTF-8 string. Works for arbitrary Unicode (including chars
 * outside the Latin-1 range like the emdash "—"), unlike a bare `btoa(str)`.
 */
export function toBase64(value: string): string {
  const bytes = new TextEncoder().encode(value);
  let binary = '';
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

/**
 * Inverse of {@link toBase64}: decodes a base64 string of UTF-8 bytes back
 * into a JavaScript string.
 */
export function fromBase64(value: string): string {
  const binary = atob(value);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return new TextDecoder().decode(bytes);
}
