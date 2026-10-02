/** A `FormRecord<boolean>` of checkboxes maps to the ids it has checked, and back. */
export function checkedRecord(ids: string[]): Record<string, boolean> {
  return Object.fromEntries(ids.map((id) => [id, true]));
}

export function checkedIds(record: Partial<Record<string, boolean>>): string[] {
  return Object.entries(record)
    .filter(([, checked]) => checked)
    .map(([id]) => id);
}
