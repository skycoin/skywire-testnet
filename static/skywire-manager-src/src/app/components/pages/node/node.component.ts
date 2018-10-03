import {Component, OnDestroy, OnInit} from '@angular/core';
import { NodeService } from '../../../services/node.service';
import {Node, NodeData} from '../../../app.datatypes';
import { ActivatedRoute, Router } from '@angular/router';
import { MatDialog} from '@angular/material';
import { Subscription } from 'rxjs/internal/Subscription';
import { TranslateService } from '@ngx-translate/core';
import {ErrorsnackbarService} from '../../../services/errorsnackbar.service';

@Component({
  selector: 'app-node',
  templateUrl: './node.component.html',
  styleUrls: ['./node.component.scss']
})
export class NodeComponent implements OnInit, OnDestroy {
  public showMenu = false;
  nodeData: NodeData;
  managerIp: string;
  managerKey: string;

  private refreshSubscription: Subscription;
  constructor(
    private nodeService: NodeService,
    private route: ActivatedRoute,
    private router: Router,
    private dialog: MatDialog,
    private errorSnackBar: ErrorsnackbarService,
    private translate: TranslateService,
  ) { }

  ngOnInit() {
    const key: string = this.route.snapshot.params['key'];

    this.nodeService.node(key).subscribe(
      (node: Node) => {
        this.nodeService.setCurrentNode({ key, ...node });

        this.refreshSubscription = this.nodeService.nodeData().subscribe((nodeData: NodeData) => {
          // Fake data used to style the list because it is
          // difficult to see real transports while developing.
          // const transport = {
          //   download_bandwidth: 1333323,
          //   download_total: 4323331,
          //   from_app: '02746d5570118259d98e0ee445bc4ae82ecda258cb64e87d5f1f48cc29badb492f',
          //   to_app: '02746d5570118259d98e0ee445bc4ae82ecda258cb64e87d5f1f48cc29badb492f',
          //   from_node: '02746d5570118259d98e0ee445bc4ae82ecda258cb64e87d5f1f48cc29badb492f',
          //   to_node: '02746d5570118259d98e0ee445bc4ae82ecda258cb64e87d5f1f48cc29badb492f',
          //   upload_bandwidth: 100,
          //   upload_total: 33333333
          // };
          // nodeData.info.transports = [transport, transport, transport];
          this.nodeData = nodeData;
        });

        this.refreshSubscription.add(
          this.nodeService.refreshNodeData(this.onError.bind(this))
        );

        this.nodeService.serverInfo().subscribe(info => {
          const data = info.split('-');

          this.managerIp = data[0].replace('localhost', '127.0.0.1');
          this.managerKey = data[1];
        });
      },
      () => this.router.navigate(['nodes'])
    );
  }

  /*get managerIp() {
    let ipText = this.translate.instant('node.details.manager-ip-not-found');
    const manager = this.nodeData.allNodes.find((node) => isManager(node));

    if (manager && manager.addr) {
      ipText = manager.addr;
    }

    return ipText;
  }*/

  ngOnDestroy() {
    this.refreshSubscription.unsubscribe();
  }

  public toggleMenu() {
    this.showMenu = !this.showMenu;
  }

  private onError() {
    this.translate.get('node.error-load').subscribe(str => {
      this.errorSnackBar.open(str);
    });
  }

  get operationalNodesCount(): number {
    return this.nodeData.allNodes.filter((node) => node.online === true).length;
  }
}
