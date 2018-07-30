import { Component, Input } from '@angular/core';
import { MatDialog } from '@angular/material';
import { SockscStartupComponent } from './socksc-startup/socksc-startup.component';
import { SockscConnectComponent } from './socksc-connect/socksc-connect.component';
import { AppsService } from '../../../../../services/apps.service';
import { Keypair, NodeInfo } from '../../../../../app.datatypes';
import {MenuItem, NodeAppButtonComponent} from "../node-app-button/node-app-button.component";

@Component({
  selector: 'app-app-socksc',
  templateUrl: '../node-app-button/node-app-button.component.html',
  styleUrls: ['./app-socksc.component.css', '../node-app-button/node-app-button.component.scss']
})
export class AppSockscComponent extends NodeAppButtonComponent
{
  @Input() nodeInfo: NodeInfo;

  private menuItems: MenuItem[] = [{
    name: 'Startup config',
    callback: this.showStartupConfig.bind(this),
    enabled: true
  }, {
    name: 'Messages',
    callback: this.showLog.bind(this),
    enabled: this.isRunning
  }];

  title="Connect to Node";
  icon="near_me";

  get parsedDiscoveries() {
    return Object.keys(this.nodeInfo.discoveries).map(disc => disc.split('-')[1]);
  }

  constructor(
    private appsService: AppsService,
    private dialog: MatDialog,
  ) {
    super(dialog);
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
}
