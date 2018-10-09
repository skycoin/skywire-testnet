import { Component, OnInit, ViewChild } from '@angular/core';
import { MatIconRegistry, MatTooltip } from '@angular/material';
import { ApiService, WalletAddress } from '../../service';

@Component({
  selector: 'app-wallet',
  templateUrl: 'wallet.component.html',
  styleUrls: ['./wallet.component.scss']
})

export class WalletComponent implements OnInit {
  @ViewChild('copyTooltip') tooltip: MatTooltip;
  key = '';
  balance = '0.00000';
  address = '**********************************';
  records = [];
  constructor(private icon: MatIconRegistry, private api: ApiService) {
  }

  ngOnInit() {
    this.icon.registerFontClassAlias('fa');
    this.getWalletAddress();
  }
  getWalletAddress() {
    if (this.key) {
      const data = new FormData();
      data.append('pk', this.key);
      this.api.getWalletNewAddress(data).subscribe((info: WalletAddress) => {
        console.log('new address:', info);
        if (info) {
          // TODO Code 0 or 1
          this.address = info.address;
          this.getWalletInfo();
        }
      });
    }
  }
  getWalletInfo() {
    if (this.key) {
      const data = new FormData();
      data.append('pk', this.key);
      this.api.getWalletInfo(data).subscribe((info) => {
        console.log('get info:', info);
        if (info) {
          // TODO Code 0 or 1
          this.balance = info.count;
        }
      });
    }
  }
  copy(result: boolean) {
    if (result) {
      this.tooltip.disabled = false;
      this.tooltip.message = 'copied!';
      this.tooltip.hideDelay = 500;
      this.tooltip.show();
      setTimeout(() => {
        this.tooltip.disabled = true;
      }, 500);
    }
  }
}
