import { Pipe, PipeTransform } from '@angular/core';
import { DomSanitizer } from '@angular/platform-browser';

@Pipe({
  name: 'safe',
})
export class SafePipe implements PipeTransform {
  constructor(private sanitizer: DomSanitizer) {
  }

  transform(html: string, type: string): any {
    if (!html) {
      return '';
    }
    switch (type) {
      case 'html':
        return this.sanitizer.bypassSecurityTrustHtml(html);
      case 'url':
        console.log('html:', html);
        return this.sanitizer.bypassSecurityTrustResourceUrl(html);
    }
  }
}
