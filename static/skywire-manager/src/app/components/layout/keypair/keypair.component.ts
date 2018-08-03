import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import {FormControl, FormGroup} from '@angular/forms';
import { Keypair } from '../../../app.datatypes';

export interface KeyPairState
{
  keyPair: Keypair;
  valid: boolean;
}

@Component({
  selector: 'app-keypair',
  templateUrl: './keypair.component.html',
  styleUrls: ['./keypair.component.css']
})
export class KeypairComponent implements OnInit
{
  @Input() keypair: Keypair;
  @Output() keypairChange = new EventEmitter<KeyPairState>();
  form: FormGroup;

  ngOnInit()
  {
    this.form = new FormGroup({
      nodeKey: new FormControl('', [this.validateKey]),
      appKey: new FormControl('', [this.validateKey]),
    });

    this.form.valueChanges.subscribe(value =>
    {
      this.keypairChange.emit({
        keyPair: {
          nodeKey: value.nodeKey,
          appKey: value.appKey,
        },
        valid: this.form.valid
      });
    });

    if (this.keypair)
    {
      this.nodeKey.setValue(this.keypair.nodeKey);
      this.appKey.setValue(this.keypair.appKey);
      this.keypairChange.emit({
        keyPair: {
          nodeKey: this.keypair.nodeKey,
          appKey: this.keypair.appKey,
        },
        valid: this.form.valid
      });
    }
  }

  private validateKey(control: FormControl)
  {
    const key: string = control.value;

    if (!key) {
      return { required: true };
    }

    if (key.length < 66) {
      return { invalid: true };
    }
  }

  get nodeKey(): any
  {
    return this.form.get('nodeKey');
  }

  get appKey(): any
  {
    return this.form.get('appKey');
  }
}
