import { Component, HostBinding, Inject, OnInit } from '@angular/core';
import { AppsService } from '../../../../../services/apps.service';
import { LogMessage, NodeApp } from '../../../../../app.datatypes';
import { MAT_DIALOG_DATA } from '@angular/material';

@Component({
  selector: 'app-log',
  templateUrl: './log.component.html',
  styleUrls: ['./log.component.scss'],
})
export class LogComponent implements OnInit {
  @HostBinding('attr.class') hostClass = 'app-log-container';
  app: NodeApp;
  logMessages: LogMessage[] = [];
  loading = false;

  constructor(
    @Inject(MAT_DIALOG_DATA) private data: any,
    private appsService: AppsService,
  ) {
    this.app = data.app;
  }

  ngOnInit() {
    this.loading = true;
    this.appsService.getLogMessages(this.app.key).subscribe((log) => this.onLogsReceived(log), this.onLogsError.bind(this));
  }

  private onLogsReceived(log: LogMessage[] = []) {
    this.loading = false;
    this.logMessages = log;
  }

  private onLogsError() {
    this.loading = false;
  }
}
