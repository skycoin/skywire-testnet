import {Directive, ElementRef, Input, HostListener, Output, EventEmitter, Renderer} from '@angular/core';

declare var Clipboard: any;

// tslint:disable-next-line:directive-selector
@Directive({selector: '[clipboard]'})
export class ClipboardDirective {
  @Input() clipboardText = '';
  @Output() copySuccess = new EventEmitter<boolean>();
  cliboard: any = null;

  constructor(private el: ElementRef, private render: Renderer) {
    this.cliboard = new Clipboard(this.el.nativeElement);
    this.render.setElementClass(this.el.nativeElement, 'copy-btn', true);
    this.cliboard.on('success', (e) => {
      this.copySuccess.emit(true);
      e.clearSelection();
    }, err => {
      console.error(err);
      this.copySuccess.emit(false);
    });
  }

  @HostListener('click', ['$event'])
  _click(ev: Event) {
    this.cliboard.text = () => {
      return this.clipboardText;
    };
  }
}
