import { Component, EventEmitter, HostBinding, Input, OnInit, Output } from '@angular/core';
import { Keypair } from '../../../app.datatypes';
import {KeyInputEvent} from '../key-input/key-input.component';

export interface KeyPairState {
  keyPair: Keypair;
  valid: boolean;
}

@Component({
  selector: 'app-keypair',
  templateUrl: './keypair.component.html',
  styleUrls: ['./keypair.component.css'],
})
export class KeypairComponent implements OnInit {
  @HostBinding('attr.class') hostClass = 'keypair-component';
  @Input() keypair: Keypair = {
    nodeKey: '',
    appKey: ''
  };
  @Output() keypairChange = new EventEmitter<KeyPairState>();
  @Input() required = false;
  private nodeKeyValid = false;
  private appKeyValid = false;

  onNodeValueChanged({value, valid}: KeyInputEvent) {
    this.keypair.nodeKey = value;
    this.nodeKeyValid = valid;
    this.onPairChanged();
  }

  onAppValueChanged({value, valid}: KeyInputEvent) {
    this.keypair.appKey = value;
    this.appKeyValid = valid;
    this.onPairChanged();
  }

  onPairChanged() {
    this.keypairChange.emit({
      keyPair: this.keypair,
      valid: this.valid
    });
  }

  private get valid() {
    return this.nodeKeyValid && this.appKeyValid;
  }

  ngOnInit() {
    if (this.keypair) {
      this.onPairChanged();
    }
  }
}
