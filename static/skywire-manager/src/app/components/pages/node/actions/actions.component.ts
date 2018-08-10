import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { NodeService } from '../../../../services/node.service';
import { Node, NodeInfo } from '../../../../app.datatypes';
import { MatDialog, MatSnackBar } from '@angular/material';
import { ConfigurationComponent } from './configuration/configuration.component';
import { TerminalComponent } from './terminal/terminal.component';
import {SshWarningDialogComponent} from "./ssh-warning-dialog/ssh-warning-dialog.component";
import { ButtonComponent } from '../../../layout/button/button.component';

@Component({
  selector: 'app-actions',
  templateUrl: './actions.component.html',
  styleUrls: ['./actions.component.scss']
})
export class ActionsComponent implements OnInit {
  @Input() node: Node;
  @Input() nodeInfo: NodeInfo;

  @ViewChild('button0') button0: ButtonComponent;
  @ViewChild('button1') button1: ButtonComponent;
  @ViewChild('button2') button2: ButtonComponent;
  @ViewChild('button3') button3: ButtonComponent;
  constructor(
    private nodeService: NodeService,
    private snackbar: MatSnackBar,
    private dialog: MatDialog,
  ) { }

  ngOnInit() {
    // this.button0.loading();
    this.button1.loading();
    // this.button2.loading();
    this.button3.loading();
  }
  reboot() {
    this.nodeService.reboot().subscribe(
      () => console.log('reboot ok'),
      (e) => this.snackbar.open(e.message),
    );
  }

  update() {
    this.nodeService.checkUpdate().subscribe(
      () => console.log('new update available'),
      (e) => console.warn('check update problem', e),
    );
  }

  configuration() {
    this.dialog.open(ConfigurationComponent, {
      data: {
        node: this.node,
        discoveries: this.nodeInfo.discoveries,
      }
    });
  }

  terminal()
  {
    this.dialog.open(SshWarningDialogComponent, {
      data: {
        acceptButtonCallback: this.openTerminal.bind(this),
      }
    });
  }

  openTerminal(): void {
    this.dialog.open(TerminalComponent, {
      width: '700px',
      id: 'terminal-dialog',
      data: {
        addr: this.node.addr,
      }
    });
  }
}
