export function formatRemoteAddress(addr: string): string {
  if (addr.includes(']')) {
    // IPv6
    return addr.substring(0, addr.lastIndexOf(']') + 1);
  } else if (addr.includes(':')) {
    // IPv4
    return addr.substring(0, addr.lastIndexOf(':'));
  } else {
    // fallback for undetermined format
    return addr;
  }
}
