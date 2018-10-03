import { Component } from '@angular/core';
import { SshcStartupComponent } from './sshc-startup/sshc-startup.component';
import { SshcKeysComponent } from './sshc-keys/sshc-keys.component';
import { Keypair } from '../../../../../app.datatypes';
import {MenuItem, NodeAppButtonComponent} from '../node-app-button/node-app-button.component';

@Component({
  selector: 'app-app-sshc',
  templateUrl: '../node-app-button/node-app-button.component.html',
  styleUrls: ['./app-sshc.component.css', '../node-app-button/node-app-button.component.scss']
})
export class AppSshcComponent extends NodeAppButtonComponent {
  title = 'apps.sshc.title';
  name = 'sshc';

  startApp(): void {
    this.dialog.open(SshcKeysComponent).afterClosed().subscribe((keypair: Keypair) => {
      if (keypair) {
        this.setLoading();
        this.appsService.startSshClient(keypair.nodeKey, keypair.appKey).subscribe();
      }
    });
  }

  showStartupConfig() {
    this.dialog.open(SshcStartupComponent);
  }

  protected getMenuItems(): MenuItem[] {
    return [{
      name: 'apps.menu.startup-config',
      callback: this.showStartupConfig.bind(this),
      enabled: true
    }, {
      name: 'apps.menu.log',
      callback: this.showLog.bind(this),
      enabled: this.isRunning
    }];
  }
}
