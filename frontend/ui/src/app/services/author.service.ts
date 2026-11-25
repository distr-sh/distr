import {inject, Injectable} from '@angular/core';
import {catchError, combineLatest, map, Observable, of} from 'rxjs';
import {isNotFound, toastDisplayedError} from '../../util/errors';
import {organizationKind} from '../../util/organization-kind';
import {Actor} from '../types/actor';
import {AuthService} from './auth.service';
import {ContextService} from './context.service';
import {OrganizationBrandingService} from './organization-branding.service';
import {OrganizationService} from './organization.service';
import {ToastService} from './toast.service';

export interface Author {
  name: string;
  imageUrl?: string;
  /** Set for an organization standing in for its member, whose image is then a logo. */
  isOrganization?: boolean;
}

export interface AuthorContext {
  vendor?: Author;
  /** Stands in for the members of the partner with this id only. */
  partner?: {id: string; author: Author};
}

@Injectable({providedIn: 'root'})
export class AuthorService {
  private readonly auth = inject(AuthService);
  private readonly context = inject(ContextService);
  private readonly organizationService = inject(OrganizationService);
  private readonly organizationBranding = inject(OrganizationBrandingService);
  private readonly toast = inject(ToastService);

  /** The organizations that stand in for the users the API withholds, which only a customer or partner is denied. */
  public getContext(): Observable<AuthorContext> {
    if (this.auth.isVendor()) {
      return of({});
    }
    return combineLatest([
      this.organizationService.get(),
      this.organizationBranding.get().pipe(
        catchError((e) => {
          if (!isNotFound(e)) {
            toastDisplayedError(this.toast, e);
          }
          return of(undefined);
        })
      ),
      this.context.getPartnerOrganization(),
    ]).pipe(
      map(([org, branding, partner]) => ({
        vendor: {name: org.name, imageUrl: branding?.logoImageId, isOrganization: true},
        partner: partner && {id: partner.id, author: {name: partner.name, isOrganization: true}},
      }))
    );
  }
}

/**
 * Who did something, as far as the viewer may know. A vendor or partner member the API withholds stands in as their
 * organization: the partner if it is the viewer's own or the one the viewer's customer organization is assigned to,
 * the vendor otherwise. A deleted user stays anonymous.
 */
export function actorAuthor(actor: Actor | undefined, context: AuthorContext): Author | undefined {
  if (!actor) {
    return undefined;
  }
  if (actor.name) {
    return {name: actor.name, imageUrl: actor.imageUrl};
  }
  if (actor.deleted) {
    return undefined;
  }
  switch (organizationKind(actor)) {
    case 'vendor':
      return context.vendor;
    case 'partner':
      return actor.partnerOrganizationId === context.partner?.id ? context.partner?.author : context.vendor;
    default:
      return undefined;
  }
}
