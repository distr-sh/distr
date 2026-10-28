import {HttpClient, HttpErrorResponse} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {
  Application,
  ApplicationVersion,
  ApplicationVersionResource,
  PatchApplicationRequest,
} from '@distr-sh/distr-sdk';
import {catchError, map, Observable, of, switchMap, tap, throwError} from 'rxjs';
import {DefaultReactiveList} from './cache';
import {CrudService} from './interfaces';

@Injectable({
  providedIn: 'root',
})
export class ApplicationsService implements CrudService<Application> {
  private readonly httpClient = inject(HttpClient);

  private readonly applicationsUrl = '/api/v1/applications';
  private readonly cache = new DefaultReactiveList(() => this.httpClient.get<Application[]>(this.applicationsUrl));

  list(): Observable<Application[]> {
    return this.cache.get();
  }

  refresh(): Promise<Application[]> {
    return this.cache.refresh();
  }

  get(applicationId: string): Observable<Application> {
    return this.httpClient
      .get<Application>(`${this.applicationsUrl}/${applicationId}`)
      .pipe(tap((it) => this.cache.save(it)));
  }

  create(application: Application): Observable<Application> {
    return this.httpClient.post<Application>(this.applicationsUrl, application).pipe(tap((it) => this.cache.save(it)));
  }

  update(application: Application): Observable<Application> {
    return this.httpClient
      .put<Application>(`${this.applicationsUrl}/${application.id}`, application)
      .pipe(tap((it) => this.cache.save(it)));
  }

  patch(applicationId: string, patch: PatchApplicationRequest): Observable<Application> {
    return this.httpClient
      .patch<Application>(`${this.applicationsUrl}/${applicationId}`, patch)
      .pipe(tap((it) => this.cache.save(it)));
  }

  delete(application: Application): Observable<void> {
    return this.httpClient
      .delete<void>(`${this.applicationsUrl}/${application.id}`)
      .pipe(tap(() => this.cache.remove(application)));
  }

  getTemplateFile(applicationId: string, versionId: string): Observable<string | null> {
    return this.getFile(applicationId, versionId, 'template-file');
  }

  getValuesFile(applicationId: string, versionId: string): Observable<string | null> {
    return this.getFile(applicationId, versionId, 'values-file');
  }

  getComposeFile(applicationId: string, versionId: string): Observable<string | null> {
    return this.getFile(applicationId, versionId, 'compose-file');
  }

  getResources(applicationId: string, versionId: string): Observable<ApplicationVersionResource[]> {
    return this.httpClient.get<ApplicationVersionResource[]>(
      `${this.applicationsUrl}/${applicationId}/versions/${versionId}/resources`
    );
  }

  private getFile(applicationId: string, versionId: string, file: string): Observable<string | null> {
    return this.httpClient
      .get(`${this.applicationsUrl}/${applicationId}/versions/${versionId}/${file}`, {responseType: 'text'})
      .pipe(
        catchError((e) => {
          if (e instanceof HttpErrorResponse && e.status == 404) {
            return of(null);
          } else {
            return throwError(() => e);
          }
        })
      );
  }

  createApplicationVersionForDocker(
    application: Application,
    applicationVersion: ApplicationVersion,
    compose: string,
    template?: string | null
  ): Observable<ApplicationVersion> {
    const formData = new FormData();
    formData.append('composefile', new Blob([compose], {type: 'application/yaml'}));
    if (template) {
      formData.append('templatefile', new Blob([template], {type: 'application/yaml'}));
    }

    return this.doCreateVersion(application, applicationVersion, formData);
  }

  createApplicationVersionForKubernetes(
    application: Application,
    applicationVersion: ApplicationVersion,
    baseValues?: string | null,
    template?: string | null
  ): Observable<ApplicationVersion> {
    const formData = new FormData();
    if (baseValues) {
      formData.append('valuesfile', new Blob([baseValues], {type: 'application/yaml'}));
    }
    if (template) {
      formData.append('templatefile', new Blob([template], {type: 'application/yaml'}));
    }
    return this.doCreateVersion(application, applicationVersion, formData);
  }

  private doCreateVersion(application: Application, applicationVersion: ApplicationVersion, formData: FormData) {
    formData.append('applicationversion', JSON.stringify(applicationVersion));
    return this.httpClient
      .post<ApplicationVersion>(`${this.applicationsUrl}/${application.id}/versions`, formData)
      .pipe(switchMap((it) => this.get(application.id!).pipe(map(() => it))));
  }

  updateApplicationVersion(app: Application, version: ApplicationVersion): Observable<ApplicationVersion> {
    return this.httpClient
      .put<ApplicationVersion>(`${this.applicationsUrl}/${app.id}/versions/${version.id}`, version)
      .pipe(switchMap((it) => this.get(app.id!).pipe(map(() => it))));
  }

  public patchImage(artifactsId: string, imageId: string) {
    return this.httpClient
      .patch<Application>(`${this.applicationsUrl}/${artifactsId}/image`, {imageId})
      .pipe(tap((it) => this.cache.save(it)));
  }
}
