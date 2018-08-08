import {Component, Inject} from '@angular/core';
import {MAT_DIALOG_DATA} from '@angular/material';
import { AppsService } from '../../../../../../services/apps.service';
import {KeyInputComponent} from "../../../../../layout/key-input/key-input.component";
import {EditableKeyComponent} from "../../../../../layout/editable-key/editable-key.component";
import {DatatableProvider} from "../../../../../layout/datatable/datatable.component";

@Component({
  selector: 'app-sshs-whitelist',
  templateUrl: './sshs-whitelist.component.html',
  styleUrls: ['./sshs-whitelist.component.scss']
})
export class SshsWhitelistComponent implements DatatableProvider
{
  constructor(
    @Inject(MAT_DIALOG_DATA) private data: any,
    private appsService: AppsService
  ) {}

  save(values: string[])
  {
    this.appsService.startSshServer(values).subscribe();
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

  getEditableRowData(index: number, currentValue: string)
  {
    return {
      autofocus: true,
      value: currentValue
    }
  }
}
