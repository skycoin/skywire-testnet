import { Component } from '@angular/core';
import { AppAutoStartConfig } from '../../apps.component';
import { NodeService } from '../../../../../../services/node.service';
import { Keypair } from '../../../../../../app.datatypes';
import { MatSlideToggleChange } from '@angular/material';

@Component({
  selector: 'app-sshc-startup',
  templateUrl: './sshc-startup.component.html',
  styleUrls: ['./sshc-startup.component.css']
})
export class SshcStartupComponent extends AppAutoStartConfig {
  constructor(
    private nodeService: NodeService,
  ) {
    super(nodeService);
  }

  save() {
    this.nodeService.setAutoStartConfig(this.autoStartConfig).subscribe();
  }

  keypairChange(keypair: Keypair) {
    this.autoStartConfig.sshc_conf_nodeKey = keypair.nodeKey;
    this.autoStartConfig.sshc_conf_appKey = keypair.appKey;
  }

  toggle(event: MatSlideToggleChange) {
    this.autoStartConfig.sshc = event.checked;
  }
}
