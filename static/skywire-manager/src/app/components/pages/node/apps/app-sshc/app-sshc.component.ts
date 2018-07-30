import { Component } from '@angular/core';
import { AppWrapper } from '../apps.component';
import { AppsService } from '../../../../../services/apps.service';
import { MatDialog } from '@angular/material';
import { SshcStartupComponent } from './sshc-startup/sshc-startup.component';
import { SshcKeysComponent } from './sshc-keys/sshc-keys.component';
import { Keypair } from '../../../../../app.datatypes';
import {MenuItem} from "../node-app-button/node-app-button.component";

@Component({
  selector: 'app-app-sshc',
  templateUrl: './app-sshc.component.html',
  styleUrls: ['./app-sshc.component.css']
})
export class AppSshcComponent extends AppWrapper
{
  private menuItems: MenuItem[] = [{
    name: 'Startup config',
    callback: this.showStartupConfig.bind(this)
  }, {
    name: 'Messages',
    callback: this.showLog.bind(this)
  }];

  constructor(
    private appsService: AppsService,
    private dialog: MatDialog,
  ) {
    super(dialog);
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
