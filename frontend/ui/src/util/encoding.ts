export class Base64EncodeError extends Error {
  constructor() {
    super(
      'Cannot encode value: contains characters outside the Latin-1 range (codepoint > 0xFF), ' +
        'e.g. the emdash "—", smart quotes, or other non-Latin-1 Unicode. ' +
        'Please replace them with Latin-1 equivalents (ASCII works).'
    );
    this.name = 'Base64EncodeError';
  }
}

export function toBase64(value: string): string {
  for (let i = 0; i < value.length; i++) {
    if (value.charCodeAt(i) > 0xff) {
      throw new Base64EncodeError();
    }
  }
  return btoa(value);
}
