import {Component, ViewChild} from '@angular/core';
import { AppAutoStartConfig } from '../../apps.component';
import { NodeService } from '../../../../../../services/node.service';
import {MatDialogRef, MatSlideToggleChange} from '@angular/material';
import {KeypairComponent, KeyPairState} from "../../../../../layout/keypair/keypair.component";

@Component({
  selector: 'app-sshc-startup',
  templateUrl: './sshc-startup.component.html',
  styleUrls: ['./sshc-startup.component.css']
})
export class SshcStartupComponent extends AppAutoStartConfig
{
  @ViewChild(KeypairComponent) keyPairComp: KeypairComponent;
  keyPairValid: boolean = false;

  constructor(
    private nodeService: NodeService,
    public dialogRef: MatDialogRef<SshcStartupComponent>
  ) {
    super(nodeService);
  }

  save()
  {
    this.nodeService.setAutoStartConfig(this.autoStartConfig).subscribe();
    this.dialogRef.close();
  }

  keypairChange({ keyPair, valid}: KeyPairState)
  {
    if (valid)
    {
      this.autoStartConfig.sshc_conf_nodeKey = keyPair.nodeKey;
      this.autoStartConfig.sshc_conf_appKey = keyPair.appKey;
    }

    this.keyPairValid = valid;
  }

  toggle(event: MatSlideToggleChange)
  {
    this.autoStartConfig.sshc = event.checked;
  }
}
