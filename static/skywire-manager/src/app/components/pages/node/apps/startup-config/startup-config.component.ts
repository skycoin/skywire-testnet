import {Component, Input, OnInit} from '@angular/core';
import {MatDialogRef, MatSlideToggleChange} from "@angular/material";
import {NodeService} from "../../../../../services/node.service";
import {KeyPairState} from "../../../../layout/keypair/keypair.component";
import {AutoStartConfig, Keypair} from "../../../../../app.datatypes";

@Component({
  selector: 'app-startup-config',
  templateUrl: './startup-config.component.html',
  styleUrls: ['./startup-config.component.css']
})
export class StartupConfigComponent implements OnInit
{
  private validKeyPair: boolean = true;
  private autoStartConfig: AutoStartConfig;
  protected hasKeyPair: boolean = true;

  protected appConfigField: string;
  protected nodeKeyConfigField: string;
  protected appKeyConfigField: string;
  protected autoStartTitle: string;

  @Input() automaticStartTitle;

  public constructor(
    private nodeService: NodeService,
    public dialogRef: MatDialogRef<StartupConfigComponent>
  ) {}

  save()
  {
    this.nodeService.setAutoStartConfig(this.autoStartConfig).subscribe();
    this.dialogRef.close();
  }

  private get keyPair(): Keypair
  {
    return {
      nodeKey: this.nodeKey,
      appKey: this.appKey
    }
  }

  protected get nodeKey(): string
  {
    return this.autoStartConfig[this.nodeKeyConfigField];
  }

  protected get appKey(): string
  {
    return this.autoStartConfig[this.appKeyConfigField];
  }

  protected get isAutoStartChecked(): boolean
  {
    return this.autoStartConfig[this.appConfigField];
  }

  private keypairChange({ keyPair, valid}: KeyPairState)
  {
    if (valid)
    {
      this.autoStartConfig[this.nodeKeyConfigField] = keyPair.nodeKey;
      this.autoStartConfig[this.appKeyConfigField] = keyPair.appKey;
    }

    this.validKeyPair = valid;
  }

  protected get isFormValid()
  {
    return this.validKeyPair;
  }

  private toggle(event: MatSlideToggleChange)
  {
    this.autoStartConfig[this.appConfigField] = event.checked;
  }

  ngOnInit()
  {
    this.nodeService.getAutoStartConfig().subscribe(config => this.autoStartConfig = config);
  }
}
