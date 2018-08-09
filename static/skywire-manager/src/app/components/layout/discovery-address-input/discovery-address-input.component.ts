import {Component, Input, OnInit} from '@angular/core';
import {NodeAppButtonComponent} from "../../pages/node/apps/node-app-button/node-app-button.component";
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

  constructor(protected dialog: MatDialog,
              protected appsService: AppsService) { }

  ngOnInit() {}

  onKeyChange({value, valid}: KeyInputEvent)
  {
    if (valid)
    {
      this.value.publicKey = value;
    }
  }

  onDomainChange(domain: string)
  {
    this.value.domain = domain;
  }
}

export interface DiscoveryAddress
{
  domain: string;
  publicKey: string;
}
