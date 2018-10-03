import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'byteTo'
})
export class ByteToPipe implements PipeTransform {
  private KB = 1024;
  private MB = this.KB * this.KB;
  private GB = this.MB * this.KB;
  private TB = this.GB * this.KB;
  transform(value: number, args?: any): any {
    if (value < this.KB) {
      return value + 'B';
    } else if (value > this.KB && value < this.MB) {
      return (value / this.KB).toFixed(2) + 'KB';
    } else if (value > this.MB && value < this.GB) {
      return (value / this.KB / this.KB).toFixed(2) + 'MB';
    } else if (value > this.GB && value < this.TB) {
      return (value / this.KB / this.KB / this.KB).toFixed(2) + 'GB';
    }
  }

}
