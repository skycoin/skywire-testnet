import {Component, Input, OnInit} from '@angular/core';
import {MatDialogRef, MatSlideToggleChange} from '@angular/material';
import {NodeService} from '../../../../../services/node.service';
import {KeyPairState} from '../../../../layout/keypair/keypair.component';
import {AutoStartConfig, Keypair} from '../../../../../app.datatypes';

@Component({
  selector: 'app-startup-config',
  templateUrl: './startup-config.component.html',
  styleUrls: ['./startup-config.component.css']
})
export class StartupConfigComponent implements OnInit {
  private validKeyPair = true;
  private autoStartConfig: AutoStartConfig;
  protected hasKeyPair = true;

  protected appConfigField: string;
  protected nodeKeyConfigField: string;
  protected appKeyConfigField: string;
  protected autoStartTitle: string;

  @Input() automaticStartTitle;

  public constructor(
    public dialogRef: MatDialogRef<StartupConfigComponent>,
    private nodeService: NodeService,
  ) {}

  save() {
    this.nodeService.setAutoStartConfig(this.autoStartConfig).subscribe();
    this.dialogRef.close();
  }

  protected get nodeKey(): string {
    return this.autoStartConfig[this.nodeKeyConfigField];
  }

  protected get appKey(): string {
    return this.autoStartConfig[this.appKeyConfigField];
  }

  protected get isAutoStartChecked(): boolean {
    return this.autoStartConfig[this.appConfigField];
  }

  protected get isFormValid() {
    return this.validKeyPair;
  }

  ngOnInit() {
    this.nodeService.getAutoStartConfig().subscribe(config => this.autoStartConfig = config);
  }
}
