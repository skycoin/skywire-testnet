import { Component, Inject, OnInit } from '@angular/core';
import { Keypair} from '../../../../../../app.datatypes';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material';
import {KeyPairState} from '../../../../../layout/keypair/keypair.component';

@Component({
  selector: 'app-socksc-connect',
  templateUrl: './socksc-connect.component.html',
  styleUrls: ['./socksc-connect.component.css']
})
export class SockscConnectComponent implements OnInit {
  keypair: Keypair;

  private discoveries = [];

  constructor(
    public dialogRef: MatDialogRef<SockscConnectComponent>,
    @Inject(MAT_DIALOG_DATA) private data: any,
  ) { }

  ngOnInit() {
    this.discoveries = this.data.discoveries;
  }

  keypairChange({keyPair, valid}: KeyPairState) {
    if (valid) {
      this.keypair = keyPair;
    } else {
      this.keypair = null;
    }
  }

  connect(keypair?: Keypair) {
    if (keypair) {
      this.keypair = keypair;
    }

    this.dialogRef.close(this.keypair);
  }
}
