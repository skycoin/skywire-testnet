import { Injectable } from '@angular/core';
import { Observable, Subject, throwError, timer, Unsubscribable } from 'rxjs';
import { AutoStartConfig, Node, NodeApp, NodeInfo } from '../app.datatypes';
import { ApiService } from './api.service';
import { map } from 'rxjs/operators';

@Injectable({
  providedIn: 'root'
})
export class NodeService {
  private nodes = new Subject<Node[]>();
  private nodesSubscription: Unsubscribable;
  private currentNode: Node;
  private nodeLabels = {};

  constructor(
    private apiService: ApiService,
  ) {
  }

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

  getLabel(key: string) {
    return key in this.nodeLabels ? this.nodeLabels[key] : '';
  }

  setLabel(key: string, label: string) {
    this.nodeLabels[key] = label;
  }

  node(key: string): Observable<Node> {
    return this.apiService.post('conn/getNode', {key}, {type: 'form'});
  }

  setCurrentNode(node: Node) {
    this.currentNode = node;
  }

  nodeApps(): Observable<NodeApp[]> {
    return this.nodeRequest('getApps');
  }

  nodeInfo(): Observable<NodeInfo> {
    return this.nodeRequest('getInfo');
  }

  setNodeConfig(data: any) {
    return this.nodeRequest('run/setNodeConfig', data, {type: 'form'});
  }

  updateNodeConfig() {
    return this.nodeRequest('run/updateNodeConfig');
  }

  getAutoStartConfig(): Observable<AutoStartConfig> {
    return this.nodeRequest('run/getAutoStartConfig', {
      key: this.currentNode.key
    }, {
      type: 'form',
    });
  }

  setAutoStartConfig(config: AutoStartConfig) {
    return this.nodeRequest('run/setAutoStartConfig', {
      key: this.currentNode.key,
      data: JSON.stringify(config),
    }, {
      type: 'form',
    });
  }

  reboot(): Observable<any> {
    return this.nodeRequest('reboot', {}, {responseType: 'text'}).pipe(map(result => {
      if (result.indexOf('darwin') !== -1) {
        throw new Error(result);
      }

      return result;
    }));
  }

  checkUpdate(): Observable<boolean> {
    return this.nodeRequest('run/checkUpdate').pipe(map(result => {
      return result ? result : throwError(new Error('No update available.'));
    }));
  }

  update(): Observable<any> {
    return this.nodeRequest('update');
  }

  nodeRequest(endpoint: string, body: any = {}, options: any = {}) {
    const nodeAddress = this.currentNode.addr;

    options.params = Object.assign(options.params || {}, {
      addr: this.nodeRequestAddress(nodeAddress, endpoint),
    });

    return this.apiService.post('req', body, options);
  }

  private nodeRequestAddress(nodeAddress: string, endpoint: string) {
    return 'http://' + nodeAddress + '/node/' + endpoint;
  }
}
