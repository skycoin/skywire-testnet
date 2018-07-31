import { Component } from '@angular/core';
import { NodeService } from '../../../services/node.service';
import {Node, NodeApp, NodeTransport, NodeInfo} from '../../../app.datatypes';
import { ActivatedRoute, Router } from '@angular/router';
import {MatDialog} from "@angular/material";
import {Subscription} from "rxjs/internal/Subscription";

@Component({
  selector: 'app-node',
  templateUrl: './node.component.html',
  styleUrls: ['./node.component.scss']
})
export class NodeComponent {
  node: Node;
  nodeApps: NodeApp[] = [];
  nodeInfo: NodeInfo;
  refreshSeconds: number = 10;

  connectionsList: NodeTransport[] =
  [{
    from_node: '0383321972b09cae77dfab35e0947ad07721a3ce6173d7566a35057d0fc085b1b0',
    to_node: '0375967c2d171f7c71732b53085bd80720bb0e649eae1703c0d05337cd6faa3b9a',
    from_app: '0319ca3757706f1c86d0d3b2b9027de74aee9571b8b7ab2d555170f6ca0037333a',
    to_app: '03a75c3bc56b0329d77aed0347b2815f6e6a772ca0b30730cd51ed1b10793a8f57',
    upload_bandwidth: 1,
    download_bandwidth: 2,
    upload_total: 120,
    download_total: 1
  },
  {
    from_node: '0383321972b09cae77dfab35e0947ad07721a3ce6173d7566a35057d0fc085b1b0',
    to_node: '0375967c2d171f7c71732b53085bd80720bb0e649eae1703c0d05337cd6faa3b9a',
    from_app: '0319ca3757706f1c86d0d3b2b9027de74aee9571b8b7ab2d555170f6ca0037333a',
    to_app: '03a75c3bc56b0329d77aed0347b2815f6e6a772ca0b30730cd51ed1b10793a8f57',
    upload_bandwidth: 1,
    download_bandwidth: 2,
    upload_total: 120,
    download_total: 1
  }];

  appsList: NodeApp[] =
  [{
    key: '0383321972b09cae77dfab35e0947ad07721a3ce6173d7566a35057d0fc085b1b0',
    attributes: ['SSH', 'client'],
    allow_nodes: null
  },
  {
    key: '03a75c3bc56b0329d77aed0347b2815f6e6a772ca0b30730cd51ed1b10793a8f57',
    attributes: ['NODE', 'very long text 123213123123123', 'att3', 'att4'],
    allow_nodes: null
  },
  {
    key: '03a75c3bc56b0329d77aed0347b2815f6e6a772ca0b30730cd51ed1b10793a8f57',
    attributes: ['CLIENT'],
    allow_nodes: null
  }];
  private refreshSubscription: Subscription;

  constructor(
    private nodeService: NodeService,
    private route: ActivatedRoute,
    private router: Router,
    private dialog: MatDialog
  ) {
    this.scheduleNodeRefresh();
  }

  get key(): string
  {
    return this.route.snapshot.params['key'];
  }

  onNodeReceived(node: Node)
  {
    const key: string = this.route.snapshot.params['key'];
    this.node = { key, ...node };
    this.nodeService.setCurrentNode(this.node);

    console.log('onNodeReceived');
    this.loadData();
  }

  private loadData(): void
  {
    this.nodeService.nodeApps().subscribe(apps => this.nodeApps = apps);
    this.nodeService.nodeInfo().subscribe(info => this.nodeInfo = info);
  }

  back(): void
  {
    this.router.navigate(['nodes']);
  }

  onRefreshTimeChanged($seconds): void
  {
    this.refreshSeconds = Math.max(1, $seconds);
    this.scheduleNodeRefresh();
  }

  private onNodeError(): void
  {
    this.router.navigate(['nodes']);
  }

  private scheduleNodeRefresh(): void
  {
    console.log(`scheduleNodeRefresh ${this.refreshSeconds}`);
    if (this.refreshSubscription)
    {
      this.refreshSubscription.unsubscribe();
    }
    this.refreshSubscription = this.nodeService.refreshNode(this.key, this.refreshSeconds).subscribe(
      this.onNodeReceived.bind(this),
      this.onNodeError.bind(this)
    );
  }
}
