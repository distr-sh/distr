import {HttpClient} from '@angular/common/http';
import {inject, Injectable, Pipe, PipeTransform} from '@angular/core';
import {SafeUrl} from '@angular/platform-browser';
import {catchError, map, Observable, of, shareReplay, throwError} from 'rxjs';

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/**
 * Fetches images that require authentication and hands out one object URL per image for the whole application.
 * A file is immutable for the lifetime of its ID, so a resolved image is never invalidated. Its object URL is
 * therefore never revoked either, which is what allows every `<img>` of the same image to share it.
 */
@Injectable({providedIn: 'root'})
export class SecureImageCache {
  private readonly httpClient = inject(HttpClient);
  private readonly images = new Map<string, Observable<SafeUrl>>();

  public get(urlOrUuid: string): Observable<SafeUrl> {
    if (uuidPattern.test(urlOrUuid)) {
      return this.fetch('/api/v1/files/' + urlOrUuid);
    } else if (urlOrUuid.startsWith('/api/')) {
      return this.fetch(urlOrUuid);
    } else {
      return of(urlOrUuid);
    }
  }

  private fetch(url: string): Observable<SafeUrl> {
    let image = this.images.get(url);
    if (!image) {
      image = this.httpClient.get(url, {responseType: 'blob'}).pipe(
        map((data) => URL.createObjectURL(data)),
        // shareReplay also replays a failure to every later subscriber, so forget it to allow a retry.
        catchError((err) => {
          this.images.delete(url);
          return throwError(() => err);
        }),
        shareReplay({bufferSize: 1, refCount: false})
      );
      this.images.set(url, image);
    }
    return image;
  }
}

@Pipe({name: 'secureImage'})
export class SecureImagePipe implements PipeTransform {
  private readonly cache = inject(SecureImageCache);

  public transform(urlOrUuid: string): Observable<SafeUrl> {
    return this.cache.get(urlOrUuid);
  }
}
