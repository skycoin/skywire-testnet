import { Component } from '@angular/core';
import { MatDialogRef } from '@angular/material';
import { Keypair } from '../../../../../../app.datatypes';

@Component({
  selector: 'app-sshc-keys',
  templateUrl: './sshc-keys.component.html',
  styleUrls: ['./sshc-keys.component.css']
})
export class SshcKeysComponent {
  keypair: Keypair;

  constructor(
    private dialogRef: MatDialogRef<SshcKeysComponent>,
  ) { }

  connect() {
    this.dialogRef.close(this.keypair);
  }

  keypairChange(keypair: Keypair) {
    this.keypair = keypair;
  }
}
