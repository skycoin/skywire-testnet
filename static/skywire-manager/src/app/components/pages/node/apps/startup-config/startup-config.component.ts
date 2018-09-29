import {Component, Input, OnInit} from '@angular/core';
import {MatDialogRef, MatSlideToggleChange} from '@angular/material';
import {NodeService} from '../../../../../services/node.service';
import {AutoStartConfig, Keypair} from '../../../../../app.datatypes';
import {KeyPairState} from '../../../../layout/keypair/keypair.component';

@Component({
  selector: 'app-startup-config',
  templateUrl: './startup-config.component.html',
  styleUrls: ['./startup-config.component.css']
})
export class StartupConfigComponent implements OnInit {
  autoStartConfig: AutoStartConfig;
  private validKeyPair = false;
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

  get keyPair(): Keypair {
    return {
      nodeKey: this.nodeKey,
      appKey: this.appKey
    };
  }

  keypairChange({ keyPair, valid}: KeyPairState) {
    if (valid) {
      this.autoStartConfig[this.nodeKeyConfigField] = keyPair.nodeKey;
      this.autoStartConfig[this.appKeyConfigField] = keyPair.appKey;
    }

    this.validKeyPair = valid;

    console.log(`validKeyPair: ${this.validKeyPair}`);
  }

  protected get nodeKey(): string {
    return this.autoStartConfig[this.nodeKeyConfigField];
  }

  protected get appKey(): string {
    return this.autoStartConfig[this.appKeyConfigField];
  }

  get isAutoStartChecked(): boolean {
    return this.autoStartConfig[this.appConfigField];
  }

  toggle(event: MatSlideToggleChange) {
    this.autoStartConfig[this.appConfigField] = event.checked;
  }

  ngOnInit() {
    this.nodeService.getAutoStartConfig().subscribe(config => this.autoStartConfig = config);
  }

  get formValid() {
    return this.hasKeyPair ? this.validKeyPair : true;
  }
}
