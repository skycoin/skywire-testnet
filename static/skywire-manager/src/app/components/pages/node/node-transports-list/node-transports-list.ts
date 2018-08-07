import {Component, Input, OnChanges, OnInit, SimpleChanges} from '@angular/core';
import {Node, NodeTransport} from "../../../../app.datatypes";
import {MatTableDataSource} from "@angular/material";

@Component({
  selector: 'node-transports-list',
  templateUrl: './node-transports-list.html',
  styleUrls: ['./node-transports-list.scss']
})
export class NodeTransportsList implements OnChanges
{
  displayedColumns: string[] = ['index', 'upload_total', 'download_total', 'from_node', 'from_app', 'to_node', 'to_app'];
  dataSource = new MatTableDataSource<NodeTransport>();
  @Input() connections: NodeTransport[] = [];

  constructor() {
    this.dataSource = new MatTableDataSource<NodeTransport>();
  }

  ngOnChanges(changes: SimpleChanges): void
  {
    this.dataSource.data = this.connections;
  }
}
