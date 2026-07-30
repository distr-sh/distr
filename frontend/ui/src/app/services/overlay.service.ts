import {BlockScrollStrategy, GlobalPositionStrategy, Overlay, OverlayConfig, ViewportRuler} from '@angular/cdk/overlay';
import {ComponentPortal, ComponentType, TemplatePortal} from '@angular/cdk/portal';
import {
  inject,
  Injectable,
  InjectionToken,
  Injector,
  StaticProvider,
  TemplateRef,
  ViewContainerRef,
} from '@angular/core';
import {filter, fromEvent, map, merge, Observable, Subject, take, takeUntil} from 'rxjs';
import {ConfirmConfig, ConfirmDialogComponent} from '../components/confirm-dialog/confirm-dialog.component';

type OnClosedHook<T> = (result: T | null) => Promise<void> | void;

export class DialogRef<T = void> {
  private readonly result$ = new Subject<T>();
  private readonly dismissed$ = new Subject<null>();

  public onClosed: OnClosedHook<T>[] = [];

  public addOnClosedHook(hook: OnClosedHook<T>) {
    this.onClosed.push(hook);
  }

  public async close(data: T) {
    await this.waitForOnClosedHooks(data);
    this.result$.next(data);
  }

  public async dismiss() {
    await this.waitForOnClosedHooks(null);
    this.dismissed$.next(null);
  }

  public closed(): Observable<void> {
    return this.result().pipe(map(() => undefined));
  }

  public result(): Observable<T | null> {
    return merge(this.result$, this.dismissed$).pipe(take(1));
  }

  private async waitForOnClosedHooks(data: T | null) {
    await Promise.all(this.onClosed.map((it) => it(data)));
  }
}

export class ExtendedOverlayConfig extends OverlayConfig {
  backdropStyleOnly?: boolean;
  data?: unknown;
}

export const OverlayData = new InjectionToken<unknown>('OVERLAY_DATA');

@Injectable()
export class OverlayService {
  private readonly overlay = inject(Overlay);
  private readonly viewportRuler = inject(ViewportRuler);
  private readonly viewContainerRef = inject(ViewContainerRef);

  public confirm(messageOrConfig: ConfirmConfig | string) {
    const config = typeof messageOrConfig === 'string' ? {message: {message: messageOrConfig}} : messageOrConfig;
    return this.showModal<boolean>(ConfirmDialogComponent, {data: config}).result();
  }

  /**
   * @param templateRef the template to show
   * @param viewContainerRef needed to create a TemplatePortal. you can get it by injecting `ViewContainerRef`
   * @param config optional overlay config
   * @returns a handle of the modal with some control functions
   */
  public showModal<T = void>(
    templateRefOrComponentType: TemplateRef<unknown> | ComponentType<unknown>,
    config?: ExtendedOverlayConfig
  ): DialogRef<T> {
    return this.show(templateRefOrComponentType, {
      hasBackdrop: true,
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().top(),
      ...config,
    });
  }

  public showDrawer<T = void>(templateRefOrComponentType: TemplateRef<unknown> | ComponentType<unknown>): DialogRef<T> {
    return this.show(templateRefOrComponentType, {
      hasBackdrop: true,
      positionStrategy: new GlobalPositionStrategy().end().centerVertically(),
    });
  }

  private show<T>(
    templateRefOrComponentType: TemplateRef<unknown> | ComponentType<unknown>,
    config: ExtendedOverlayConfig
  ): DialogRef<T> {
    const overlayRef = this.overlay.create({
      scrollStrategy: new BlockScrollStrategy(this.viewportRuler, document),
      ...config,
    });
    const dialogRef = new DialogRef<T>();
    // DialogRef is always available, so that a dialog can close itself whether or not it was
    // handed any data. OverlayData is only provided when there is something to inject.
    const providers: StaticProvider[] = [{provide: DialogRef, useValue: dialogRef}];
    if (config.data !== null && config.data !== undefined) {
      providers.push({provide: OverlayData, useValue: config.data});
    }
    const injector = Injector.create({parent: this.viewContainerRef.injector, providers});

    if (templateRefOrComponentType instanceof TemplateRef) {
      const embeddedViewRef = overlayRef.attach(
        new TemplatePortal(templateRefOrComponentType, this.viewContainerRef, undefined, injector)
      );
      dialogRef.closed().subscribe(() => embeddedViewRef.destroy());
    } else if (!templateRefOrComponentType) {
      throw new Error('templateRefOrComponentType is ' + templateRefOrComponentType);
    } else {
      const componentRef = overlayRef.attach(new ComponentPortal(templateRefOrComponentType, null, injector));
      dialogRef.closed().subscribe(() => componentRef.destroy());
    }

    if (!config.backdropStyleOnly) {
      merge(
        overlayRef.backdropClick(),
        fromEvent<KeyboardEvent>(window, 'keydown').pipe(filter((it) => it.key === 'Escape'))
      )
        .pipe(takeUntil(dialogRef.closed()))
        .subscribe(() => dialogRef.dismiss());
    }

    return dialogRef;
  }
}
