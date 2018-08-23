import { Injectable } from '@angular/core';
import { forkJoin, interval, Observable, Subject, timer, Unsubscribable } from 'rxjs';
import { AutoStartConfig, Node, NodeApp, NodeData, NodeInfo, SearchResult } from '../app.datatypes';
import { ApiService } from './api.service';
import {filter, flatMap, map, switchMap, take, timeout} from 'rxjs/operators';
import {StorageService} from './storage.service';
import {Subscription} from 'rxjs/internal/Subscription';
import {Observer} from 'rxjs/internal/types';
import {getNodeLabel} from "../utils/nodeUtils";

@Injectable({
  providedIn: 'root'
})
export class NodeService
{
  private nodes = new Subject<Node[]>();
  private nodesSubscription: Unsubscribable;
  private refreshNodeObservable: Observable<Node>;
  private refresNodeTimerSubscription: Subscription;
  private currentNode: Node;
  private currentNodeData = new Subject<NodeData>();
  private nodeDataSubscription: Unsubscribable;

  constructor(
    private apiService: ApiService,
    private storageService: StorageService
  ) {}

  allNodes(): Observable<Node[]> {
    return this.nodes.asObservable();
  }

  /**
   * Fetch getAll endpoint and then add the address
   * for each node calling the node endpoint.
   */
  getAllNodes(): Observable<Node[]> {
    return this.apiService.get('conn/getAll').pipe(
      flatMap(
        nodes => forkJoin(
          nodes.map(
            node => this.node(node.key).pipe(
              map(nodeWithAddress => {
                node.addr = nodeWithAddress.addr;
                return node;
              })
            )
          )
        )
      )
    );
  }

  refreshNodes(successCallback: any = null, errorCallback: any = null): Unsubscribable {
    if (this.nodesSubscription) {
      this.nodesSubscription.unsubscribe();
    }

    return this.nodesSubscription = timer(0, this.storageService.getRefreshTime() * 1000).pipe(flatMap(() => {
      return this.getAllNodes();
    })).subscribe(
      (allNodes: Node[]) =>
      {
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

    return this.nodeDataSubscription = timer(0, refreshMilliseconds).pipe(flatMap(() => forkJoin(
      this.node(this.currentNode.key),
      this.nodeApps(),
      this.nodeInfo(),
      this.getAllNodes()
    ))).subscribe(data => {
      this.currentNodeData.next({
        node: { ...data[0], key: this.currentNode.key },
        apps: data[1] || [],
        info: { ...data[2], transports: data[2].transports || [] },
        allNodes: data[3] || []
      });
    }, errorCallback);
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
    return this.nodeRequest('run/updateNodeConfig');
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
    return this.nodeRequest('update');
  }

  getManagerPort() {
    return this.apiService.post('getPort');
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
