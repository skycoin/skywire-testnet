import { Component, Inject, OnInit } from '@angular/core';
import { FormControl, FormGroup } from '@angular/forms';
import { MAT_DIALOG_DATA } from '@angular/material';
import { AppsService } from '../../../../../../services/apps.service';

@Component({
  selector: 'app-sshs-whitelist',
  templateUrl: './sshs-whitelist.component.html',
  styleUrls: ['./sshs-whitelist.component.css']
})
export class SshsWhitelistComponent implements OnInit {
  form: FormGroup;
  whitelistedNodes: string[] = [];

  constructor(
    @Inject(MAT_DIALOG_DATA) private data: any,
    private appsService: AppsService,
  ) {
    this.whitelistedNodes = data.app.allow_nodes;
  }

  ngOnInit() {
    this.form = new FormGroup({
      'keys': new FormControl('', [this.validateKeys]),
    });
  }

  save() {
    if (this.form.valid) {
      const keys = (this.form.get('keys').value as string).split(',');

      this.appsService.startSshServer(keys).subscribe();
    }
  }

  private validateKeys(control: FormControl) {
    const value: string = control.value;

    const isInvalid = value
      .split(',')
      .map(key => key.length === 66)
      .some(result => result === false);

    return { invalid: isInvalid };
  }
}
