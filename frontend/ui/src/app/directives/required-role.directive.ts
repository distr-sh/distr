import {Directive, effect, EmbeddedViewRef, inject, input, OnInit, TemplateRef, ViewContainerRef} from '@angular/core';
import {UserRole} from '@distr-sh/distr-sdk';
import {AuthService} from '../services/auth.service';

abstract class EmbeddedViewToggler {
  private readonly templateRef = inject(TemplateRef);
  private readonly viewContainerRef = inject(ViewContainerRef);
  private embeddedViewRef: EmbeddedViewRef<unknown> | null = null;

  protected showEmbeddedView() {
    if (this.embeddedViewRef === null) {
      this.embeddedViewRef = this.viewContainerRef.createEmbeddedView(this.templateRef);
    }
  }

  protected hideEmbeddedView() {
    if (this.embeddedViewRef !== null) {
      this.embeddedViewRef.destroy();
      this.embeddedViewRef = null;
    }
  }

  protected toggleEmbeddedView(value: boolean) {
    if (value) {
      this.showEmbeddedView();
    } else {
      this.hideEmbeddedView();
    }
  }
}

@Directive({selector: '[appRequiredRole]'})
export class RequireRoleDirective extends EmbeddedViewToggler {
  public readonly role = input.required<UserRole | UserRole[]>({alias: 'appRequiredRole'});

  private readonly auth = inject(AuthService);

  constructor() {
    super();
    effect(() => this.toggleEmbeddedView(this.auth.isSuperAdmin() || this.auth.hasAnyRole(...[this.role()].flat())));
  }
}

@Directive({selector: '[appRequireVendor]'})
export class RequireVendorDirective extends EmbeddedViewToggler implements OnInit {
  private readonly auth = inject(AuthService);

  public ngOnInit(): void {
    this.toggleEmbeddedView(this.auth.isVendor());
  }
}

@Directive({selector: '[appRequireCustomer]'})
export class RequireCustomerDirective extends EmbeddedViewToggler implements OnInit {
  private readonly auth = inject(AuthService);

  public ngOnInit(): void {
    this.toggleEmbeddedView(this.auth.isCustomer());
  }
}

export interface PermissionsInput {
  customer?: boolean;
  vendor?: boolean;
  role?: UserRole | UserRole[];
}

@Directive({selector: '[appPermissions]'})
export class PermissionsDirective extends EmbeddedViewToggler {
  public readonly permissions = input.required<PermissionsInput>({alias: 'appPermissions'});

  private readonly auth = inject(AuthService);

  constructor() {
    super();
    effect(() => this.toggleEmbeddedView(this.isSatisfied(this.permissions())));
  }

  private isSatisfied(permissions: PermissionsInput): boolean {
    if (permissions.customer && !this.auth.isCustomer()) {
      return false;
    }

    if (permissions.vendor && !this.auth.isVendor()) {
      return false;
    }

    if (permissions.role && !this.auth.hasAnyRole(...[permissions.role].flat())) {
      return false;
    }

    return true;
  }
}
