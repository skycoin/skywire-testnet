import { Component } from '@angular/core';
import { MatSlideToggleChange } from '@angular/material';
import { AppAutoStartConfig } from '../../apps.component';
import { NodeService } from '../../../../../../services/node.service';

@Component({
  selector: 'app-sshs-startup',
  templateUrl: './sshs-startup.component.html',
  styleUrls: ['./sshs-startup.component.css']
})
export class SshsStartupComponent extends AppAutoStartConfig {
  constructor(
    private nodeService: NodeService,
  ) {
    super(nodeService);
  }

  save() {
    this.nodeService.setAutoStartConfig(this.autoStartConfig).subscribe();
  }

  change(event: MatSlideToggleChange) {
    this.autoStartConfig.sshs = event.checked;
  }
}
