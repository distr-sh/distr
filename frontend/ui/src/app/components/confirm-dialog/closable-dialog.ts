import {inject, signal} from '@angular/core';
import {Subject, firstValueFrom} from 'rxjs';
import {DialogRef} from '../../services/overlay.service';

export class ClosableDialog<T = unknown> {
  protected readonly dialogRef = inject(DialogRef) as DialogRef<T>;
  protected readonly isClosing = signal(false);
  private readonly animationComplete$ = new Subject<void>();

  constructor() {
    this.dialogRef.addOnClosedHook(async () => {
      this.isClosing.set(true);
      await firstValueFrom(this.animationComplete$);
    });
  }

  protected animationComplete(): void {
    this.animationComplete$.next();
  }
}
