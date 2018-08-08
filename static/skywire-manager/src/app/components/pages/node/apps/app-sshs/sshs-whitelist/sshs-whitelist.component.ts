import {Component, ComponentFactoryResolver, EventEmitter, Inject, OnInit, ViewChild} from '@angular/core';
import {MAT_DIALOG_DATA, MatTableDataSource} from '@angular/material';
import { AppsService } from '../../../../../../services/apps.service';
import {KeyInputComponent, KeyInputEvent} from "../../../../../layout/key-input/key-input.component";
import {EditableKeyComponent} from "../../../../../layout/editable-key/editable-key.component";

@Component({
  selector: 'app-sshs-whitelist',
  templateUrl: './sshs-whitelist.component.html',
  styleUrls: ['./sshs-whitelist.component.scss']
})
export class SshsWhitelistComponent implements OnInit
{
  displayedColumns = [ 'index', 'key', 'remove' ];
  dataSource = new MatTableDataSource<string>();
  private valueToAdd: string;

  @ViewChild(KeyInputComponent) newKeyInput: KeyInputComponent;
  removeRowTooltipText: string = "Remove key";
  addButtonTitle: string = "Add to list";

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
    private componentFactoryResolver: ComponentFactoryResolver
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

  onAddBtnClicked()
  {
    let dataCopy = this.keysValues();
    dataCopy.push(this.valueToAdd);
    this.updateKeys(dataCopy);

    this.newKeyInput.clear();
    this.valueToAdd = null;
  }

  onAddRowValueChange({value, valid}: KeyInputEvent)
  {
    if (valid)
    {
      this.valueToAdd = value;
    }
  }

  onRemoveBtnClicked(position)
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

  getEditableRowComponentClass() {
    return EditableKeyComponent;
  }

  getAddRowComponentClass() {
    return KeyInputComponent;
  }

  getAddRowData()
  {
    return {
      required: false,
      placeholder: 'Enter node key',
      onKeyChangeSubscriber: this.onAddRowValueChange.bind(this)
    }
  }

  getEditableRowData(index: number)
  {
    return {
      autofocus: true,
      value: this.keysValues()[index],
      subscriber: this.onKeyAtPositionChanged.bind(this, index)
    }
  }
}
