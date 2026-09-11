import {Injectable} from '@angular/core';
import {AccessTokenWithKey} from '@distr-sh/distr-sdk';

// Hands a freshly created token to the page that displays it. The navigation state would do the
// same, but it lives in the browser history entry, where a back navigation restores it and any
// script can read it, which a token that is shown exactly once must not survive.
@Injectable({providedIn: 'root'})
export class CreatedAccessTokenStore {
  private created: AccessTokenWithKey | null = null;

  public put(token: AccessTokenWithKey) {
    this.created = token;
  }

  public take(id: string): AccessTokenWithKey | null {
    const created = this.created?.id === id ? this.created : null;
    this.created = null;
    return created;
  }
}
