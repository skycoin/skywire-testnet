import { Component, OnDestroy, OnInit } from '@angular/core';
import { NodeService } from '../../../services/node.service';
import { Node, NodeData } from '../../../app.datatypes';
import { ActivatedRoute, Router } from '@angular/router';
import { MatDialog, MatSnackBar } from '@angular/material';
import { Subscription } from 'rxjs/internal/Subscription';
import { TranslateService } from '@ngx-translate/core';

@Component({
  selector: 'app-node',
  templateUrl: './node.component.html',
  styleUrls: ['./node.component.scss']
})
export class NodeComponent implements OnInit, OnDestroy {
  nodeData: NodeData;

  private refreshSubscription: Subscription;

  constructor(
    private nodeService: NodeService,
    private route: ActivatedRoute,
    private router: Router,
    private dialog: MatDialog,
    private snackBar: MatSnackBar,
    private translate: TranslateService,
  ) { }

  ngOnInit() {
    const key: string = this.route.snapshot.params['key'];

    this.nodeService.node(key).subscribe(
      (node: Node) => {
        this.nodeService.setCurrentNode({ key, ...node });

        this.refreshSubscription = this.nodeService.nodeData().subscribe((nodeData: NodeData) => {
          this.nodeData = nodeData;
        });

        this.refreshSubscription.add(
          this.nodeService.refreshNodeData(this.onError.bind(this))
        );
      },
      () => this.router.navigate(['nodes'])
    );
  }

  ngOnDestroy() {
    this.refreshSubscription.unsubscribe();
  }

  private onError() {
    this.translate.get('node.error-load').subscribe(str => {
      this.snackBar.open(str);
    });
  }

  /**
   * Node is online if at least one discovery is seeing it.
   */
  private get isOnline()
  {
    let isOnline = false;
    Object.keys(this.nodeData.info.discoveries).map((discovery) =>
    {
      isOnline = isOnline || this.nodeData.info.discoveries[discovery];
    });
    return isOnline;
  }

  getOnlineTooltip(): string
  {
    let tooltip;
    if (this.isOnline)
    {
      tooltip = "Online: the node is correctly detected by the Skycoin network. ";
    }
    else
    {
      tooltip = "Offline: the node is not detected by the Skycoin network. Probably it has no internet connectivity.";
    }
    return tooltip;
  }
}
