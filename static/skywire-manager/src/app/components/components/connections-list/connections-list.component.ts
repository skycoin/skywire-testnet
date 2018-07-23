import {Component, Input, OnInit} from '@angular/core';
import {Node, NodeTransport} from "../../../app.datatypes";
import {MatTableDataSource} from "@angular/material";

@Component({
  selector: 'app-connections-list',
  templateUrl: './connections-list.component.html',
  styleUrls: ['./connections-list.component.scss']
})
export class ConnectionsListComponent implements OnInit
{
  displayedColumns: string[] = ['index', 'upload_total', 'download_total', 'from_node', 'from_app', 'to_node', 'to_app'];
  dataSource = new MatTableDataSource<NodeTransport>();
  @Input() connections: NodeTransport[] = [];

  constructor() {
    this.dataSource = new MatTableDataSource<NodeTransport>();
  }

  ngOnInit()
  {
    this.dataSource.data = this.connections;
  }

}
