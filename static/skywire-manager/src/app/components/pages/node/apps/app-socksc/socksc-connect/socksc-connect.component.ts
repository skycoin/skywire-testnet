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
export class SockscConnectComponent implements OnInit {
  readonly serviceKey = 'sockss';
  readonly limit = 5;

  connectMode = 1;
  keypair: Keypair;
  currentPage = 1;
  pages = 1;
  discoveries: string[];
  discovery: string;
  results: SearchResultItem[] = [];
  count = 0;

  constructor(
    @Inject(MAT_DIALOG_DATA) private data: any,
    private dialogRef: MatDialogRef<SockscConnectComponent>,
    private nodeService: NodeService,
  ) {
    this.discoveries = data.discoveries;
    this.discovery = this.discoveries[0];
  }

  ngOnInit() {
  }

  search() {
    this.nodeService.searchServices(this.serviceKey, this.currentPage, this.limit, this.discovery)
      .subscribe((result: SearchResult) => {
        this.results = result.result;
        this.count = result.count;
        this.pages = Math.floor(this.count / this.limit);
      });
  }

  changeConnectMode(mode) {
    this.connectMode = mode;

    if (mode === 2) {
      this.search();
    }
  }

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

  prevPage() {
    this.currentPage = Math.max(1, this.currentPage - 1);
    this.search();
  }

  nextPage() {
    this.currentPage = Math.min(this.pages, this.currentPage + 1);
    this.search();
  }
}
