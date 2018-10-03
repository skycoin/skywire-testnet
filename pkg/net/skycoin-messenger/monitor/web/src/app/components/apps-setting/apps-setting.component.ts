import { Component, OnInit } from '@angular/core';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { ApiService, ConnectServiceInfo, AlertService } from '../../service';
import { MatDialogRef } from '@angular/material';

@Component({
  // tslint:disable-next-line:component-selector
  selector: 'apps-setting',
  templateUrl: 'apps-setting.component.html',
  styleUrls: ['./apps-setting.component.scss']
})

export class AppsSettingComponent implements OnInit {
  socks_items: Array<AppSetting> = [];
  ssh_items: Array<AppSetting> = [];
  settingForm = new FormGroup({
    sshs: new FormControl('', Validators.required),
    sshc: new FormControl('', Validators.required),
    sshc_conf: new FormControl('', Validators.required),
    sshc_conf_nodeKey: new FormControl(''),
    sshc_conf_appKey: new FormControl(''),
    sockss: new FormControl('', Validators.required),
    socksc: new FormControl('', Validators.required),
    socksc_conf: new FormControl('', Validators.required),
    socksc_conf_nodeKey: new FormControl(''),
    socksc_conf_appKey: new FormControl(''),
  });
  sshc = 'sshc';
  sshs = 'sshs';
  sockss = 'sockss';
  socksc = 'socksc';
  socksc_opts: Array<ConnectServiceInfo> = [];
  sshc_opts: Array<ConnectServiceInfo> = [];
  addr = '';
  key = '';
  version = '';
  client = '';
  constructor(private api: ApiService, private dialogRef: MatDialogRef<AppsSettingComponent>, private alert: AlertService) { }

  ngOnInit() {
    const data = new FormData();
    data.append('client', this.socksc);
    this.api.getClientConnection(data).subscribe(info => {
      this.socksc_opts = info;
      data.set('client', this.sshc);
      this.api.getClientConnection(data).subscribe(i => {
        this.sshc_opts = i;
      });
    });

    data.append('key', this.key);
    this.api.getAutoStart(this.addr, data).subscribe((config) => {
      // console.log('auto config:', config);
      this.settingForm.patchValue(config);
      this.version = config.version;
    });
  }
  save() {
    let isCheck = true;
    const data = new FormData();
    const json = this.settingForm.value;
    if (json[this.sshc]) {
      if (!json['sshc_conf_nodeKey'] || !json['sshc_conf_appKey']) {
        isCheck = false;
      }
    }
    if (json[this.socksc]) {
      if (!json['socksc_conf_nodeKey'] || !json['socksc_conf_appKey']) {
        isCheck = false;
      }
    }
    if (!isCheck) {
      this.alert.error('Please fill in the required parameters for the full client.');
      return;
    }
    // json['sshc_conf'] = '';
    // json['socksc_conf'] = '';
    json['version'] = this.version;
    data.append('data', JSON.stringify(json));
    data.append('key', this.key);
    this.api.setAutoStart(this.addr, data).subscribe((result) => {
      if (result) {
        this.dialogRef.close();
      }
    });
  }
  setKey(client: string) {
    const info: ConnectServiceInfo = this.settingForm.get(`${client}_conf`).value;
    if (info) {
      this.settingForm.get(`${client}_conf_nodeKey`).patchValue(info.nodeKey);
      this.settingForm.get(`${client}_conf_appKey`).patchValue(info.appKey);
    }
  }
}

export interface AppSetting {
  text?: string;
  type?: string;
  opts?: Array<string>;
}
