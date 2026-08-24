export const slugPattern = /^[a-z0-9]+((\.|_|__|-+)[a-z0-9]+)*$/;
export const slugMaxLength = 64;

/**
 * Every run of characters that may not appear in a slug becomes a single hyphen, so the result always
 * satisfies `slugPattern`. It is empty when the name has no alphanumerics at all.
 */
export function toSlug(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .slice(0, slugMaxLength)
    .replace(/^-+|-+$/g, '');
}
