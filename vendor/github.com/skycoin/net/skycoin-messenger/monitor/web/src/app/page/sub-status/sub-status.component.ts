import { Component, OnInit, ViewEncapsulation, OnDestroy } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { environment as env } from '../../../environments/environment';
import { ApiService, NodeServices, App, Transports } from '../../service';
import { MatSnackBar } from '@angular/material';
import { DataSource } from '@angular/cdk/collections';
import { Observable } from 'rxjs/Observable';
import 'rxjs/add/observable/of';

@Component({
  selector: 'app-sub-status',
  templateUrl: './sub-status.component.html',
  styleUrls: ['./sub-status.component.scss'],
  encapsulation: ViewEncapsulation.None
})
export class SubStatusComponent implements OnInit, OnDestroy {
  displayedColumns = ['index', 'key', 'app'];
  transportColumns = ['index', 'fromNode', 'fromApp', 'toNode', 'toApp'];
  appSource: SubStatusDataSource = null;
  transportSource: SubStatusDataSource = null;
  key = '';
  power = '';
  transports: Array<Transports> = [];
  status: NodeServices = null;
  apps: Array<App> = [];
  task = null;
  isManager = false;
  socketColor = 'close-status';
  sshColor = 'close-status';
  constructor(
    private router: Router,
    private route: ActivatedRoute,
    private api: ApiService,
    private snackBar: MatSnackBar) {
  }

  ngOnInit() {
    this.route.queryParams.subscribe(params => {
      this.key = params.key;
      this.startTask();
      this.power = 'warn';
      this.isManager = env.isManager;
    });
  }
  ngOnDestroy() {
    this.close();
  }
  refresh(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    this.init();
    this.snackBar.open('Refreshed', 'Dismiss', {
      duration: 3000,
      verticalPosition: 'top',
      extraClasses: ['bg-success']
    });
  }
  runSocketServer(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    this.api.runSockServer(this.status.addr).subscribe(isOk => {
      if (isOk) {
        this.socketColor = 'mat-primary';
        console.log('start socket server');
      }
    });
  }
  runSSHServer(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    this.api.runSSHServer(this.status.addr).subscribe(isOk => {
      if (isOk) {
        this.sshColor = 'mat-primary';
        console.log('start ssh server');
      }
    });
  }
  reboot(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    console.log('reboot');
    this.api.reboot(this.status.addr).subscribe(isOk => {
      if (isOk) {
        if (this.task) {
          this.close();
        }
        this.startTask();
      }
    });
  }
  startTask() {
    this.init();
    this.task = setInterval(() => {
      this.init();
    }, 10000);
  }
  close() {
    clearInterval(this.task);
    this.task = null;
  }
  init() {
    const data = new FormData();
    data.append('key', this.key);
    this.api.getNodeStatus(data).subscribe((nodeServices: NodeServices) => {
      if (nodeServices) {
        this.status = nodeServices;
        if (env.isManager && nodeServices.addr) {
          this.api.getTransport(nodeServices.addr).subscribe((allTransports: Array<Transports>) => {
            if (allTransports && allTransports.length > 0) {
              this.transports = allTransports;
              this.transportSource = new SubStatusDataSource(allTransports);
            }
          });
        }
        if (nodeServices.apps) {
          this.appSource = new SubStatusDataSource(this.status.apps);
        }
      }
    });
  }
}
export class SubStatusDataSource extends DataSource<any> {
  size = 0;
  constructor(private apps: any) {
    super();
  }
  connect(): Observable<any> {
    return Observable.of(this.apps);
  }

  disconnect() { }
}
