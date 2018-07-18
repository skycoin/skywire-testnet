import { Component } from '@angular/core';
import { NodeService } from '../../../services/node.service';
import { Node, NodeApp, NodeInfo } from '../../../app.datatypes';
import { ActivatedRoute, Router } from '@angular/router';

@Component({
  selector: 'app-node',
  templateUrl: './node.component.html',
  styleUrls: ['./node.component.css']
})
export class NodeComponent {
  node: Node;
  nodeApps: NodeApp[] = [];
  nodeInfo: NodeInfo;

  constructor(
    private nodeService: NodeService,
    private route: ActivatedRoute,
    private router: Router,
  ) {
    const key: string = route.snapshot.params['key'];

    nodeService.node(key).subscribe(
      node => {
        this.node = { key, ...node };

        nodeService.setCurrentNode(this.node);

        this.loadData();
      },
      () => router.navigate(['nodes']),
    );
  }

  private loadData() {
    this.nodeService.nodeApps().subscribe(apps => this.nodeApps = apps);
    this.nodeService.nodeInfo().subscribe(info => this.nodeInfo = info);
  }
}
