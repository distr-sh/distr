import {Pipe, PipeTransform} from '@angular/core';
import dayjs from 'dayjs';
import {Duration} from 'dayjs/plugin/duration';

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
