import {OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe} from '@angular/common';
import {Component, inject, input} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {CustomerOrganization} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faBox,
  faBuildingUser,
  faCircleExclamation,
  faHeartPulse,
  faPen,
  faShip,
  faTrash,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {SemVer} from 'semver';
import {maxBy} from '../../../util/arrays';
import {SecureImagePipe} from '../../../util/secureImage';
import {drawerFlyInOut} from '../../animations/drawer';
import {dropdownAnimation} from '../../animations/dropdown';
import {modalFlyInOut} from '../../animations/modal';
import {TaggedArtifactVersion} from '../../services/artifacts.service';
import {AuthService} from '../../services/auth.service';
import {DashboardArtifact} from '../../services/dashboard.service';

@Component({
  selector: 'app-artifacts-by-customer-card',
  templateUrl: './artifacts-by-customer-card.component.html',
  imports: [FaIconComponent, OverlayModule, ReactiveFormsModule, AsyncPipe, SecureImagePipe, RouterLink],
  animations: [modalFlyInOut, drawerFlyInOut, dropdownAnimation],
})
export class ArtifactsByCustomerCardComponent {
  protected readonly auth = inject(AuthService);

  public readonly customer = input.required<CustomerOrganization>();
  public readonly artifacts = input.required<DashboardArtifact[]>();

  protected isOnLatest(artifact: DashboardArtifact): boolean {
    const max = this.findMaxVersion(artifact.artifact.versions ?? []);
    const includesPulled = max?.tags.map((t) => t.name).includes(artifact.latestPulledVersion) ?? false;
    if (includesPulled) {
      return true;
    }
    const pulledStillAvailable = (artifact.artifact.versions ?? []).some((v) =>
      v.tags.some((t) => t.name === artifact.latestPulledVersion)
    );
    return !pulledStillAvailable;
  }

  private findMaxVersion(versions: TaggedArtifactVersion[]): TaggedArtifactVersion | undefined {
    try {
      return maxBy(
        versions,
        (version) => this.getMaxSemverTag(version),
        (a, b) => a.compare(b) > 0
      );
    } catch (e) {
      return maxBy(versions, (version) => new Date(version.createdAt!));
    }
  }

  private getMaxSemverTag(version: TaggedArtifactVersion) {
    const maxSemver = maxBy(
      version.tags,
      (tag) => new SemVer(tag.name),
      (a, b) => a.compare(b) > 0
    );
    if (!maxSemver) {
      throw 'no semver tag';
    }
    return new SemVer(maxSemver.name);
  }

  protected readonly faShip = faShip;
  protected readonly faPen = faPen;
  protected readonly faTrash = faTrash;
  protected readonly faHeartPulse = faHeartPulse;
  protected readonly faXmark = faXmark;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faBox = faBox;
  protected readonly faBuildingUser = faBuildingUser;
}
