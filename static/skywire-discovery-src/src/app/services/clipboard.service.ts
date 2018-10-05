import { Inject } from '@angular/core';
import { Injectable } from '@angular/core';
import { DOCUMENT } from '@angular/common';

@Injectable({
  providedIn: 'root'
})
export class ClipboardService {
  private dom: Document;

  constructor(@Inject(DOCUMENT) dom: Document) {
    this.dom = dom;
  }

  copy(value: string): Promise<string> {
    return new Promise<string>(resolve => {
      let textarea = null;

      try {
        textarea = this.dom.createElement('textarea');
        textarea.style.height = '0px';
        textarea.style.left = '-100px';
        textarea.style.opacity = '0';
        textarea.style.position = 'fixed';
        textarea.style.top = '-100px';
        textarea.style.width = '0px';
        this.dom.body.appendChild(textarea);

        textarea.value = value;
        textarea.select();

        this.dom.execCommand('copy');

        resolve(value);
      } finally {
        if (textarea && textarea.parentNode) {
          textarea.parentNode.removeChild(textarea);
        }
      }
    });
  }
}
