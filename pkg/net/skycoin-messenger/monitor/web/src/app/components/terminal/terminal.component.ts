import { Component, OnInit, ViewChild, ElementRef, OnDestroy, HostListener, Renderer } from '@angular/core';
import { ApiService } from '../../service/api/api.service';
import { AlertService } from '../../service/alert/alert.service';
import { Observable } from 'rxjs/Observable';
import { Subscription } from 'rxjs/Subscription';
import { MatDialogRef } from '@angular/material';
import * as Terminal from 'xterm';
import 'rxjs/add/observable/interval';

@Component({
  selector: 'app-terminal',
  templateUrl: 'terminal.component.html',
  styleUrls: ['./terminal.component.scss']
})

export class TerminalComponent implements OnInit, OnDestroy {
  @ViewChild('edit') edit: ElementRef;
  @ViewChild('container') conainer: ElementRef;
  @ViewChild('terminal') terminal: ElementRef;
  index = -1;
  addr = '';
  task: Subscription = null;
  xterm = null;
  ws: WebSocket = null;
  url = 'ws://127.0.0.1:8000/term';
  isWrite = true;
  x = 0;
  y = 0;
  pageX = 0;
  pageY = 0;
  startWidth = 0;
  startHeight = 0;
  isMove = false;
  canResize = false;
  isResize = false;
  rows = 0;
  cols = 0;
  constructor(
    private api: ApiService,
    private dialogRef: MatDialogRef<TerminalComponent>,
    private el: ElementRef,
    private render: Renderer,
    private alert: AlertService,
  ) { }
  ngOnInit() {
    this.setUrl();
    this.api.checkLogin().subscribe(id => {
      if (!this.url.length) {
        this.alert.error('Temporarily unable to connect, please try again later.');
      } else {
        this.ws = new WebSocket(`${this.url}?url=ws://${this.addr}/node/run/term&token=${id}`);
        this.ws.binaryType = 'arraybuffer';
        this.ws.onopen = (ev) => {
          this.start();
        };
        this.ws.onclose = (ev: CloseEvent) => {
          this.isWrite = false;
          console.log('onclose:', ev);
          if (this.xterm) {
            this.xterm.writeln('The Connection interrupted... Close the terminal automatically after 3 seconds.');
            setTimeout(() => {
              this._close();
            }, 3000);
          }
        };
        this.ws.onerror = (ev: Event) => {
          this.isWrite = false;
          console.log('onerror:', ev);
          if (this.xterm) {
            this.xterm.writeln('The Connection interrupted... Close the terminal automatically after 3 seconds.');
            setTimeout(() => {
              this._close();
            }, 3000);
          }
        };
      }
    });
  }
  setUrl() {
    this.api.getManagerPort().subscribe(port => {
      const localhost = 'localhost';
      this.url = window.location.host;
      this.url = this.url.replace(localhost, '127.0.0.1');
      const tmp = this.url.split(':');
      if (tmp.length === 2) {
        this.url = tmp[0];
      }
      this.url = `ws://${this.url}:${port}/term`;
    }, err => {
      console.error('get port error:', err);
      this.url = '';
    });
  }
  ngOnDestroy() {
    this.ws.close();
  }
  resize() {
    if (!this.xterm) {
      return;
    }
    const geometry = this.xterm.proposeGeometry();
    this.rows = geometry.rows;
    this.cols = geometry.cols;
    this.xterm.resize(geometry.cols, geometry.rows);
    this.render.setElementStyle(this.conainer.nativeElement, 'height', this.conainer.nativeElement.childNodes[1].clientHeight + 'px');
  }
  send(data) {
    if (this.isWrite) {
      this.ws.send(this.stringToUint8(data));
    }
  }
  start() {
    // Terminal.loadAddon('fullscreen');
    Terminal.loadAddon('fit');
    this.xterm = new Terminal({
      cursorBlink: true
    });
    this.xterm.open(this.conainer.nativeElement, true);
    // this.xterm.fit();
    const geometry = this.xterm.proposeGeometry();
    this.rows = geometry.rows;
    this.cols = geometry.cols;
    this.ws.onmessage = (evt) => {
      if (evt.data instanceof ArrayBuffer) {
        this.xterm.write(this.utf8ArrayToStr(new Uint8Array(evt.data)));
      } else {
        console.log('ws data:', evt.data);
      }
    };
    this.xterm.on('data', (data) => {
      this.send('\x00' + data);
    });
    this.xterm.on('resize', (ev) => {
      this.send('\x01' + JSON.stringify({ cols: ev.cols, rows: ev.rows, }));
    });
  }
  _close(ev?: Event) {
    if (ev) {
      ev.stopImmediatePropagation();
      ev.stopPropagation();
      ev.preventDefault();
    }
    this.dialogRef.close();
  }
  utf8ArrayToStr(array) {
    let out, i, len, c;
    let char2, char3;

    out = '';
    len = array.length;
    i = 0;
    while (i < len) {
      c = array[i++];
      // tslint:disable-next-line:no-bitwise
      switch (c >> 4) {
        case 0:
        case 1:
        case 2:
        case 3:
        case 4:
        case 5:
        case 6:
        case 7:
          // 0xxxxxxx
          out += String.fromCharCode(c);
          break;
        case 12:
        case 13:
          // 110x xxxx   10xx xxxx
          char2 = array[i++];
          // tslint:disable-next-line:no-bitwise
          out += String.fromCharCode(((c & 0x1F) << 6) | (char2 & 0x3F));
          break;
        case 14:
          // 1110 xxxx  10xx xxxx  10xx xxxx
          char2 = array[i++];
          char3 = array[i++];
          // tslint:disable-next-line:no-bitwise
          out += String.fromCharCode(((c & 0x0F) << 12) |
            // tslint:disable-next-line:no-bitwise
            ((char2 & 0x3F) << 6) |
            // tslint:disable-next-line:no-bitwise
            ((char3 & 0x3F) << 0));
          break;
      }
    }

    return out;
  }
  stringToUint8(str: string): Uint8Array {
    const bytes = new Array();
    let len, c;
    len = str.length;
    for (let i = 0; i < len; i++) {
      c = str.charCodeAt(i);
      if (c >= 0x010000 && c <= 0x10FFFF) {
        bytes.push(((c >> 18) & 0x07) | 0xF0);
        bytes.push(((c >> 12) & 0x3F) | 0x80);
        bytes.push(((c >> 6) & 0x3F) | 0x80);
        bytes.push((c & 0x3F) | 0x80);
      } else if (c >= 0x000800 && c <= 0x00FFFF) {
        bytes.push(((c >> 12) & 0x0F) | 0xE0);
        bytes.push(((c >> 6) & 0x3F) | 0x80);
        bytes.push((c & 0x3F) | 0x80);
      } else if (c >= 0x000080 && c <= 0x0007FF) {
        bytes.push(((c >> 6) & 0x1F) | 0xC0);
        bytes.push((c & 0x3F) | 0x80);
      } else {
        bytes.push(c & 0xFF);
      }
    }
    return new Uint8Array(bytes);
  }
  _move(ev) {
    // ev.stopImmediatePropagation();
    // ev.stopPropagation();
    // ev.preventDefault();
    // top width > 10;
    const width_gap = 10;

    const y = ev.pageY - parseInt((<string>this.terminal.nativeElement.style.top).
      substring(0, this.terminal.nativeElement.style.top.length - 2), 10);
    const x = ev.pageX - parseInt((<string>this.terminal.nativeElement.style.left).
      substring(0, this.terminal.nativeElement.style.left.length - 2), 10);
    const width = this.terminal.nativeElement.clientWidth - width_gap;
    const height = this.terminal.nativeElement.clientHeight - 10;
    if (y < 2 && x > 10 && x < width) {
      console.log('top');
      this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'ns-resize');
    } else if (y > height && x > width_gap && x < width) {
      console.log('bottom');
      this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'ns-resize');
    } else if (x < 12 && y > 10 && y < height) {
      console.log('left');
      this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'ew-resize');
    } else if (x > width && y > 10 && y < height) {
      console.log('right');
      this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'ew-resize');
    } else if (x < 10 && y < 10) {
      console.log('top left');
      this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'nwse-resize');
    } else if (x > width && y < 10) {
      console.log('top right');
      this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'nesw-resize');
    } else if (x < width_gap && y > height) {
      console.log('bottom left');
      this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'nesw-resize');
    } else if (x > width && y > height) {
      console.log('bottom right');
      this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'nesw-resize');
    } else {
      this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'auto');
    }
    // if (x > width && y > height) {
    //   // bottom right
    //   this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'nwse-resize');
    // } else if (x <= 10 && y > height) {
    //   // bottom left
    //   this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'nesw-resize');
    // } else if (x > width && y < height && y > 20) {
    //   // right
    //   this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'ew-resize');
    // } else if (x <= 5 && y < height && y > 20) {
    //   // left
    //   this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'ew-resize');
    // } else if ((x > 0 && x < width) && y > height) {
    //   // bottom
    //   this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'ns-resize');
    // } else if ((x > 0 && x < width) && y < 10) {
    //   // top
    //   this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'ns-resize');
    // } else if (x > width && y < 20) {
    //   // top right
    //   this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'nesw-resize');
    // } else if (x < 20 && y < 20) {
    //   // top left
    //   this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'nwse-resize');
    // } else {
    //   this.render.setElementStyle(this.terminal.nativeElement, 'cursor', 'auto');
    // }
  }
  _mousedown(ev) {
    this.x = ev.offsetX;
    this.y = ev.offsetY;
    this.pageX = ev.clientX;
    this.pageY = ev.clientY;
    this.startWidth = parseInt(document.defaultView.getComputedStyle(this.terminal.nativeElement).width, 10);
    this.startHeight = parseInt(document.defaultView.getComputedStyle(this.terminal.nativeElement).height, 10);
    if (ev.target.className === 'terminal-header') {
      this.isMove = true;
    }
    // if (this.canResize) {
    // this.isResize = true;
    // }
  }

  @HostListener('document:mouseup', ['$event'])
  _mouseup(ev) {
    this.isMove = false;
    this.isResize = false;
  }

  @HostListener('document:mousemove', ['$event'])
  _mousemove(ev) {
    if (this.isMove) {
      this.render.setElementStyle(this.terminal.nativeElement, 'top', (<number>ev.clientY - this.y) + 'px');
      this.render.setElementStyle(this.terminal.nativeElement, 'left', (<number>ev.clientX - this.x) + 'px');
    } else if (this.isResize) {
      this.render.setElementStyle(this.terminal.nativeElement, 'width', (this.startWidth + ev.clientX - this.pageX) + 'px');
      this.render.setElementStyle(this.conainer.nativeElement, 'height', (this.startHeight + ev.clientY - this.pageY) + 'px');
      this.resize();
    }
  }
}
