import {Component, Inject} from '@angular/core';
import {MAT_DIALOG_DATA} from '@angular/material';
import { AppsService } from '../../../../../../services/apps.service';
import {KeyInputComponent} from "../../../../../layout/key-input/key-input.component";
import {EditableKeyComponent} from "../../../../../layout/editable-key/editable-key.component";

@Component({
  selector: 'app-sshs-whitelist',
  templateUrl: './sshs-whitelist.component.html',
  styleUrls: ['./sshs-whitelist.component.scss']
})
export class SshsWhitelistComponent
{
  constructor(
    @Inject(MAT_DIALOG_DATA) private data: any,
    private appsService: AppsService
  ) {
  }

  save(keys: string[])
  {
    this.appsService.startSshServer(keys).subscribe();
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
      placeholder: 'Enter node key'
    }
  }

  getEditableRowData(index: number, currentValue: string, rowChangeCallback)
  {
    return {
      autofocus: true,
      value: currentValue,
      subscriber: (value) => rowChangeCallback(index, value)
    }
  }
}
