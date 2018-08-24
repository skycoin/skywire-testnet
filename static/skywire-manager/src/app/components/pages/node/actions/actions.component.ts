import { Component, Input } from '@angular/core';
import { NodeService } from '../../../../services/node.service';
import { Node, NodeInfo } from '../../../../app.datatypes';
import { MatDialog, MatSnackBar } from '@angular/material';
import { ConfigurationComponent } from './configuration/configuration.component';
import { TerminalComponent } from './terminal/terminal.component';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import {UpdateNodeComponent} from "./update-node/update-node.component";

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
    private translate: TranslateService
  ) { }

  reboot() {
    this.nodeService.reboot().subscribe(
      () => {
        this.translate.get('actions.config.success').subscribe(str => {
          this.snackbar.open(str);
          this.router.navigate(['nodes']);
        });
      },
      (e) => this.snackbar.open(e.message),
    );
  }

  update() {
    this.dialog.open(UpdateNodeComponent).afterClosed().subscribe((updated) =>
    {
      if (updated)
      {
        this.snackbar.open(this.translate.instant('actions.update.update-success'));
      }
    });
  }

  configuration() {
    this.dialog.open(ConfigurationComponent, {
      data: {
        node: this.node,
        discoveries: this.nodeInfo.discoveries,
      },
      width: '800px'
    });
  }

  terminal() {
    this.dialog.open(TerminalComponent, {
      width: '700px',
      data: {
        addr: this.node.addr,
      }
    });
  }

  back() {
    this.router.navigate(['nodes']);
  }
}
