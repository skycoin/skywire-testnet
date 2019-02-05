import {Component, OnChanges} from '@angular/core';
import { SshsStartupComponent } from './sshs-startup/sshs-startup.component';
import { SshsWhitelistComponent } from './sshs-whitelist/sshs-whitelist.component';
import {NodeAppButtonComponent} from '../node-app-button/node-app-button.component';

@Component({
  selector: 'app-app-sshs',
  templateUrl: '../node-app-button/node-app-button.component.html',
  styleUrls: ['./app-sshs.component.css', '../node-app-button/node-app-button.component.scss']
})
export class AppSshsComponent extends NodeAppButtonComponent implements OnChanges {
  title = 'apps.sshs.title';
  name = 'sshs';

  showStartupConfig() {
    this.dialog.open(SshsStartupComponent);
  }

  showWhitelist() {
    this.dialog.open(SshsWhitelistComponent, {
      data: {
        node: this.app,
        app: this.app,
      },
      width: '700px'
    }).beforeClose().subscribe(() => this.setLoading());
  }

  getMenuItems() {
    return [{
      name: 'apps.menu.startup-config',
      callback: this.showStartupConfig.bind(this),
      enabled: true
    }, {
      name: 'apps.menu.whitelist',
      callback: this.showWhitelist.bind(this),
      enabled: this.isRunning
    }, {
      name: 'apps.menu.log',
      callback: this.showLog.bind(this),
      enabled: this.isRunning
    }];
  }

  startApp(): void {
    this.setLoading();
    this.appsService.startSshServerWithoutWhitelist().subscribe(undefined, () => {
      this.setLoading(false);
    });
  }
}
