import { Component, OnInit, ViewEncapsulation } from '@angular/core';

@Component({
  selector: 'app-info-dialog',
  templateUrl: 'im-info-dialog.component.html',
  styleUrls: ['./im-info-dialog.component.scss'],
  encapsulation: ViewEncapsulation.None
})

export class ImInfoDialogComponent implements OnInit {
  key = '';
  hint = false;
  constructor() { }

  ngOnInit() { }
}
