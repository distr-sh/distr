const shortHexLength = 10;

/**
 * Shortens an OCI digest (e.g. `sha256:abcdef…`) to its algorithm and the first
 * few hex characters. Values that are not digests, such as tag names, are
 * returned unchanged.
 */
export function shortDigest(digest: string): string {
  const separator = digest.indexOf(':');
  if (separator < 0) {
    return digest;
  }
  return digest.substring(0, separator + 1 + shortHexLength);
}
