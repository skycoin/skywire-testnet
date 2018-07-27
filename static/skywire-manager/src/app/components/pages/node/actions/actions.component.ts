import { Component, Input } from '@angular/core';
import { NodeService } from '../../../../services/node.service';
import { Node, NodeInfo } from '../../../../app.datatypes';
import { MatDialog, MatSnackBar } from '@angular/material';
import { ConfigurationComponent } from './configuration/configuration.component';
import { TerminalComponent } from './terminal/terminal.component';
import {SshWarningDialogComponent} from "./ssh-warning-dialog/ssh-warning-dialog.component";

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
  ) { }

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

  openTerminal(): void
  {
    this.dialog.open(TerminalComponent);
  }
}
