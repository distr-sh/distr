import {Pipe, PipeTransform} from '@angular/core';
import {BaseModel} from '@distr-sh/distr-sdk';
import dayjs from 'dayjs';
import {Duration} from 'dayjs/plugin/duration';
import {isOlderThan} from './dates';

export function isStale(model: BaseModel, duration: Duration = dayjs.duration({seconds: 60})): boolean {
  return isOlderThan(model.createdAt, duration);
}

@Pipe({name: 'isStale'})
export class IsStalePipe implements PipeTransform {
  transform(value: BaseModel): boolean {
    return isStale(value);
  }
}
