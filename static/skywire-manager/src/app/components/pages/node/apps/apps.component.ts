import { Component, Input } from '@angular/core';
import {AutoStartConfig, Node, NodeApp, NodeFeedback, NodeInfo} from '../../../../app.datatypes';
import { NodeService } from '../../../../services/node.service';

@Component({
  selector: 'app-apps',
  templateUrl: './apps.component.html',
  styleUrls: ['./apps.component.scss']
})
export class AppsComponent {
  @Input() node: Node;
  @Input() apps: NodeApp[] = [];
  @Input() nodeInfo: NodeInfo;

  getApp(name: string)
  {
    let app = null;
    if (this.apps)
    {
      app = this.apps.find(app => app.attributes.some(attr => attr === name));
    }
    return app;
  }

  getFeedback(appName: string)
  {
    const appKey = this.getApp(appName) ? this.getApp(appName).key : null;
    let feedback: NodeFeedback;
    if (appKey && this.nodeInfo && this.nodeInfo.app_feedbacks)
    {
      feedback = this.nodeInfo.app_feedbacks.find(fb => fb.key === appKey);
    }
    return feedback;
  }
}
export class AppAutoStartConfig {
  autoStartConfig: AutoStartConfig;

  constructor(
    private _nodeService: NodeService,
  ) {
    this._nodeService.getAutoStartConfig().subscribe(config => this.autoStartConfig = config);
  }
}
