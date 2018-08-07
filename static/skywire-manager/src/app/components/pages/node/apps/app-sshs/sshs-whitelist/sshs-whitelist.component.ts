import { Component, Inject, OnInit } from '@angular/core';
import {MAT_DIALOG_DATA, MatTableDataSource} from '@angular/material';
import { AppsService } from '../../../../../../services/apps.service';
import {KeyInputEvent} from "../../../../../layout/key-input/key-input.component";

@Component({
  selector: 'app-sshs-whitelist',
  templateUrl: './sshs-whitelist.component.html',
  styleUrls: ['./sshs-whitelist.component.scss']
})
export class SshsWhitelistComponent implements OnInit {
  displayedColumns = [ 'index', 'key' ];
  dataSource = new MatTableDataSource<string>();
  private keyToAdd: string;

  constructor(
    @Inject(MAT_DIALOG_DATA) private data: any,
    private appsService: AppsService,
  ) {
    this.dataSource.data = data.app.allow_nodes || [];
  }

  save()
  {
    this.appsService.startSshServer(this.keysValues()).subscribe();
  }

  private keysValues(): string[]
  {
    return this.dataSource.data.concat([]);
  }

  ngOnInit(): void {
  }

  addNodeKey()
  {
    let dataCopy = this.keysValues();
    dataCopy.push(this.keyToAdd);

    this.dataSource.data = dataCopy;
  }

  onKeyChange({value, valid}: KeyInputEvent)
  {
    if (valid)
    {
      this.keyToAdd = value;
    }
  }
}
