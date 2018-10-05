import { Directive, Output, Input, HostListener } from '@angular/core';
import { EventEmitter } from '@angular/core';
import { ClipboardService } from '../services/clipboard.service';

@Directive({
  /* tslint:disable:directive-selector */
  selector: '[clipboard]',
})
export class ClipboardDirective {
  @Output() copyEvent: EventEmitter<string>;
  @Output() errorEvent: EventEmitter<Error>;
  /* tslint:disable:no-input-rename */
  @Input('clipboard') value: string;

  constructor(private clipboardService: ClipboardService) {
    this.copyEvent = new EventEmitter();
    this.errorEvent = new EventEmitter();
    this.value = '';
  }

  @HostListener('click') copyToClipboard() {
    this.clipboardService
      .copy(this.value)
      .then((value: string) => {
        this.copyEvent.emit(value);
      })
      .catch((error: Error) => {
        this.errorEvent.emit(error);
      });
  }
}
