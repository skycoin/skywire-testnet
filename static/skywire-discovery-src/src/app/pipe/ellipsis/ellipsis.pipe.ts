import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'ellipsis'
})

export class EllipsisPipe implements PipeTransform {
  transform(value: string, ...args: any[]): any {
    let result = '';
    const position = args[0];
    const residue = args[1];
    switch (position) {
      case 'center':
        const count = residue / 2;
        const start = value.substring(0, count);
        const end = value.substring((value.length - count));
        result = start + '...' + end;
        break;
      case 'end':
        break;
    }
    if (result) {
      return result;
    }else {
      return value;
    }
  }
}
