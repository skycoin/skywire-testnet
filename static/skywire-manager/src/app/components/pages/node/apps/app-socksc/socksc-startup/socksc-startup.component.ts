import { Component } from '@angular/core';
import { AppAutoStartConfig } from '../../apps.component';
import { NodeService } from '../../../../../../services/node.service';
import { MatSlideToggleChange } from '@angular/material';
import { Keypair } from '../../../../../../app.datatypes';

@Component({
  selector: 'app-socksc-startup',
  templateUrl: './socksc-startup.component.html',
  styleUrls: ['./socksc-startup.component.css']
})
export class SockscStartupComponent extends AppAutoStartConfig {
  constructor(
    private nodeService: NodeService,
  ) {
    super(nodeService);
  }

  save() {
    this.nodeService.setAutoStartConfig(this.autoStartConfig).subscribe();
  }

  keypairChange(keypair: Keypair) {
    this.autoStartConfig.socksc_conf_nodeKey = keypair.nodeKey;
    this.autoStartConfig.socksc_conf_appKey = keypair.appKey;
  }

  toggle(event: MatSlideToggleChange) {
    this.autoStartConfig.socksc = event.checked;
  }
}
