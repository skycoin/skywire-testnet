import {Component, Input, OnDestroy, OnInit} from '@angular/core';
import {NodeApp, NodeTransport} from "../../../app.datatypes";
import {MatTableDataSource} from "@angular/material";
import * as ClipboardJs from 'clipboard/dist/clipboard.js';

@Component({
  selector: 'node-app-keys-list',
  templateUrl: './node-apps-list.component.html',
  styleUrls: ['./node-apps-list.component.scss']
})
export class NodeAppsListComponent implements OnInit, OnDestroy
{
  displayedColumns: string[] = ['index', 'key', 'type'];
  dataSource = new MatTableDataSource<NodeApp>();
  clipboard: ClipboardJs;
  @Input() apps: NodeApp[] = [];

  constructor() { }

  ngOnInit()
  {
    this.dataSource.data = this.apps;
    this.clipboard = new ClipboardJs('.clipBtn');
  }

  ngOnDestroy()
  {
    this.clipboard.destroy();
  }
}
