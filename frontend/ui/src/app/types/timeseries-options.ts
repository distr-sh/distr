export type TimeseriesOptions = {limit?: number; before?: Date; after?: Date; filter?: string};

export function timeseriesOptionsAsParams(options?: TimeseriesOptions): Record<string, string> {
  const params: Record<string, string> = {};
  if (options?.limit !== undefined) {
    params['limit'] = options.limit.toFixed();
  }
  if (options?.before !== undefined) {
    params['before'] = options.before.toISOString();
  }
  if (options?.after !== undefined) {
    params['after'] = options.after.toISOString();
  }
  if (options?.filter !== undefined && options.filter !== '') {
    params['filter'] = options.filter;
  }
  return params;
}
