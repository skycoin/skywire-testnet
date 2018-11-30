import { Component, ElementRef, Inject, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { NodeService } from '../../../../../services/node.service';
import { forkJoin } from 'rxjs';
import { AuthService } from '../../../../../services/auth.service';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material';
import { Terminal } from 'xterm';
import { proposeGeometry } from 'xterm/lib/addons/fit/fit';
import { TranslateService } from '@ngx-translate/core';

@Component({
  selector: 'app-terminal',
  templateUrl: './terminal.component.html',
  styleUrls: ['./terminal.component.scss']
})
export class TerminalComponent implements OnInit, OnDestroy {
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

  ngOnDestroy() {
    this.ws.close();
  }

  private initXterm() {
    this.xterm = new Terminal({
      cursorBlink: true,
    });

    this.xterm.open(this.terminalElement.nativeElement);
    this.xterm.on('data', data => this.ws.send('\x00' + data));
    this.xterm.focus();

    const geometry = proposeGeometry(this.xterm);
    this.xterm.resize(geometry.cols, geometry.rows);
    this.ws.send(`\x00stty rows ${geometry.rows} cols ${geometry.cols}\nclear\n`);
  }

  private close() {
    this.translate.get('actions.terminal.exiting').subscribe(str => {
      const hasXterm = !!this.xterm;

      if (hasXterm) {
        this.xterm.setOption('disableStdin', true);
        this.xterm.writeln(str);
      }

      setTimeout(() => this.dialogRef.close(), hasXterm ? 2000 : 0);
    });
  }

  private buildUrl(port, token) {
    const hostname = window.location.hostname.replace('localhost', '127.0.0.1');

    return `ws://${hostname}:${port}/term`
      + `?url=ws://${this.data.addr}/node/run/term&token=${token}`;
  }
}
