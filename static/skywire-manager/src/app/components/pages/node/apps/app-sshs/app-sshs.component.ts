import {Component, OnInit} from '@angular/core';
import { AppsService } from '../../../../../services/apps.service';
import { MatDialog } from '@angular/material';
import { SshsStartupComponent } from './sshs-startup/sshs-startup.component';
import { SshsWhitelistComponent } from './sshs-whitelist/sshs-whitelist.component';
import {MenuItem, NodeAppButtonComponent} from "../node-app-button/node-app-button.component";

@Component({
  selector: 'app-app-sshs',
  templateUrl: '../node-app-button/node-app-button.component.html',
  styleUrls: ['./app-sshs.component.css', '../node-app-button/node-app-button.component.scss']
})
export class AppSshsComponent extends NodeAppButtonComponent
{
  protected title="SSH";
  protected icon="laptop";

  showStartupConfig()
  {
    this.dialog.open(SshsStartupComponent);
  }

  showWhitelist()
  {
    this.dialog.open(SshsWhitelistComponent, {
      data: {
        node: this.app,
        app: this.app,
      },
    });
  }

  getMenuItems()
  {
    return [{
      name: 'Startup config',
      callback: this.showStartupConfig.bind(this),
      enabled: true
    }, {
      name: 'Whitelist',
      callback: this.showWhitelist.bind(this),
      enabled: this.isRunning
    }, {
      name: 'Messages',
      callback: this.showLog.bind(this),
      enabled: this.isRunning
    }];
  }

  startApp(): void
  {
    this.appsService.startSshServer().subscribe()
  }
}
