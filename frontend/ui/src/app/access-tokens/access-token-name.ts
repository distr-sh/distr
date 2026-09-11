import {AccessToken} from '@distr-sh/distr-sdk';

// A label is optional, so a token that has none is named after the beginning of its id, which is
// also what the list and the detail page show to tell two unlabeled tokens apart.
export function accessTokenName(token: AccessToken): string {
  return token.label || `Token ${token.id!.slice(0, 8)}`;
}
