import { Pipe, PipeTransform } from '@angular/core';
import * as moment from 'moment';

@Pipe({
  name: 'relativeTime'
})
export class RelativeTimePipe implements PipeTransform {

  transform(value: number, withoutSuffix: boolean): string {
    return moment().subtract(value, 'seconds').fromNow(withoutSuffix);
  }

}
