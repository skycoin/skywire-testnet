import { Component, OnInit, Inject } from '@angular/core';
import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material';

@Component({
  selector: 'app-loading',
  templateUrl: 'loading.component.html'
})

export class LoadingComponent implements OnInit {
  time = 0;
  task = null;
  constructor(public dialogRef: MatDialogRef<LoadingComponent>,@Inject(MAT_DIALOG_DATA) public data: any) {

  }

  ngOnInit() {
    if (this.data.taskTime) {
      this.time = this.data.taskTime;
      this.task = setInterval(() => {
        if (this.time <= 0) {
          clearInterval(this.task);
          this.dialogRef.close();
          return;
        }
        this.time -= 1;
      }, 1000);
    }
  }
}
