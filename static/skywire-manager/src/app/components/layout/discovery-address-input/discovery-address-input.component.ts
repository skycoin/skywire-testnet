import {Component, EventEmitter, Input, OnChanges, OnInit, Output, SimpleChanges} from '@angular/core';
import {MatDialog} from "@angular/material";
import {KeyInputEvent} from "../key-input/key-input.component";
import {InputState} from "../validation-input/validation-input.component";
const INITIAL_STATE: DiscoveryAddress = {domain: '', publicKey: ''};

@Component({
  selector: 'app-discovery-address-input',
  templateUrl: './discovery-address-input.component.html',
  styleUrls: ['./discovery-address-input.component.scss'],
  host: {class: 'discovery-address-input-container'}
})
export class DiscoveryAddressInputComponent implements OnInit
{
  @Input() autofocus: boolean;
  @Input() value: DiscoveryAddress = INITIAL_STATE;
  @Input() required: boolean;
  @Output() onValueChanged = new EventEmitter<{valid: boolean, value: DiscoveryAddress}>();
  @Output() onBlur = new EventEmitter();
  private domainValid: boolean = false;
  private keyValid: boolean = false;

  constructor(protected dialog: MatDialog) {}

  ngOnInit() {}

  onKeyChange({value, valid}: KeyInputEvent)
  {
    this.keyValid = valid;
    this.value.publicKey = value;
    this.emit();
  }

  clear()
  {
    this.value = INITIAL_STATE;
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
    this.value = value || INITIAL_STATE;
    this.onValueChanged.subscribe(subscriber);
    clearInputEmitter.subscribe(this.clear.bind(this));
  }
}

export interface DiscoveryAddress
{
  domain: string;
  publicKey: string;
}
