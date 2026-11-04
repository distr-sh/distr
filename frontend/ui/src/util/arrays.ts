export type Predicate<T, R> = (arg: T) => R;

export function distinctBy<T>(predicate: Predicate<T, unknown>): (input: T[]) => T[] {
  return (input) =>
    input.filter((value: T, index, self) => {
      return self.findIndex((element) => predicate(element) === predicate(value)) === index;
    });
}

export function compareBy<T>(predicate: Predicate<T, string>, inverted: boolean = false): (a: T, b: T) => number {
  const mod = inverted ? -1 : 1;
  return (a, b) => mod * predicate(a).localeCompare(predicate(b));
}

export function arraysEqual<T>(a: T[], b: T[]): boolean {
  return a.length === b.length && a.every((value, index) => value === b[index]);
}

export function maxBy<T, E>(
  input: T[],
  predicate: Predicate<T, E>,
  cmp: (a: E, b: E) => boolean = (a, b) => a > b
): T | undefined {
  let max: T | undefined;
  let maxp: E | undefined;
  for (const el of input) {
    const elp = predicate(el);
    if (maxp === undefined || cmp(elp, maxp)) {
      max = el;
      maxp = elp;
    }
  }
  return max;
}
