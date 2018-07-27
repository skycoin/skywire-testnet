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
    this.subscription = nodeService.allNodes().subscribe(allNodes =>
    {
      this.fetchNodesLabelsIfNeeded(allNodes);
      this.dataSource.data = allNodes;
    });
  }

  ngOnDestroy() {
    this.subscription.unsubscribe();
  }

  refresh() {
    this.nodeService.refreshNodes();
  }

  getLabel(node: Node) {
    return this.nodeService.getLabel(node);
  }

  editLabel(value:string, node: Node) {
    this.nodeService.setLabel(node, value);
  }

  viewNode(node) {
    console.log(node);
    this.router.navigate(['nodes', node.key]);
  }

  private fetchNodeInfo(key: string)
  {
    this.nodeService.node(key).subscribe(node =>
      {
        let dataCopy = [].concat(this.dataSource.data),
            updateNode = dataCopy.find((node) => node.key === key);

        if (updateNode)
        {
          updateNode.addr = node.addr;
          this.refreshList(dataCopy)
        }
      },
      () => this.router.navigate(['login']));
  }

  private refreshList(data: Node[])
  {
    this.dataSource.data = data;
  }

  /**
   * A call to fetchNodeInfo is needed in order to obtain the node's IP from
   * which we will get the default label.
   *
   * The the endpoint will only be called once for each node, as the labels are
   * stored in the localStorage afterwards.
   *
   * @param {Node[]} allNodes
   */
  private fetchNodesLabelsIfNeeded(allNodes: Node[]): void
  {
    allNodes.forEach((node) =>
    {
      let nodeLabel = this.nodeService.getLabel(node);
      if (nodeLabel === null)
      {
        this.fetchNodeInfo(node.key);
      }
    });
  }
}
