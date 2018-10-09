import { Component, OnInit, ViewEncapsulation, ViewChild, ElementRef, Output, EventEmitter, OnDestroy } from '@angular/core';
import 'rxjs/add/operator/map'
import { EmojiService } from '../../providers'

declare const Buffer

@Component({
  selector: 'app-editor',
  templateUrl: './editor.component.html',
  styleUrls: ['./editor.components.scss'],
  encapsulation: ViewEncapsulation.None
})
export class EditorComponent implements OnInit, OnDestroy {
  emojiBox = false;
  @Output() onEnter = new EventEmitter<string>();
  @ViewChild('editor') editor: ElementRef;
  constructor(public emoji: EmojiService) {
  }
  ngOnInit() {
    this.editor.nativeElement.focus();
  }
  clear() {
    this.editor.nativeElement.innerHTML = '';
  }

  ngOnDestroy() {
    this.clear();
  }
  openEmoji(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    this.emojiBox = !this.emojiBox;
  }
  selectEmoji(ev: Event, data: any) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    if (data.short === '') {
      return;
    }
    const nodes: NodeList = this.editor.nativeElement.childNodes;
    console.log('nodes length:', nodes.length);
    if (nodes.length > 0) {
      for (let index = 0; index < nodes.length; index++) {
        if (nodes.item(index).nodeName.toUpperCase() === 'BR') {
          this.editor.nativeElement.removeChild(nodes.item(index));
        }
      }
    }
    this.edit(data.code);
  }

  _enter(e: Event) {
    e.preventDefault();
    e.stopImmediatePropagation();
    e.stopPropagation();
    let msg = '';
    const nodes = this.editor.nativeElement.childNodes
    nodes.forEach(el => {
      switch (el.nodeName) {
        case '#text':
          msg += el.data;
          break;
        case 'IMG':
          msg += el.title;
          break;
        default:
          break;
      }
    });
    if (Buffer.byteLength(msg, 'utf8') >= 500) {
      console.log('max length 500');
      return;
    }
    this.onEnter.emit(msg);
    this.editor.nativeElement.innerHTML = '';
  }
  close(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    this.emojiBox = false;
  }
  edit(code: string) {
    this.emojiBox = false;
    this.editor.nativeElement.focus();
    this.editor.nativeElement.innerHTML += code;
    const range = document.createRange();
    const sel = window.getSelection();
    range.selectNodeContents(this.editor.nativeElement);
    sel.removeAllRanges();
    sel.addRange(range);
    sel.collapseToEnd();
  }
}
