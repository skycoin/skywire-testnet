import { Injectable } from '@angular/core';
import { Observable, Subject, timer, Unsubscribable } from 'rxjs';
import { Node, NodeApp } from '../app.datatypes';
import { ApiService } from './api.service';

@Injectable({
  providedIn: 'root'
})
export class NodeService {
  private nodes = new Subject<Node[]>();
  private nodesSubscription: Unsubscribable;

  constructor(
    private apiService: ApiService,
  ) { }

  allNodes(): Observable<Node[]> {
    this.refreshNodes();

    return this.nodes.asObservable();
  }

  refreshNodes() {
    if (this.nodesSubscription) {
      this.nodesSubscription.unsubscribe();
    }

    this.nodesSubscription = timer(0, 10000).subscribe(() => {
      this.apiService.get('conn/getAll').subscribe((allNodes: Node[]) => {
        this.nodes.next(allNodes);
      });
    });
  }

  node(key: string): Observable<Node> {
    return this.apiService.post('conn/getNode', { key }, { type: 'form' });
  }

  nodeApps(address: string): Observable<NodeApp[]> {
    return this.nodeRequest(address, 'getApps');
  }

  private nodeRequest(nodeAddress: string, endpoint: string, body: any = {}, options: any = {}) {
    options.params = Object.assign(options.params || {}, {
      addr: this.nodeRequestAddress(nodeAddress, endpoint),
    });

    return this.apiService.post('req', body, options);
  }

  private nodeRequestAddress(nodeAddress: string, endpoint: string) {
    return 'http://' + nodeAddress + '/node/' + endpoint;
  }
}
