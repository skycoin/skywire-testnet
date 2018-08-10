import { Component, Inject, OnInit } from '@angular/core';
import { Keypair, SearchResult, SearchResultItem } from '../../../../../../app.datatypes';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material';
import { NodeService } from '../../../../../../services/node.service';
import {KeyPairState} from "../../../../../layout/keypair/keypair.component";

@Component({
  selector: 'app-socksc-connect',
  templateUrl: './socksc-connect.component.html',
  styleUrls: ['./socksc-connect.component.css']
})
export class SockscConnectComponent implements OnInit
{
  keypair: Keypair;
  constructor(
    @Inject(MAT_DIALOG_DATA) private data: any,
    private dialogRef: MatDialogRef<SockscConnectComponent>
  ) {}

  ngOnInit() {}

  keypairChange({keyPair, valid}: KeyPairState)
  {
    if (valid)
    {
      this.keypair = keyPair;
    }
  }

  connect(keypair?: Keypair)
  {
    if (keypair)
    {
      this.keypair = keypair;
    }

    this.dialogRef.close(this.keypair);
  }
}
