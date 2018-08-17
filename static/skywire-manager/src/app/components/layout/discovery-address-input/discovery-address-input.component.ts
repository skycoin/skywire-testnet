import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {MatDialog} from "@angular/material";
import {KeyInputEvent} from "../key-input/key-input.component";
import {InputState} from "../validation-input/validation-input.component";
import {DiscoveryAddress} from "../../../app.datatypes";

export interface DiscoveryAddressState
{
  valid: boolean;
  value: DiscoveryAddress;
}

@Component({
  selector: 'app-discovery-address-input',
  templateUrl: './discovery-address-input.component.html',
  styleUrls: ['./discovery-address-input.component.scss'],
  host: {class: 'discovery-address-input-container'}
})
export class DiscoveryAddressInputComponent implements OnInit
{
  @Input() autofocus: boolean;
  @Input() value: DiscoveryAddress = DiscoveryAddressInputComponent.initialState;
  @Input() required: boolean = false;
  @Output() onValueChanged = new EventEmitter<DiscoveryAddressState>();
  private domainValid: boolean = true;
  private keyValid: boolean = true;

  constructor(protected dialog: MatDialog) {}

  ngOnInit() {}

  onKeyChange({value, valid}: KeyInputEvent)
  {
    this.keyValid = valid;
    this.value.publicKey = value;
    this.emit();
  }

  static get initialState()
  {
    return {domain: '', publicKey: ''};
  }

  clear()
  {
    this.value = DiscoveryAddressInputComponent.initialState;
  }

  onDomainChange({valid, value}: InputState)
  {
    this.domainValid = valid;
    this.value.domain = value;
    this.emit();
  }

  get valid()
  {
    let emptyValid =
      (this.value.publicKey.length > 0 && this.value.domain.length > 0)
      ||
      (this.value.publicKey.length === 0 && this.value.domain.length === 0);

    return emptyValid && this.keyValid && this.domainValid;
  }

  private emit()
  {
    this.onValueChanged.emit({valid: this.valid, value: this.value});
  }

  set data({autofocus, value, subscriber, clearInputEmitter}: { autofocus: boolean, value: DiscoveryAddress, subscriber: (next: DiscoveryAddress) => void, clearInputEmitter: EventEmitter<void>})
  {
    this.autofocus = autofocus;
    this.value = value || DiscoveryAddressInputComponent.initialState;
    this.onValueChanged.subscribe(subscriber);
    clearInputEmitter.subscribe(this.clear.bind(this));
  }
}
