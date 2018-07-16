import {Component, OnDestroy} from '@angular/core';
import {NodeService} from '../../../services/node.service';
import {Node} from '../../../app.datatypes';
import {Unsubscribable} from 'rxjs';
import {MatTableDataSource} from '@angular/material';
import * as moment from 'moment';

@Component({
  selector: 'app-node-list',
  templateUrl: './node-list.component.html',
  styleUrls: ['./node-list.component.scss']
})
export class NodeListComponent implements OnDestroy {
  private subscription: Unsubscribable;
  dataSource = new MatTableDataSource<Node>();
  displayedColumns: string[] = ['enabled', 'index', 'label', 'key', 'start_time', 'refresh'];


  constructor(
    private nodeService: NodeService,
  ) {
    this.subscription = nodeService.allNodes().subscribe(allNodes => {
      console.log(allNodes);
      this.dataSource.data = allNodes;
    });
  }

  ngOnDestroy() {
    this.subscription.unsubscribe();
  }

  refresh() {
    this.nodeService.refreshNodes();
  }
}
