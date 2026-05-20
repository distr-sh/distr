export class Base64EncodeError extends Error {
  constructor(cause: unknown) {
    super(
      'Cannot encode value: contains characters outside the Latin-1 range (e.g. emdash "—", smart quotes, or other Unicode). ' +
        'Please replace them with plain ASCII equivalents.'
    );
    this.name = 'Base64EncodeError';
    this.cause = cause;
  }
}

export function toBase64(value: string): string {
  try {
    return btoa(value);
  } catch (e) {
    throw new Base64EncodeError(e);
  }
}
