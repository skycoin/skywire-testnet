import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'timeAgo'
})
export class TimeAgoPipe implements PipeTransform {
  private SECOND = 60;
  private HOUR = this.SECOND * 60; // 1 hour
  private DAY = this.HOUR * 24;
  private MONTH = this.DAY * 31;
  transform(value: number, isNumber?: boolean): string | number {
    // const now = parseInt((new Date().getTime() / 1000) + '', 10);
    // const ago = now - value;
    if (!isNumber) {
      let timeStr = '0 second ago';
      if (value < this.SECOND) {
        timeStr = value + ' second ago';
      } else if (value > this.SECOND && value < this.HOUR) {
        timeStr = parseInt((value / 60) + '', 10) + '  minute ago';
      } else if (value > this.HOUR && value < this.DAY) {
        timeStr = parseInt((value / 60 / 60) + '', 10) + '  hour ago';
      } else if (value > this.DAY && value < this.MONTH) {
        timeStr = parseInt((value / 60 / 60 / 24) + '', 10) + '  day ago';
      } else if (value > this.MONTH) {
        timeStr = parseInt((value / 60 / 60 / 24 / 12) + '', 10) + '  month ago';
      }
      return timeStr;
    } else {
      return value;
    }
  }
}
