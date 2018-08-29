import { Component, OnInit, ViewEncapsulation, OnDestroy, ViewChild } from '@angular/core';
import { ApiService, Conn, ConnData, ConnsResponse, UserService } from '../../service';
import { DataSource } from '@angular/cdk/collections';
import { Observable } from 'rxjs/Observable';
import { BehaviorSubject } from 'rxjs/BehaviorSubject';
import { Subscription } from 'rxjs/Subscription';
import { Router } from '@angular/router';
import { MatSnackBar } from '@angular/material';
import 'rxjs/add/observable/timer';
import 'rxjs/add/operator/map';
import { Subject } from 'rxjs/Subject';

@Component({
  selector: 'app-dashboard',
  templateUrl: './dashboard.component.html',
  styleUrls: ['./dashboard.component.scss'],
  encapsulation: ViewEncapsulation.None
})
export class DashboardComponent implements OnInit, OnDestroy {
  displayedColumns = ['index', 'label', 'status', 'key', 'seen'];
  dataSource: ConnDataSource | null;
  _database: ConnDatabase | null;
  labelObj = null;
  discoveryPubKey = '';
  constructor(private api: ApiService, private snackBar: MatSnackBar, private router: Router, private user: UserService) { }
  ngOnInit() {
    this.api.checkLogin().subscribe((result) => {
      console.log('isLogin:', result);
    });
    this._database = new ConnDatabase(this.api);
    this.dataSource = new ConnDataSource(this._database);
    this.labelObj = this.user.get(this.user.HOMENODELABLE);
  }

  ngOnDestroy() {
    this.close();
  }
  editLabel(label: string, nodeKey: string) {
    if (!label) {
      return;
    }
    this.user.saveHomeLabel(nodeKey, label);
    this.labelObj = this.user.get(this.user.HOMENODELABLE);
  }
  transportsNodeBy(index, node) {
    return node ? node.key : undefined;
  }
  getLabel(key: string) {
    if (this.labelObj) {
      return this.labelObj[key] ? this.labelObj[key] : '';
    }
    return '';
  }
  status(ago: number) {
    const now = new Date().getTime() / 1000;
    return (now - ago) < 180;
  }
  refresh(ev?: Event) {
    if (ev) {
      ev.stopImmediatePropagation();
      ev.stopPropagation();
      ev.preventDefault();
    }
    this._database.refresh();
    if (ev) {
      this.snackBar.open('Refreshed', 'Dismiss', {
        duration: 3000,
        verticalPosition: 'top',
        extraClasses: ['bg-success']
      });
    }
  }
  openStatus(ev: Event, conn: Conn) {
    if (!conn) {
      this.snackBar.open('Unable to obtain the node state', 'Dismiss', {
        duration: 3000,
        verticalPosition: 'top'
      });
      return;
    }
    this.router.navigate(['node'], { queryParams: { key: conn.key } });
  }
  close() {
    if (this._database) {
      this._database.close();
    }
  }
}

export class ConnDatabase {
  /** Stream that emits whenever the data has been modified. */
  dataChange: BehaviorSubject<Conn[]> = new BehaviorSubject<Conn[]>([]);
  timer: any;
  task = new Subject();
  get data(): Conn[] { return this.dataChange.value; }

  constructor(private api: ApiService) {
    // Fill up the database with 100 users.
    this.task.debounceTime(100).subscribe(() => {
      this.GetConns();
    });
    this.timer = Observable.timer(0, 5000).subscribe(() => {
      this.task.next();
    });
  }
  close() {
    this.timer.unsubscribe();
  }
  GetConns() {
    this.api.getAllNode().map((conns: Array<Conn>) => {
      conns.sort((a, b) => {
        if (a.key !== b.key) {
          return a.key.localeCompare(b.key);
        } else {
          if (a.start_time < b.start_time) {
            return 1;
          }
          if (a.start_time > b.start_time) {
            return -1;
          }
          return 0;
        }
      });
      return conns;
    }).subscribe((conns: Array<Conn>) => {
      this.dataChange.next(conns);
    });
  }
  refresh() {
    this.GetConns();
  }
}


export class ConnDataSource extends DataSource<any> {
  size = 0;
  constructor(private _database: ConnDatabase) {
    super();
  }
  connect(): Observable<Conn[]> {
    return this._database.dataChange;
  }

  disconnect() { }
}
