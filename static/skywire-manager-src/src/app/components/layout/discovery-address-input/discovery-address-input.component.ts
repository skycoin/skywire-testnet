import { Component, EventEmitter, HostBinding, Input, Output } from '@angular/core';
import {MatDialog} from '@angular/material';
import {KeyInputEvent} from '../key-input/key-input.component';
import {InputState} from '../validation-input/validation-input.component';
import {DiscoveryAddress} from '../../../app.datatypes';

export interface DiscoveryAddressState {
  valid: boolean;
  value: DiscoveryAddress;
}

@Component({
  selector: 'app-discovery-address-input',
  templateUrl: './discovery-address-input.component.html',
  styleUrls: ['./discovery-address-input.component.scss'],
})
export class DiscoveryAddressInputComponent {
  @HostBinding('attr.class') hostClass = 'discovery-address-input-container';
  @Input() autofocus: boolean;
  @Input() value: DiscoveryAddress = DiscoveryAddressInputComponent.initialState;
  @Input() required = false;
  @Output() valueChanged = new EventEmitter<DiscoveryAddressState>();
  private domainValid = true;
  private keyValid = true;

  constructor(protected dialog: MatDialog) {}

  onKeyChange({value, valid}: KeyInputEvent) {
    this.keyValid = valid;
    this.value.publicKey = value;
    this.emit();
  }

  static get initialState() {
    return {domain: '', publicKey: ''};
  }

  clear() {
    this.value = DiscoveryAddressInputComponent.initialState;
  }

  onDomainChange({valid, value}: InputState) {
    this.domainValid = valid;
    this.value.domain = value;
    this.emit();
  }

  get valid() {
    const emptyValid =
      (this.value.publicKey.length > 0 && this.value.domain.length > 0)
      ||
      (this.value.publicKey.length === 0 && this.value.domain.length === 0);

    return emptyValid && this.keyValid && this.domainValid;
  }

  private emit() {
    this.valueChanged.emit({valid: this.valid, value: this.value});
  }

  set data(data: Data) {
    this.autofocus = data.autofocus;
    this.value = data.value || DiscoveryAddressInputComponent.initialState;
    this.valueChanged.subscribe(data.subscriber);
    data.clearInputEmitter.subscribe(this.clear.bind(this));
  }
}

interface Data {
  autofocus: boolean;
  value: DiscoveryAddress;
  subscriber: (next: DiscoveryAddress) => void;
  clearInputEmitter: EventEmitter<void>;
}
