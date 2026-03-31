import {inject, signal} from '@angular/core';
import {Subject, firstValueFrom, race, timer} from 'rxjs';
import {DialogRef} from '../../services/overlay.service';

export class ClosableDialog<T = unknown> {
  protected readonly dialogRef = inject(DialogRef) as DialogRef<T>;
  protected readonly isClosing = signal(false);
  protected readonly animationTimeout = 1000;
  private readonly animationComplete$ = new Subject<void>();

  constructor() {
    this.dialogRef.addOnClosedHook(async () => {
      this.isClosing.set(true);

      await firstValueFrom(race(this.animationComplete$, timer(this.animationTimeout)));
    });
  }

  protected animationComplete(): void {
    this.animationComplete$.next();
  }
}
