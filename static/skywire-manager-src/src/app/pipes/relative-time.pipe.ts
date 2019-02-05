import { Pipe, PipeTransform } from '@angular/core';
import * as moment from 'moment';
import {TranslateService} from '@ngx-translate/core';

@Pipe({
  name: 'relativeTime'
})
export class RelativeTimePipe implements PipeTransform {
  constructor(
    private translate: TranslateService
  ) {}

  transform(value: number, withoutSuffix: boolean): string {
    return moment().locale(this.translate.currentLang).subtract(value, 'seconds').fromNow(withoutSuffix);
  }

}
