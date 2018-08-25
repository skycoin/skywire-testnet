import { Directive, ElementRef } from '@angular/core';

@Directive({ selector: '[appDebug]' })
export class DebugDirective {
  constructor(private el: ElementRef) {
    console.log('el:', this.el);
  }
}
