import { Component, OnInit } from '@angular/core';
import {MatDialogRef, MatSlideToggleChange} from "@angular/material";
import {SshWarningDialogComponent} from "../ssh-warning-dialog/ssh-warning-dialog.component";

@Component({
  selector: 'app-apps-settings',
  templateUrl: './apps-settings.component.html',
  styleUrls: ['./apps-settings.component.scss']
})
export class AppsSettingsComponent implements OnInit {

  constructor(public dialogRef: MatDialogRef<SshWarningDialogComponent>) { }

  ngOnInit() {
  }

  onNodeServerToggle($event: MatSlideToggleChange)
  {
    console.log(`onNodeServerToggle ${$event.checked}`);
  }

  onNodeClientToggle($event: MatSlideToggleChange)
  {
    console.log(`onNodeClientToggle ${$event.checked}`);
  }

  onSSHServerToggle($event: MatSlideToggleChange)
  {
    console.log(`onSSHServerToggle ${$event.checked}`);
  }

  onSSHClientToggle($event: MatSlideToggleChange)
  {
    console.log(`onSSHClientToggle ${$event.checked}`);
  }

  onNodeKeyChanged($event)
  {
    console.log(`onNodeKeyChanged ${$event.target.value}`);
  }

  onNodeAppKeyChanged($event)
  {
    console.log(`onNodeAppKeyChanged ${$event.target.value}`);
  }

  onSSHNodeKeyChanged($event)
  {
    console.log(`onSSHNodeKeyChanged ${$event.target.value}`);
  }

  onSSHAppKeyChanged($event)
  {
    console.log(`onSSHAppKeyChanged ${$event.target.value}`);
  }

  onSaveClicked()
  {
    this.dialogRef.close();
    console.log(`onSaveClicked`);
  }
}
