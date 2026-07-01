import {ChangeDetectionStrategy, Component, inject, OnDestroy, OnInit} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {distinctUntilChanged, filter, lastValueFrom, map, Subject, takeUntil} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {PlaceholderDirective} from '../directives/placeholder.directive';
import {AuthService} from '../services/auth.service';

@Component({
  selector: 'app-forgot',
  imports: [ReactiveFormsModule, RouterLink, AutotrimDirective, PlaceholderDirective],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './forgot.component.html',
})
export class ForgotComponent implements OnInit, OnDestroy {
  public readonly formGroup = new FormGroup({
    email: new FormControl('', [Validators.required, Validators.email]),
  });
  public errorMessage?: string;
  public success = false;
  loading = false;
  private readonly auth = inject(AuthService);
  private readonly route = inject(ActivatedRoute);
  private readonly destroyed$ = new Subject<void>();

  public ngOnInit(): void {
    this.route.queryParams
      .pipe(
        map((params) => params['email']),
        filter((email) => email),
        distinctUntilChanged(),
        takeUntil(this.destroyed$)
      )
      .subscribe((email) => {
        this.formGroup.patchValue({email});
      });
    this.route.queryParams
      .pipe(
        map((params) => params['reason']),
        filter((reason) => reason),
        distinctUntilChanged(),
        takeUntil(this.destroyed$)
      )
      .subscribe((reason) => {
        if (reason === 'invite-expired') {
          this.errorMessage =
            'Your invite link has expired. To finalize your account setup, ' + 'please request a password reset here. ';
        } else if (reason === 'reset-expired') {
          this.errorMessage = 'Your password reset link has expired. Please request a new link here.';
        }
      });
  }

  public ngOnDestroy(): void {
    this.destroyed$.next();
    this.destroyed$.complete();
  }

  public async submit(): Promise<void> {
    this.formGroup.markAllAsTouched();
    this.errorMessage = undefined;
    if (this.formGroup.valid) {
      this.loading = true;
      const value = this.formGroup.value;
      try {
        await lastValueFrom(this.auth.resetPassword(value.email!));
        this.success = true;
      } catch (e) {
        this.errorMessage = getFormDisplayedError(e);
      } finally {
        this.loading = false;
      }
    }
  }
}
