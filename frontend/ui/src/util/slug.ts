export const slugPattern = /^[a-z0-9]+((\.|_|__|-+)[a-z0-9]+)*$/;
export const slugMaxLength = 64;

export function toSlug(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .slice(0, slugMaxLength)
    .replace(/^-+|-+$/g, '');
}
