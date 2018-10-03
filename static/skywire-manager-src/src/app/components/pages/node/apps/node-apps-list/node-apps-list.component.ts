import {Component, Input, OnChanges, SimpleChanges} from '@angular/core';
import {NodeApp} from '../../../../../app.datatypes';
import {MatTableDataSource} from '@angular/material';
import {AppsService} from '../../../../../services/apps.service';

@Component({
  selector: 'app-node-app-list',
  templateUrl: './node-apps-list.component.html',
  styleUrls: ['./node-apps-list.component.scss']
})
export class NodeAppsListComponent implements OnChanges {
  displayedColumns: string[] = ['index', 'key', 'type'];
  dataSource = new MatTableDataSource<NodeApp>();
  @Input() apps: NodeApp[] = [];

  constructor(private appsService: AppsService) { }

  ngOnChanges(changes: SimpleChanges): void {
    if (this.apps) {
      this.apps.sort((app1: NodeApp, app2: NodeApp) => app1.key.localeCompare(app2.key));
    }
    this.dataSource.data = this.apps;
  }

  onCloseAppClicked(appName: string): void {
    this.appsService.closeApp(appName).subscribe();
  }
}
