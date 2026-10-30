import {NgPlural, NgPluralCase, NgTemplateOutlet} from '@angular/common';
import {Component, computed, input, signal} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronRight, faFolder, faFolderOpen} from '@fortawesome/free-solid-svg-icons';
import {SupportBundleResourceFolder} from './resource-tree';
import {SupportBundleResourceComponent} from './support-bundle-resource.component';

@Component({
  selector: 'app-support-bundle-resource-folder',
  templateUrl: './support-bundle-resource-folder.component.html',
  imports: [FaIconComponent, NgPlural, NgPluralCase, NgTemplateOutlet, SupportBundleResourceComponent],
  host: {class: 'block'},
})
export class SupportBundleResourceFolderComponent {
  public readonly bundleId = input.required<string>();
  public readonly folder = input.required<SupportBundleResourceFolder>();
  public readonly depth = input(0);

  protected readonly faChevronRight = faChevronRight;
  protected readonly faFolder = faFolder;
  protected readonly faFolderOpen = faFolderOpen;

  protected readonly open = signal(false);
  // The children stay rendered once the folder was opened, so that closing it keeps the content the
  // files loaded and which of them are open.
  protected readonly opened = signal(false);
  protected readonly root = computed(() => this.folder().path === '');
  protected readonly childDepth = computed(() => (this.root() ? 0 : this.depth() + 1));

  protected toggled(open: boolean): void {
    this.open.set(open);
    if (open) {
      this.opened.set(true);
    }
  }
}
