import {Component, Inject, OnInit, ViewChild} from '@angular/core';
import {MAT_DIALOG_DATA, MatTableDataSource} from '@angular/material';
import { AppsService } from '../../../../../../services/apps.service';
import {KeyInputComponent, KeyInputEvent} from "../../../../../layout/key-input/key-input.component";

@Component({
  selector: 'app-sshs-whitelist',
  templateUrl: './sshs-whitelist.component.html',
  styleUrls: ['./sshs-whitelist.component.scss']
})
export class SshsWhitelistComponent implements OnInit
{
  displayedColumns = [ 'index', 'key', 'remove' ];
  dataSource = new MatTableDataSource<string>();
  private keyToAdd: string;

  @ViewChild(KeyInputComponent) newKeyInput: KeyInputComponent;

  onKeyAtPositionChanged(position: number, keyValue: string)
  {
    let dataCopy = this.keysValues();
    dataCopy[position] = keyValue;
    this.dataSource.data = dataCopy;
    this.save();
  }

  constructor(
    @Inject(MAT_DIALOG_DATA) private data: any,
    private appsService: AppsService,
  ) {
    this.updateKeys(data.app.allow_nodes || []);
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
    this.updateKeys(dataCopy);

    this.newKeyInput.clear();
  }

  onKeyChange({value, valid}: KeyInputEvent)
  {
    if (valid)
    {
      this.keyToAdd = value;
    }
  }

  removeKey(position)
  {
    let keys = this.keysValues();
    keys.splice(position, 1);
    this.updateKeys(keys);
  }

  private updateKeys(keys: string[])
  {
    this.dataSource.data = keys;
    this.save();
  }
}
