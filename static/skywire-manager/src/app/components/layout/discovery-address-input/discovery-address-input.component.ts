import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {MatDialog} from "@angular/material";
import {AppsService} from "../../../services/apps.service";
import {KeyInputEvent} from "../key-input/key-input.component";

@Component({
  selector: 'app-discovery-address-input',
  templateUrl: './discovery-address-input.component.html',
  styleUrls: ['./discovery-address-input.component.scss'],
  host: {class: 'discovery-address-input-container'}
})
export class DiscoveryAddressInputComponent implements OnInit
{
  @Input() autofocus: boolean;
  @Input() value: DiscoveryAddress;
  @Input() required: boolean;
  @Output() onValueChanged = new EventEmitter<DiscoveryAddress>();
  @Output() onBlur = new EventEmitter<>();

  constructor(protected dialog: MatDialog,
              protected appsService: AppsService) { }

  ngOnInit() {}

  onKeyChange({value, valid}: KeyInputEvent)
  {
    if (valid)
    {
      this.value.publicKey = value;
      this.emitIfValid();
    }
  }

  onDomainChange(domain: string)
  {
    this.value.domain = domain;
    this.emitIfValid();
  }

  private emitIfValid()
  {
    if (this.value.publicKey && this.value.domain)
    {
      this.onValueChanged.emit(this.value);
    }
  }
}

export interface DiscoveryAddress
{
  domain: string;
  publicKey: string;
}
