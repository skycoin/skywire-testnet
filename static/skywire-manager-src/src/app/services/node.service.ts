import { Injectable } from '@angular/core';
import {
  forkJoin,
  interval,
  Observable,
  of,
  Subject,
  timer,
  Unsubscribable
} from 'rxjs';
import {
  AutoStartConfig,
  Node,
  NodeApp,
  NodeData,
  NodeInfo,
  NodeStatus,
  NodeStatusInfo,
  SearchResult
} from '../app.datatypes';
import { ApiService } from './api.service';
import {
  catchError,
  delay,
  filter,
  finalize,
  flatMap,
  map,
  switchMap,
  take, tap,
  timeout
} from 'rxjs/operators';
import { StorageService } from './storage.service';
import { getNodeLabel, isDiscovered } from '../utils/nodeUtils';

@Injectable({
  providedIn: 'root'
})
export class NodeService {
  private nodes = new Subject<NodeStatusInfo[]>();
  private nodesSubscription: Unsubscribable;
  private currentNode: Node;
  private currentNodeData = new Subject<NodeData>();
  private nodeDataSubscription: Unsubscribable;

  constructor(
    private apiService: ApiService,
    private storageService: StorageService
  ) {}

  allNodes(): Observable<NodeStatusInfo[]> {
    return this.nodes.asObservable();
  }

  /**
   * Fetch getAll endpoint and then for each node, call:
   *
   * 1 - node endpoint
   * 2 - nodeInfo endpoint
   *
   * And join all the information
   */
  getAllNodes(): Observable<NodeStatusInfo[]> {
    return this.apiService.get('conn/getAll').pipe(
      flatMap(nodes => {
          if (nodes.length === 0) {
            return of([]);
          }

          return forkJoin(nodes.map(node => this.node(node.key).pipe(
            map(nodeWithAddress => ({...node, ...nodeWithAddress})),
            catchError(() => of({...node}))
          )));
      }),
      flatMap(nodes => {
        if (nodes.length === 0) {
          return of([]);
        }

        return forkJoin(nodes.map((node: Node) => this.nodeInfo(node).pipe(
          map(nodeInfo => ({...node, status: isDiscovered(nodeInfo) ? NodeStatus.DISCOVERED : NodeStatus.ONLINE})),
          catchError(() => of({...node, status: NodeStatus.UNKNOWN})),
        )));
      }),
      map((nodes: Node[]) => {
        let storedNodes = this.storageService.getNodes();

        if (storedNodes.length === 0) {
          nodes.forEach(node => this.storageService.addNode(node.key));
          storedNodes = this.storageService.getNodes();
        }

        const allNodes = storedNodes.map(nodeKey => ({ key: nodeKey, status: NodeStatus.OFFLINE }));

        return allNodes.reduce((all, current) => {
          const existing = nodes.find(node => node.key === current.key);

          all.push(existing ? existing : current);

          return all;
        }, []);
      })
    );
  }

  /*refreshNodes(successCallback: any = null, errorCallback: any = null): void {
    this.ngZone.runOutsideAngular(() => {
      if (this.nodesSubscription) {
        this.nodesSubscription.unsubscribe();
      }
      this.nodesSubscription = timer(0, this.storageService.getRefreshTime() * 1000).pipe(flatMap(() => {
        return this.getAllNodes();
      })).subscribe(
        (allNodes: NodeStatusInfo[]) => {
          this.ngZone.run(() => {
            this.nodes.next(allNodes);
            if (successCallback) {
              successCallback();
            }
          });
        },
        errorCallback,
      );
    });
  }*/

  refreshNodes(successCallback: any = null, errorCallback: any = null): Unsubscribable {
    if (this.nodesSubscription) {
      this.nodesSubscription.unsubscribe();
    }

    return this.nodesSubscription = timer(0, this.storageService.getRefreshTime() * 1000).pipe(flatMap(() => {
      return this.getAllNodes();
    })).subscribe(
      (allNodes: NodeStatusInfo[]) => {
        this.nodes.next(allNodes);

        if (successCallback) {
          successCallback();
        }
      },
      errorCallback,
    );
  }

  nodeData(): Observable<NodeData> {
    return this.currentNodeData.asObservable();
  }

  refreshNodeData(errorCallback: any = null): Unsubscribable {
    if (this.nodeDataSubscription) {
      this.nodeDataSubscription.unsubscribe();
    }

    const refreshMilliseconds = this.storageService.getRefreshTime() * 1000;

    return this.nodeDataSubscription = timer(0, refreshMilliseconds).subscribe(() => {
      this.requestRefreshNodeData().subscribe(this.notifyNodeDataRefreshed.bind(this), errorCallback);
    });
  }

  notifyNodeDataRefreshed(data: any) {
    this.currentNodeData.next({
      node: { ...data[0], key: this.currentNode.key },
      apps: data[1] || [],
      info: { ...data[2], transports: data[2].transports || [] },
      allNodes: data[3] || []
    });
  }

  requestRefreshNodeData() {
    return forkJoin(
      this.node(this.currentNode.key),
      this.nodeApps(),
      this.nodeInfo(),
      this.getAllNodes()
    );
  }

  refreshAppData() {
    this.requestRefreshNodeData().subscribe(this.notifyNodeDataRefreshed.bind(this));
  }

  /**
   *
   * Get the label for a given node:
   * (1) If no label is stored in the browser's localStorage --> return a name corresponding to the IP
   * (2) If a label has already been stored in localStorage --> return it

   * @param {Node} node
   * @returns {string | null}
   */
  getLabel(node: Node): string | null {
    let nodeLabel = this.storageService.getNodeLabel(node.key);
    if (nodeLabel === null) {
      nodeLabel = getNodeLabel(node);
      if (nodeLabel !== null) {
        this.setLabel(node, nodeLabel);
      }
    }

    return nodeLabel;
  }

  setLabel(node: Node, label: string) {
    this.storageService.setNodeLabel(node.key, label);
  }

  node(key: string): Observable<Node> {
    return this.apiService.post('conn/getNode', {key});
  }

  setCurrentNode(node: Node) {
    this.currentNode = node;
  }

  nodeApps(): Observable<NodeApp[]> {
    return this.nodeRequest('getApps');
  }

  nodeInfo(node?: Node): Observable<NodeInfo> {
    return this.nodeRequest('getInfo', undefined, undefined, node);
  }

  setNodeConfig(data: any) {
    return this.nodeRequest('run/setNodeConfig', data);
  }

  updateNodeConfig() {
    return this.nodeRequest('run/updateNode');
  }

  getAutoStartConfig(): Observable<AutoStartConfig> {
    return this.nodeRequest('run/getAutoStartConfig', {
      key: this.currentNode.key,
    });
  }

  setAutoStartConfig(config: AutoStartConfig) {
    return this.nodeRequest('run/setAutoStartConfig', {
      key: this.currentNode.key,
      data: JSON.stringify(config),
    });
  }

  searchServices(key: string, pages: number, limit: number, discoveryKey: string): Observable<SearchResult> {
    return this.nodeRequest('run/searchServices', {key, pages, limit, discoveryKey})
      .pipe(switchMap(() => {
        return interval(500).pipe(
          flatMap(() => this.nodeRequest('run/getSearchServicesResult')),
          filter(result => result !== null),
          map(result => result[0]),
          take(1),
          timeout(5000),
        );
      }));
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
    return this.nodeRequest('run/checkUpdate');
  }

  update(): Observable<boolean> {
    return this.nodeRequest('run/update');
  }

  getManagerPort() {
    return this.apiService.post('getPort');
  }

  serverInfo(): Observable<string> {
    return this.apiService.get('conn/getServerInfo', { responseType: 'text' });
  }

  /**
   * Calls nodeRequest and after the request has completed, it refreshes the app status. This is intended
   * to force the UI update when a large refresh interval is selected (if it's 60 seconds, the user wouldn't see
   * any change in 1 minute, which would be a very bad user experience).
   *
   */
  nodeRequestWithRefresh(endpoint: string, body: any = {}, options: any = {}, node = this.currentNode) {
    return this.nodeRequest(endpoint, body, options, node).pipe(delay(5000), finalize(() => this.refreshAppData()));
  }

  nodeRequest(endpoint: string, body: any = {}, options: any = {}, node = this.currentNode) {
    const nodeAddress = node.addr;

    options.params = Object.assign(options.params || {}, {
      addr: this.nodeRequestAddress(nodeAddress, endpoint),
    });

    return this.apiService.post('req', body, options);
  }

  private nodeRequestAddress(nodeAddress: string, endpoint: string) {
    return 'http://' + nodeAddress + '/node/' + endpoint;
  }
}
