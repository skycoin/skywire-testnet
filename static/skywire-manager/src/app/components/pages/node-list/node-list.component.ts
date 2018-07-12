import { Component, OnDestroy } from '@angular/core';
import { NodeService } from '../../../services/node.service';
import { Node } from '../../../app.datatypes';
import { Unsubscribable } from 'rxjs';

@Component({
  selector: 'app-node-list',
  templateUrl: './node-list.component.html',
  styleUrls: ['./node-list.component.css']
})
export class NodeListComponent implements OnDestroy {
  nodes: Node[] = [];
  private subscription: Unsubscribable;

  constructor(
    private nodeService: NodeService,
  ) {
    this.subscription = nodeService.allNodes().subscribe(allNodes => this.nodes = allNodes);
  }

  ngOnDestroy() {
    this.subscription.unsubscribe();
  }

  refresh() {
    this.nodeService.refreshNodes();
  }
}
