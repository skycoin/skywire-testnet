import { Component, Inject, OnInit } from '@angular/core';
import { AppsService } from '../../../../../services/apps.service';
import { LogMessage, NodeApp } from '../../../../../app.datatypes';
import { MAT_DIALOG_DATA } from '@angular/material';

@Component({
  selector: 'app-log',
  templateUrl: './log.component.html',
  styleUrls: ['./log.component.scss'],
  host: {'class': 'app-log-container'}
})
export class LogComponent implements OnInit {
  app: NodeApp;
  logMessages: LogMessage[] = [];

  constructor(
    @Inject(MAT_DIALOG_DATA) private data: any,
    private appsService: AppsService,
  ) {
    this.app = data.app;
  }

  ngOnInit() {
    this.appsService.getLogMessages(this.app.key).subscribe(log => this.logMessages = log);
  }
}
