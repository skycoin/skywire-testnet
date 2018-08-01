import { Component } from '@angular/core';
import { AppsService } from '../../../../../services/apps.service';
import { MatDialog } from '@angular/material';
import { SshcStartupComponent } from './sshc-startup/sshc-startup.component';
import { SshcKeysComponent } from './sshc-keys/sshc-keys.component';
import { Keypair } from '../../../../../app.datatypes';
import {MenuItem, NodeAppButtonComponent} from "../node-app-button/node-app-button.component";

@Component({
  selector: 'app-app-sshc',
  templateUrl: '../node-app-button/node-app-button.component.html',
  styleUrls: ['./app-sshc.component.css', '../node-app-button/node-app-button.component.scss']
})
export class AppSshcComponent extends NodeAppButtonComponent
{
  title="SSH Client";
  icon="laptop";

  constructor(
    private appsService: AppsService,
    private dialog: MatDialog,
  ) {
    super(dialog);

    this.menuItems = [{
      name: 'Startup config',
      callback: this.showStartupConfig.bind(this),
      enabled: true
    }, {
      name: 'Messages',
      callback: this.showLog.bind(this),
      enabled: this.isRunning
    }];
  }

  start() {
    this.dialog.open(SshcKeysComponent).afterClosed().subscribe((keypair: Keypair) => {
      if (keypair) {
        this.appsService.startSshClient(keypair.nodeKey, keypair.appKey).subscribe();
      }
    });
  }

  showStartupConfig() {
    this.dialog.open(SshcStartupComponent);
  }
}
