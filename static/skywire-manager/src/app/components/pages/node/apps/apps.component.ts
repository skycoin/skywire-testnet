import { Component, Input } from '@angular/core';
import { AutoStartConfig, Node, NodeApp, NodeInfo } from '../../../../app.datatypes';
import { NodeService } from '../../../../services/node.service';
import { LogComponent } from './log/log.component';
import { MatDialog } from '@angular/material';

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

  constructor(
    private _dialog: MatDialog,
  ) { }

  showLog() {
    this._dialog.open(LogComponent, {
      data: {
        app: this.app,
      },
    });
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
