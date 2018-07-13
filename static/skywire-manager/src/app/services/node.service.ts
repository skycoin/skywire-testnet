import { Injectable } from '@angular/core';
import { Subject, timer, Unsubscribable } from 'rxjs';
import { Node } from '../app.datatypes';
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

  allNodes() {
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
}
