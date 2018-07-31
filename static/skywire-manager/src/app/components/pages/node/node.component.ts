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
  transports: NodeTransport[] = [];

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
    this.nodeService.nodeInfo().subscribe(this.onNodeInfoReceived.bind(this));
  }

  onNodeInfoReceived(info: NodeInfo)
  {
    this.nodeInfo = info;
    this.transports = info.transports || [];
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
