import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, inject, signal, TemplateRef} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPlus, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, startWith, Subject, switchMap} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {drawerFlyInOut} from '../../animations/drawer';
import {ClipComponent} from '../../components/clip.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';

@Component({
  selector: 'app-customer-support-bundles',
  templateUrl: './customer-support-bundles.component.html',
  imports: [AsyncPipe, DatePipe, ReactiveFormsModule, RouterLink, FaIconComponent, ClipComponent, AutotrimDirective],
  animations: [drawerFlyInOut],
})
export class CustomerSupportBundlesComponent {
  private readonly supportBundlesService = inject(SupportBundlesService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);

  protected readonly faPlus = faPlus;
  protected readonly faXmark = faXmark;

  private readonly refresh$ = new Subject<void>();
  protected readonly bundles$ = this.refresh$.pipe(
    startWith(0),
    switchMap(() => this.supportBundlesService.list()),
    takeUntilDestroyed()
  );

  protected readonly createForm = new FormGroup({
    title: new FormControl('', {nonNullable: true}),
    description: new FormControl('', {nonNullable: true}),
  });

  protected createFormLoading = false;
  protected collectCommand = signal<string | null>(null);

  private drawerRef: DialogRef | null = null;

  openDrawer(templateRef: TemplateRef<unknown>) {
    this.hideDrawer();
    this.createForm.reset();
    this.collectCommand.set(null);
    this.drawerRef = this.overlay.showDrawer(templateRef);
  }

  hideDrawer() {
    this.drawerRef?.dismiss();
    this.drawerRef = null;
  }

  async createBundle() {
    this.createFormLoading = true;
    try {
      const request: {title?: string; description?: string} = {};
      if (this.createForm.value.title) {
        request.title = this.createForm.value.title;
      }
      if (this.createForm.value.description) {
        request.description = this.createForm.value.description;
      }
      const response = await firstValueFrom(this.supportBundlesService.create(request));
      this.collectCommand.set(response.collectCommand);
      this.toast.success('Support bundle created');
      this.refresh$.next();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.createFormLoading = false;
    }
  }
}
