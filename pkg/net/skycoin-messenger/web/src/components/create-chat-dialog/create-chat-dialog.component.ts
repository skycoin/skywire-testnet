import { Component, OnInit, ViewEncapsulation } from '@angular/core';
import { FormControl, Validators } from '@angular/forms';
import { MdDialogRef } from '@angular/material';

@Component({
  // tslint:disable-next-line:component-selector
  selector: 'create-chat-dialog',
  templateUrl: './create-chat-dialog.component.html',
  styleUrls: ['./create-chat-dialog.component.scss'],
  encapsulation: ViewEncapsulation.None
})

export class CreateChatDialogComponent implements OnInit {
  keys = '';
  keyFormControl = new FormControl('', [
    Validators.required]);
  constructor(private ref: MdDialogRef<CreateChatDialogComponent>) { }

  ngOnInit() {
  }

  onKeyUpEnter(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    if (this.keyFormControl.valid) {
      this.ref.close(this.keyFormControl.value);
    }
  }
}
