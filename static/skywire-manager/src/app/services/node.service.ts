import { Injectable } from '@angular/core';
import { forkJoin, interval, Observable, Subject, throwError, timer, Unsubscribable } from 'rxjs';
import { AutoStartConfig, Node, NodeApp, NodeData, NodeInfo, SearchResult } from '../app.datatypes';
import { ApiService } from './api.service';
import { filter, flatMap, map, switchMap, take, timeout } from 'rxjs/operators';
import {StorageService} from "./storage.service";
import {Subscription} from "rxjs/internal/Subscription";
import {Observer} from "rxjs/internal/types";

@Injectable({
  providedIn: 'root'
})
export class NodeService {
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

  refreshNodes(successCallback: any = null, errorCallback: any = null): Unsubscribable {
    if (this.nodesSubscription) {
      this.nodesSubscription.unsubscribe();
    }

    return this.nodesSubscription = timer(0, 10000).pipe(flatMap(() => {
      return this.apiService.get('conn/getAll');
    })).subscribe(
      (allNodes: Node[]) => {
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

    return this.nodeDataSubscription = timer(0, 10000).pipe(flatMap(() => forkJoin(
      this.node(this.currentNode.key),
      this.nodeApps(),
      this.nodeInfo(),
    ))).subscribe(data => {
      this.currentNodeData.next({
        node: data[0],
        apps: data[1] || [],
        info: { ...data[2], transports: data[2].transports || [] }
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
  getLabel(node: Node): string | null
  {
    let nodeLabel = this.storageService.getNodeLabel(node.key);
    if (nodeLabel === null)
    {
      nodeLabel = NodeService.getDefaultNodeLabel(node);
      if (nodeLabel !== null)
      {
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

  refreshNode(key: string, refreshSeconds: number): Observable<Node>
  {
    const refreshMillis = refreshSeconds * 1000;

    if (this.refresNodeTimerSubscription)
    {
      this.refresNodeTimerSubscription.unsubscribe();
    }

    this.refreshNodeObservable = Observable.create((observer: Observer<Node>) =>
    {

      this.refresNodeTimerSubscription = timer(0, refreshMillis).subscribe(
        () => this.node(key).subscribe(
          (node) => observer.next(node),
          (err) => observer.error(err)
        ));
    });

    return this.refreshNodeObservable;
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
    return this.nodeRequest('run/checkUpdate').pipe(map(result => {
      return result ? result : throwError(new Error('No update available.'));
    }));
  }

  update(): Observable<any> {
    return this.nodeRequest('update');
  }

  getManagerPort() {
    return this.apiService.post('getPort');
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

  /**
   * (1) Return a name corresponding to the node's IP
   *
   * Manager (IP:192.168.0.2)
   * Node1 (IP:192.168.0.3)
   * Node2 (IP:192.168.0.4)
   * Node3 (IP:192.168.0.5)
   * Node4 (IP:192.168.0.6)
   * Node5 (IP:192.168.0.7)
   * Node6 (IP:192.168.0.8)
   * Node7 (IP:192.168.0.9)
   *
   * @param {Node} node
   * @returns {string}
   */
  public static getDefaultNodeLabel(node: Node): string
  {
    let nodeLabel = null;
    try
    {
      const ipWithourPort = node.addr.split(':')[0],
        nodeNumber = parseInt(ipWithourPort.split('.')[3]);

      if (nodeNumber == 2)
      {
        nodeLabel = 'Manager';
      }
      else if (nodeNumber > 2 && nodeNumber < 8)
      {
        nodeLabel = `Node ${nodeNumber - 2}`;
      }
      else
      {
        nodeLabel = ipWithourPort;
      }
    }
    catch (e) {}

    return nodeLabel;
  }
}
