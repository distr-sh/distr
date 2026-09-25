import {Pipe, PipeTransform} from '@angular/core';
import dayjs from 'dayjs';
import duration, {Duration} from 'dayjs/plugin/duration';
import relativeTime from 'dayjs/plugin/relativeTime';
import utc from 'dayjs/plugin/utc';

// The plugins this module calls have to be registered here rather than only during bootstrap,
// because importing them is also what declares their methods on Dayjs. Without it, a program that
// does not include main.ts, such as a unit test, does not compile.
dayjs.extend(duration);
dayjs.extend(relativeTime);
dayjs.extend(utc);

export function isOlderThan(date: dayjs.ConfigType, duration: Duration): boolean {
  return dayjs.duration(Math.abs(dayjs(date).diff(dayjs()))) > duration;
}

@Pipe({name: 'relativeDate'})
export class RelativeDatePipe implements PipeTransform {
  transform(value: dayjs.ConfigType, withoutSuffix: boolean = false): string {
    return dayjs(value).fromNow(withoutSuffix);
  }
}

export function isExpired(obj: {expiresAt?: Date | string}): boolean {
  return obj.expiresAt ? dayjs(obj.expiresAt).isBefore() : false;
}

export function isArchived(obj: {archivedAt?: Date | string}): boolean {
  return obj.archivedAt ? dayjs(obj.archivedAt).isBefore() : false;
}

export function dateTimeLocalToISO(dateTimeLocal: string | null | undefined): string | null {
  return dateTimeLocal ? dayjs(dateTimeLocal).toISOString() : null;
}

export function isoToDateTimeLocal(iso: string | null | undefined): string {
  return iso ? dayjs(iso).local().format('YYYY-MM-DDTHH:mm') : '';
}
