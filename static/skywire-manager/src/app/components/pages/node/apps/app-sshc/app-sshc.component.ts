import { Component } from '@angular/core';
import { AppWrapper } from '../apps.component';
import { AppsService } from '../../../../../services/apps.service';
import { MatDialog } from '@angular/material';
import { SshcStartupComponent } from './sshc-startup/sshc-startup.component';
import { SshcKeysComponent } from './sshc-keys/sshc-keys.component';
import { Keypair } from '../../../../../app.datatypes';

@Component({
  selector: 'app-app-sshc',
  templateUrl: './app-sshc.component.html',
  styleUrls: ['./app-sshc.component.css']
})
export class AppSshcComponent extends AppWrapper {
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
