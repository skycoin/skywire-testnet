import { Component, Input, Input } from '@angular/core';
import { AutoStartConfig, Node, NodeApp, NodeInfo } from '../../../../app.datatypes';
import { NodeService } from '../../../../services/node.service';

@Component({
  selector: 'app-apps',
  templateUrl: './apps.component.html',
  styleUrls: ['./apps.component.css']
})
export class AppsComponent {
  @Input() node: Node;
  @Input() apps: NodeApp[] = [];
  @Input() nodeInfo: NodeInfo;

  getApp(name: string) {
    return this.apps.find(app => app.attributes.some(attr => attr === name));
  }
}

export class AppWrapper {
  @Input() node: Node;
  @Input() app: NodeApp|null;

  get isRunning(): boolean {
    return !!this.app;
  }
}

export class AppAutoStartConfig {
  autoStartConfig: AutoStartConfig;

  constructor(
    private nodeService: NodeService,
  ) {
    this.nodeService.getAutoStartConfig().subscribe(config => this.autoStartConfig = config);
  }
}
