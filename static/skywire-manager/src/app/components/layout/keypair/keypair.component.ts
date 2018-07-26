import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { FormControl, FormGroup } from '@angular/forms';
import { Keypair } from '../../../app.datatypes';

@Component({
  selector: 'app-keypair',
  templateUrl: './keypair.component.html',
  styleUrls: ['./keypair.component.css']
})
export class KeypairComponent implements OnInit {
  @Input() keypair: Keypair;
  @Output() keypairChange = new EventEmitter<Keypair>();
  form: FormGroup;

  ngOnInit() {
    this.form = new FormGroup({
      nodeKey: new FormControl('', [this.validateKey]),
      appKey: new FormControl('', [this.validateKey]),
    });

    this.form.valueChanges.subscribe(value => {
      if (this.form.valid) {
        this.keypairChange.emit({
          nodeKey: value.nodeKey,
          appKey: value.appKey,
        });
      }
    });

    if (this.keypair) {
      this.form.get('nodeKey').setValue(this.keypair.nodeKey);
      this.form.get('appKey').setValue(this.keypair.appKey);
    }
  }

  private validateKey(control: FormControl) {
    const key: string = control.value;

    if (!key) {
      return { required: true };
    }

    if (key.length < 66) {
      return { invalid: true };
    }
  }
}
