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
    const now = parseInt((new Date().getTime() / 1000) + '', 10);
    const ago = now - value;
    if (!isNumber) {
      let timeStr = 'unkown';
      if (ago < this.SECOND) {
        timeStr = ago + ' second ago';
      } else if (ago > this.SECOND && ago < this.HOUR) {
        timeStr = parseInt((ago / 60) + '', 10) + '  minute ago';
      } else if (ago > this.HOUR && ago < this.DAY) {
        timeStr = parseInt((ago / 60 / 60) + '', 10) + '  hour ago';
      } else if (ago > this.DAY && ago < this.MONTH) {
        timeStr = parseInt((ago / 60 / 60 / 24) + '', 10) + '  day ago';
      } else if (ago > this.MONTH) {
        timeStr = parseInt((ago / 60 / 60 / 24 / 12) + '', 10) + '  month ago';
      }
      return timeStr;
    } else {
      return ago;
    }
  }
}
