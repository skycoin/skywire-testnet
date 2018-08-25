import { Component, OnInit, ViewEncapsulation, HostBinding } from '@angular/core';
@Component({
  // tslint:disable-next-line:component-selector
  selector: 'alert-dialog',
  templateUrl: './alert-dialog.component.html',
  styleUrls: ['./alert-dialog.component.scss'],
  encapsulation: ViewEncapsulation.None
})

export class AlertDialogComponent implements OnInit {
  @HostBinding('class') type = 'info';
  title = 'Title';
  message = '';
  constructor() { }

  ngOnInit() {
  }
}
