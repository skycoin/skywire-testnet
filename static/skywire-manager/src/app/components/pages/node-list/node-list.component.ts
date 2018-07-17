import {Component, OnDestroy} from '@angular/core';
import {NodeService} from '../../../services/node.service';
import {Node} from '../../../app.datatypes';
import {Unsubscribable} from 'rxjs';
import {MatTableDataSource} from '@angular/material';
import {Router} from "@angular/router";

@Component({
  selector: 'app-node-list',
  templateUrl: './node-list.component.html',
  styleUrls: ['./node-list.component.scss'],
})
export class NodeListComponent implements OnDestroy {
  private subscription: Unsubscribable;
  dataSource = new MatTableDataSource<Node>();
  displayedColumns: string[] = ['enabled', 'index', 'label', 'key', 'start_time', 'refresh'];


  constructor(
    private nodeService: NodeService,
    private router: Router
  ) {
    this.subscription = nodeService.allNodes().subscribe(allNodes => {
      this.dataSource.data = allNodes;
    });
  }

  ngOnDestroy() {
    this.subscription.unsubscribe();
  }

  refresh() {
    this.nodeService.refreshNodes();
  }

  getLabel(key: string) {
    return this.nodeService.getLabel(key);
  }

  editLabel(value:string, key: string) {
    this.nodeService.setLabel(key,value);
  }


  viewNode(node) {
    console.log(node);
    this.router.navigate(['nodes', node.key]);
  }
}
