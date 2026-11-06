export function initials(name: string | undefined): string | undefined {
  return name
    ?.split(' ')
    .map((part) => part.charAt(0))
    .join('')
    .toUpperCase()
    .substring(0, 2);
}
