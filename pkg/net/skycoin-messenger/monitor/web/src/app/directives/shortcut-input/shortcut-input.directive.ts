import { Directive, HostBinding, Input, ElementRef, OnInit, HostListener, Output, EventEmitter } from '@angular/core';

// tslint:disable-next-line:directive-selector
@Directive({ selector: '[shortcut-input]' })
export class ShortcutInputDirective implements OnInit {
  @Input() text = '';
  @HostBinding('class') classes = 'shortcut_input';
  @Output() onEdit = new EventEmitter<number>();
  isEdit = false;
  constructor(private el: ElementRef) { }
  ngOnInit() {
    if (this.text) {
      this.el.nativeElement.value = this.text;
    } else {
      this.el.nativeElement.value = '';
    }
  }

  @HostListener('click', ['$event'])
  _click(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
  }

  @HostListener('focus', ['$event'])
  _foucs(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    this.isEdit = true;
  }
  @HostListener('blur', ['$event'])
  _blur(ev: Event) {
    if (this.isEdit) {
      this.isEdit = false;
      this.onEdit.emit(this.el.nativeElement.value);
    }
  }
}
