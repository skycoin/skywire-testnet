import { Component, Input } from '@angular/core';
import { AppWrapper } from '../apps.component';
import { LogComponent } from '../log/log.component';
import { MatDialog } from '@angular/material';
import { SockscStartupComponent } from './socksc-startup/socksc-startup.component';
import { SockscConnectComponent } from './socksc-connect/socksc-connect.component';
import { AppsService } from '../../../../../services/apps.service';
import { Keypair, NodeInfo } from '../../../../../app.datatypes';

@Component({
  selector: 'app-app-socksc',
  templateUrl: './app-socksc.component.html',
  styleUrls: ['./app-socksc.component.css']
})
export class AppSockscComponent extends AppWrapper {
  @Input() nodeInfo: NodeInfo;

  get parsedDiscoveries() {
    return Object.keys(this.nodeInfo.discoveries).map(disc => disc.split('-')[1]);
  }

  constructor(
    private appsService: AppsService,
    private dialog: MatDialog,
  ) {
    super();
  }

  connect() {
    this.dialog
      .open(SockscConnectComponent, {
        data: {
          discoveries: this.parsedDiscoveries,
        },
      })
      .afterClosed()
      .subscribe((keypair: Keypair) => {
        if (keypair) {
          this.appsService.startSocksc(keypair.nodeKey, keypair.appKey).subscribe();
        }
      });
  }

  showStartupConfig() {
    this.dialog.open(SockscStartupComponent);
  }

  showLog() {
    this.dialog.open(LogComponent, {
      data: {
        app: this.app,
      },
    });
  }
}
