import { Pipe, PipeTransform } from '@angular/core';

/**
 * Iterable Pipe
 *
 * It accepts Objects and [Maps](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Map)
 *
 * Example:
 *
 *  <div *ngFor="let keyValuePair of someObject | iterable">
 *    key {{keyValuePair.key}} and value {{keyValuePair.value}}
 *  </div>
 *
 */
@Pipe({ name: 'iterable', pure: false })
export class IterablePipe implements PipeTransform {
  transform(iterable: any, args: any[]): any {
    const result = [];
    if (!iterable) {
      return result;
    }
    if (iterable.entries) {
      iterable.forEach((key, value) => {
        result.push({ key, value });
      });
    } else {
      for (const key in iterable) {
        if (iterable.hasOwnProperty(key)) {
          result.push({ key, value: iterable[key] });
        }
      }
    }

    return result;
  }
}
