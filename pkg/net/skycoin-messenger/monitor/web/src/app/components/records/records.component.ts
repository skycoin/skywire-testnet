import { Component, OnInit, ViewChild } from '@angular/core';
import { ApiService, HashSig } from '../../service';
import { MatDialog, MatDialogRef } from '@angular/material';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { IconRefreshComponent } from '../icon-refresh/icon-refresh.component';
@Component({
  selector: 'app-records',
  templateUrl: 'records.component.html',
  styleUrls: ['./records.component.scss']
})

export class RecordsComponent implements OnInit {
  @ViewChild('recordRefresh') recordRefresh: IconRefreshComponent;
  nodeAddr = '';
  nodeKey = '';
  orders = [];
  balance = '0.000000';
  convertible = 0;
  pay = '';
  limit = '10';
  len = 0;
  pageIndex = '1';
  ref: MatDialogRef<any> = null;
  whithdrawForm = new FormGroup({
    count: new FormControl('', [Validators.required, Validators.min(1)]),
    dst: new FormControl('', [Validators.required, Validators.minLength(34), Validators.maxLength(34)]),
  });
  constructor(private api: ApiService, private dialog: MatDialog) { }

  ngOnInit() {
    this.recordRefresh.start();
    this.getBalance();
    this.getOrders();
    this.recordRefresh.stop();
  }
  refresh(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    this.recordRefresh.start();
    this.getBalance();
    this.getOrders();
    setTimeout(() => {
      this.recordRefresh.stop();
    }, 1000);
  }
  addOrder(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    const hash = JSON.stringify({
      pubkey: this.nodeKey,
      timestamp: new Date().getTime(),
    });
    const count = String(this.whithdrawForm.get('count').value * 1000000);
    this.api.getSig(this.nodeAddr, hash).subscribe(s => {
      const data = new FormData();
      data.append('data', hash);
      data.append('sig', s.sig);
      data.append('count', count);
      data.append('dst', this.whithdrawForm.get('dst').value);
      this.api.addOrder(data).subscribe(result => {
        if (result) {
          this.getBalance();
          this.getOrders();
          this.ref.close();
        }
      });
    });
  }
  page(ev) {
    this.limit = ev.pageSize;
    this.pageIndex = ev.pageIndex + 1;
    this.getOrders();
  }
  openWithdraw(ev: Event, content: any) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    this.whithdrawForm.reset();
    this.getConvertible();
    this.ref = this.dialog.open(content);
  }
  getConvertible() {
    const hash = JSON.stringify({
      pubkey: this.nodeKey,
      timestamp: new Date().getTime(),
    });
    this.api.getSig(this.nodeAddr, hash).subscribe(s => {
      const data = new FormData();
      data.append('data', hash);
      data.append('sig', s.sig);
      this.api.getConvertible(data).subscribe(resp => {
        this.convertible = resp.count;
      });
    });
  }
  getBalance() {
    const hash = JSON.stringify({
      pubkey: this.nodeKey,
      timestamp: new Date().getTime(),
    });
    this.api.getSig(this.nodeAddr, hash).subscribe(s => {
      const data = new FormData();
      data.append('data', hash);
      data.append('sig', s.sig);
      this.api.getBalance(data).subscribe(resp => {
        this.balance = resp.total;
      });
    });
  }
  getOrders() {
    const hash: HashSig = {
      pubkey: this.nodeKey,
      timestamp: new Date().getTime(),
    };
    this.api.getSig(this.nodeAddr, JSON.stringify(hash)).subscribe(resp => {
      const data = new FormData();
      data.append('data', JSON.stringify(hash));
      data.append('sig', resp.sig);
      data.append('limit', this.limit);
      data.append('page', this.pageIndex);
      this.api.getNodeOrders(data).subscribe((res) => {
        this.orders = res.orders;
        this.len = res.total;
      });
    });
  }
}

