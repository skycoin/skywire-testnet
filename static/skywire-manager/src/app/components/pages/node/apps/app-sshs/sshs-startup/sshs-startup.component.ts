import { Component } from '@angular/core';
import {MatDialogRef, MatSlideToggleChange} from '@angular/material';
import { AppAutoStartConfig } from '../../apps.component';
import { NodeService } from '../../../../../../services/node.service';
import {SshWarningDialogComponent} from "../../../actions/ssh-warning-dialog/ssh-warning-dialog.component";

@Component({
  selector: 'app-sshs-startup',
  templateUrl: './sshs-startup.component.html',
  styleUrls: ['./sshs-startup.component.css']
})
export class SshsStartupComponent extends AppAutoStartConfig
{
  constructor(
    private nodeService: NodeService,
    public dialogRef: MatDialogRef<SshsStartupComponent>
  ) {
    super(nodeService);
  }

  save()
  {
    this.nodeService.setAutoStartConfig(this.autoStartConfig).subscribe();
    this.dialogRef.close();
  }

  toggle(event: MatSlideToggleChange)
  {
    this.autoStartConfig.sshs = event.checked;
  }
}
