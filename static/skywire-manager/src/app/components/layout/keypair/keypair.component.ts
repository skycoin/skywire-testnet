import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { Keypair } from '../../../app.datatypes';
import {KeyInputEvent} from "../key-input/key-input.component";

export interface KeyPairState
{
  keyPair: Keypair;
  valid: boolean;
}

@Component({
  selector: 'app-keypair',
  templateUrl: './keypair.component.html',
  styleUrls: ['./keypair.component.css'],
  host: {'class': 'keypair-component'}
})
export class KeypairComponent implements OnInit
{
  @Input() keypair: Keypair;
  @Output() keypairChange = new EventEmitter<KeyPairState>();
  private nodeKeyValid: boolean = true;
  private appKeyValid: boolean = true;

  onNodeValueChanged({value, valid}: KeyInputEvent)
  {
    this.keypair.nodeKey = value;
    this.nodeKeyValid = valid;
    this.onPairChanged();
  }

  onAppValueChanged({value, valid}: KeyInputEvent)
  {
    this.keypair.appKey = value;
    this.appKeyValid = valid;
    this.onPairChanged();
  }

  onPairChanged()
  {
    this.keypairChange.emit({
      keyPair: this.keypair,
      valid: this.valid
    });
  }

  private get valid()
  {
    return this.nodeKeyValid && this.appKeyValid;
  }

  ngOnInit()
  {
    if (this.keypair)
    {
      this.onPairChanged();
    }
  }
}
