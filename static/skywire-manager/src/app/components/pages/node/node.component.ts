import { Component, OnInit } from '@angular/core';
import { NodeService } from '../../../services/node.service';
import { Node, NodeApp } from '../../../app.datatypes';
import { ActivatedRoute, Router } from '@angular/router';

@Component({
  selector: 'app-node',
  templateUrl: './node.component.html',
  styleUrls: ['./node.component.scss']
})
export class NodeComponent implements OnInit {
  node: Node;
  nodeApps: NodeApp[] = [];

  constructor(
    private nodeService: NodeService,
    private route: ActivatedRoute,
    private router: Router,
  ) {
    const key = route.snapshot.params['key'];

    nodeService.node(key).subscribe(
      node => {
        this.node = { key, ...node };

        nodeService.nodeApps(this.node.addr).subscribe(apps => this.nodeApps = apps);
      },
      () => router.navigate(['nodes']),
    );
  }

  ngOnInit() {
  }

  back()
  {
    this.router.navigate(['nodes']);
  }
}
