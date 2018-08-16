import { Component, Input, ViewChild } from '@angular/core';
import { NodeService } from '../../../../services/node.service';
import { Node, NodeInfo } from '../../../../app.datatypes';
import { MatDialog, MatSnackBar } from '@angular/material';
import { ConfigurationComponent } from './configuration/configuration.component';
import { TerminalComponent } from './terminal/terminal.component';
import { ButtonComponent } from '../../../layout/button/button.component';
import { Router } from '@angular/router';

@Component({
  selector: 'app-actions',
  templateUrl: './actions.component.html',
  styleUrls: ['./actions.component.scss']
})
export class ActionsComponent {
  @Input() node: Node;
  @Input() nodeInfo: NodeInfo;

  constructor(
    private nodeService: NodeService,
    private snackbar: MatSnackBar,
    private dialog: MatDialog,
    private router: Router,
  ) { }

  reboot() {
    this.nodeService.reboot().subscribe(
      () => {
        this.router.navigate(['nodes']);
        this.snackbar.open('Node is rebooting');
      },
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

  terminal() {
    this.dialog.open(TerminalComponent, {
      width: '700px',
      id: 'terminal-dialog',
      data: {
        addr: this.node.addr,
      }
    });
  }

  back() {
    this.router.navigate(['nodes']);
  }
}
