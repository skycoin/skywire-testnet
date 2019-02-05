import { Component, OnInit } from '@angular/core';
import { ApiService } from './services/api.service';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent implements OnInit {
  discoveryPublicKey = '';
  nodes = [];
  nodesLoaded = false;
  nodesVisible = false;

  constructor(private apiService: ApiService) { }

  ngOnInit() {
    this.apiService.serverInfo().subscribe(discovery => this.discoveryPublicKey = discovery);
    this.apiService.allNodes().subscribe(nodes => {
      this.nodes = nodes;
      this.nodesLoaded = true;
    });
  }

  showNodes() {
    this.nodesVisible = true;
  }
}
