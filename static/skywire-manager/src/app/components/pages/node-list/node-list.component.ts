import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import {NodeService} from '../../../services/node.service';
import {Node} from '../../../app.datatypes';
import { Subscription } from 'rxjs';
import { MatDialog, MatSnackBar, MatTableDataSource } from '@angular/material';
import {Router} from '@angular/router';
import { ButtonComponent } from '../../layout/button/button.component';
import { EditLabelComponent } from './edit-label/edit-label.component';
import { TranslateService } from '@ngx-translate/core';
import {isOnline} from "../../../utils/nodeUtils";

interface NodeStatus extends Node
{
  online?: boolean;
}

@Component({
  selector: 'app-node-list',
  templateUrl: './node-list.component.html',
  styleUrls: ['./node-list.component.scss'],
})
export class NodeListComponent implements OnInit, OnDestroy {
  @ViewChild('refreshButton') refreshButton: ButtonComponent;
  dataSource = new MatTableDataSource<NodeStatus>();
  displayedColumns: string[] = ['enabled', 'index', 'label', 'key', 'start_time', 'actions'];

  private subscriptions: Subscription;

  constructor(
    private nodeService: NodeService,
    private router: Router,
    private snackbar: MatSnackBar,
    private dialog: MatDialog,
    private translate: TranslateService,
  ) { }

  ngOnInit() {
    this.subscriptions = this.nodeService.allNodes().subscribe(allNodes => {
      this.dataSource.data = allNodes as NodeStatus[];
      this.computeOnlineStatus(allNodes);
    });

    this.refresh();
  }

  ngOnDestroy() {
    this.subscriptions.unsubscribe();
  }

  refresh() {
    this.refreshButton.loading();
    this.subscriptions.add(
      this.nodeService.refreshNodes(
        () => this.refreshButton.reset(),
        () => this.onError(),
      )
    );
  }

  showEditLabelDialog(node: Node) {
    this.dialog.open(EditLabelComponent, {
      data: { node },
    });
  }

  getLabel(node: Node) {
    return this.nodeService.getLabel(node);
  }

  viewNode(node) {
    this.router.navigate(['nodes', node.key]);
  }

  private fetchNodeInfo(key: string) {
    this.nodeService.node(key).subscribe(node => {
        const dataCopy = [].concat(this.dataSource.data),
            updateNode = dataCopy.find(n => n.key === key);

        if (updateNode) {
          updateNode.addr = node.addr;
          this.refreshList(dataCopy);
        }
      },
      () => this.router.navigate(['login']));
  }

  private refreshList(data: Node[]) {
    this.dataSource.data = data;
  }

  private onError() {
    this.translate.get('nodes.error-load').subscribe(str => {
      this.snackbar.open(str);
    });
  }

  /**
   * For each node, request its info to determine if it is online (discovered by the Skycoin network)
   * or offline (not discovered, but seen by the manager).

   * @param allNodes
   */
  private computeOnlineStatus(allNodes: Node[]) {
    allNodes.forEach((node) => this.computeSingleNodeOnlineStatus(node));
  }

  private computeSingleNodeOnlineStatus(node: Node) {
    this.nodeService.nodeInfo(node).subscribe((nodeInfo) =>
    {
      let currentList = this.dataSource.data;
      let node = currentList.find((node) => node.key === node.key);
      node.online = isOnline(nodeInfo);
      this.dataSource.data = currentList;
    });
  }
}
