import {Component, Inject, OnInit} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialogRef} from "@angular/material";

interface DialogData
{
  acceptButtonCallback: Function;
}

@Component({
  selector: 'app-ssh-warning-dialog',
  templateUrl: './ssh-warning-dialog.component.html',
  styleUrls: ['./ssh-warning-dialog.component.scss']
})
export class SshWarningDialogComponent implements OnInit
{
  constructor(
    public dialogRef: MatDialogRef<SshWarningDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: DialogData
  ) {}

  onCancelButtonClicked(): void
  {
    this.dialogRef.close();
  }

  onAcceptButtonClicked(): void
  {
    this.dialogRef.close();
    this.data.acceptButtonCallback();
  }

  ngOnInit() {
  }

}
