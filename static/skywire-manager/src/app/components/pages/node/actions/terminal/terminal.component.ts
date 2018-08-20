import { Component, ElementRef, Inject, OnInit, ViewChild } from '@angular/core';
import { NodeService } from '../../../../../services/node.service';
import { forkJoin } from 'rxjs';
import { AuthService } from '../../../../../services/auth.service';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material';
import { Terminal } from 'xterm';
import { fit } from 'xterm/lib/addons/fit/fit';
import { TranslateService } from '@ngx-translate/core';

@Component({
  selector: 'app-terminal',
  templateUrl: './terminal.component.html',
  styleUrls: ['./terminal.component.css']
})
export class TerminalComponent implements OnInit {
  @ViewChild('terminal') terminalElement: ElementRef;
  ws: WebSocket;
  xterm: Terminal;
  decoder = new TextDecoder('utf-8');

  get ip() {
    return this.data.addr.split(':')[0];
  }

  constructor(
    public dialogRef: MatDialogRef<TerminalComponent>,
    @Inject(MAT_DIALOG_DATA) private data: any,
    private nodeService: NodeService,
    private authService: AuthService,
    private translate: TranslateService,
  ) { }

  ngOnInit() {
    forkJoin(
      this.nodeService.getManagerPort(),
      this.authService.authToken(),
    ).subscribe(res => {
      this.ws = new WebSocket(this.buildUrl(res[0], res[1]));
      this.ws.binaryType = 'arraybuffer';
      this.ws.onerror = this.ws.onclose = this.close.bind(this);
      this.ws.onopen = this.initXterm.bind(this);
      this.ws.onmessage = (event: MessageEvent) => {
        this.xterm.write(this.decoder.decode(event.data));
      };
    });
  }

  private initXterm() {
    this.xterm = new Terminal({
      cursorBlink: true,
    });

    this.xterm.open(this.terminalElement.nativeElement);
    this.xterm.on('data', data => this.ws.send('\x00' + data));
    this.xterm.focus();

    fit(this.xterm);
  }

  private close() {
    this.translate.get('actions.terminal.exitting').subscribe(str => {
      const hasXterm = !!this.xterm;

      if (hasXterm) {
        this.disableInput();
        this.xterm.writeln(str);
      }

      setTimeout(() => this.dialogRef.close(), hasXterm ? 2000 : 0);
    });
  }

  private disableInput() {
    this.xterm.setOption('disableStdin', true);
  }

  private buildUrl(port, token) {
    const hostname = window.location.hostname.replace('localhost', '127.0.0.1');

    return `ws://${hostname}:${port}/term`
      + `?url=ws://${this.data.addr}/node/run/term&token=${token}`;
  }
}
