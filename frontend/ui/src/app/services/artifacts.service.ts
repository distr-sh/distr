import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {map, Observable, of, switchMap, tap} from 'rxjs';
import {ReactiveList} from './cache';

export interface HasDownloads {
  downloadsTotal?: number;
  downloadedByUsersCount?: number;
  downloadedByUsers?: string[];
  downloadedByCustomerOrganizationsCount?: number;
  downloadedByCustomerOrganizations?: string[];
}

export interface ArtifactUser {
  id: string;
  avatarUrl: string;
}

export interface VulnerabilitySeverity {
  type: 'CVSS_V2' | 'CVSS_V3' | 'CVSS_V4';
  score: string;
}

/**
 * From https://ossf.github.io/osv-schema/
 *
 * Severity calculator: https://www.first.org/cvss/calculator/4.0
 */
export interface Vulnerability {
  id: string;
  severity: VulnerabilitySeverity[];
}

export interface BaseArtifact {
  id: string;
  name: string;
  imageUrl?: string;
}

export interface BaseArtifactVersion {
  id: string;
  name: string;
}

export interface Artifact extends BaseArtifact, HasDownloads {}

export interface TaggedArtifactVersion extends HasDownloads {
  id: string;
  digest: string;
  sbom?: string;
  createdAt: string;
  size: number;
  tags: {name: string; downloads: HasDownloads}[];
  vulnerabilities: Vulnerability[];
  lastScannedAt?: string;
  imageUrl?: string;
  inferredType: 'generic' | 'container-image' | 'helm-chart';
}

export interface ArtifactWithTags extends Artifact {
  versions?: TaggedArtifactVersion[];
}

class ArtifactsReactiveList extends ReactiveList<ArtifactWithTags> {
  protected override identify = (a: ArtifactWithTags) => a.id;
  protected override sortAttr = (a: ArtifactWithTags) => a.versions?.[0]?.createdAt ?? '';
  protected override sortInverted = true;
}

@Injectable({providedIn: 'root'})
export class ArtifactsService {
  private readonly artifactsUrl = '/api/v1/artifacts';
  private readonly http = inject(HttpClient);
  private readonly cache = new ArtifactsReactiveList(this.http.get<ArtifactWithTags[]>(this.artifactsUrl));

  public list(): Observable<ArtifactWithTags[]> {
    return this.cache.get();
  }

  public getByIdAndCache(id: string): Observable<ArtifactWithTags | undefined> {
    return this.list().pipe(
      map((ls) => ls.find((a) => a.id === id)),
      switchMap((existing) => {
        if ((existing?.versions ?? []).length > 0) {
          return of(existing);
        } else if (existing) {
          return this.http.get<ArtifactWithTags>(`${this.artifactsUrl}/${id}`).pipe(tap((a) => this.cache.save(a)));
        } else {
          return of(undefined);
        }
      })
    );
  }

  public patchImage(artifactsId: string, imageId: string) {
    return this.http
      .patch<ArtifactWithTags>(`${this.artifactsUrl}/${artifactsId}/image`, {imageId})
      .pipe(tap((it) => this.cache.save(it)));
  }

  public deleteArtifact(artifactId: string): Observable<void> {
    return this.http.delete<void>(`${this.artifactsUrl}/${artifactId}`).pipe(
      tap(() => {
        this.cache.remove({id: artifactId} as ArtifactWithTags);
      })
    );
  }

  public deleteArtifactTag(artifact: ArtifactWithTags, tagName: string) {
    return this.http.delete<void>(`${this.artifactsUrl}/${artifact.id}/tags/${encodeURIComponent(tagName)}`).pipe(
      tap(() => {
        artifact.versions = (artifact.versions ?? []).map((version) => {
          version.tags = version.tags.filter((tag) => tag.name !== tagName);
          return version;
        });
        this.cache.save(artifact);
      })
    );
  }
}
