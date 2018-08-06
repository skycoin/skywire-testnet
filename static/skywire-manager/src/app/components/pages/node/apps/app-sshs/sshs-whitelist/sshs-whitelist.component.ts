import { Component, Inject, OnInit } from '@angular/core';
import {AbstractControl, FormControl, FormGroup} from '@angular/forms';
import {MAT_DIALOG_DATA, MatTableDataSource} from '@angular/material';
import { AppsService } from '../../../../../../services/apps.service';
import StringUtils from "../../../../../../utils/stringUtils";
import {Node} from "../../../../../../app.datatypes";

@Component({
  selector: 'app-sshs-whitelist',
  templateUrl: './sshs-whitelist.component.html',
  styleUrls: ['./sshs-whitelist.component.scss']
})
export class SshsWhitelistComponent implements OnInit {
  form: FormGroup;
  displayedColumns = [ 'index', 'key' ];
  dataSource = new MatTableDataSource<string>();

  constructor(
    @Inject(MAT_DIALOG_DATA) private data: any,
    private appsService: AppsService,
  ) {
    this.form = new FormGroup({
      'keys': new FormControl('', [this.validateKeys.bind(this)]),
    });
    this.dataSource.data = data.app.allow_nodes;
  }

  save()
  {
    if (this.form.valid)
    {
      this.appsService.startSshServer(this.keysValues()).subscribe();
    }
  }

  private validateKeys(control: FormControl)
  {
    const isInvalid = this.keysValues(control)
      .map(key => key.length === 66)
      .some(result => result === false);

    // Must return null if the form is correct
    return isInvalid ? { invalid: isInvalid } : null;
  }

  private get keysInput(): AbstractControl
  {
    return this.form.get('keys');
  }

  private keysValues(control = this.keysInput): string[]
  {
    let value = StringUtils.removeWhitespaces((control.value as string));
    return value.split(',').filter(key => key.length > 0);
  }

  ngOnInit(): void {
  }
}
