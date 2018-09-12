import {Component, Inject, OnInit} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogRef, MatSnackBar} from '@angular/material';
import { AppsService } from '../../../../../../services/apps.service';
import {KeyInputComponent} from '../../../../../layout/key-input/key-input.component';
import {EditableKeyComponent} from '../../../../../layout/editable-key/editable-key.component';
import {DatatableProvider} from '../../../../../layout/datatable/datatable.component';
import { TranslateService } from '@ngx-translate/core';
import {ErrorsnackbarService} from '../../../../../../services/errorsnackbar.service';

@Component({
  selector: 'app-sshs-whitelist',
  templateUrl: './sshs-whitelist.component.html',
  styleUrls: ['./sshs-whitelist.component.scss']
})
export class SshsWhitelistComponent implements DatatableProvider<string>, OnInit {
  private currentWhiteList: string[] = [];
  constructor(
    @Inject(MAT_DIALOG_DATA) private data: any,
    public dialogRef: MatDialogRef<SshsWhitelistComponent>,
    private appsService: AppsService,
    private translate: TranslateService,
    private snackbar: MatSnackBar,
    private errorSnackbar: ErrorsnackbarService
  ) {}

  ngOnInit(): void {
    this.dialogRef.beforeClose().subscribe(() => {
      this._save();
    });
  }

  save(values: string[]) {
    this.currentWhiteList = values;
  }

  _save() {
    if (this.currentWhiteList.length > 0) {
      this.appsService.startSshServer(this.currentWhiteList).subscribe(
        () => this.snackbar.open(this.translate.instant('apps.sshs.whitelist.saved-correctly')),
        () => this.errorSnackbar.open(this.translate.instant('apps.sshs.whitelist.errors.cant-save'))
      );
    }
  }

  getEditableRowComponentClass() {
    return EditableKeyComponent;
  }

  getAddRowComponentClass() {
    return KeyInputComponent;
  }

  getAddRowData() {
    return {
      required: false,
      placeholder: this.translate.instant('apps.sshs.whitelist.enter-key')
    };
  }

  getEditableRowData(index: number, currentValue: string) {
    return {
      autofocus: true,
      value: currentValue
    };
  }
}
