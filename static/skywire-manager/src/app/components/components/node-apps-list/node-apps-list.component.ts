import {Component, Input, OnDestroy, OnInit} from '@angular/core';
import {NodeApp, NodeTransport} from "../../../app.datatypes";
import {MatTableDataSource} from "@angular/material";

@Component({
  selector: 'node-app-keys-list',
  templateUrl: './node-apps-list.component.html',
  styleUrls: ['./node-apps-list.component.scss']
})
export class NodeAppsListComponent implements OnInit
{
  displayedColumns: string[] = ['index', 'key', 'type'];
  dataSource = new MatTableDataSource<NodeApp>();
  @Input() apps: NodeApp[] = [];

  constructor() { }

  ngOnInit()
  {
    this.dataSource.data = this.apps;
  }
}
