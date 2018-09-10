import {Component, Input, OnChanges, SimpleChanges} from '@angular/core';
import {Node, NodeApp, NodeFeedback} from '../../../../../app.datatypes';
import {LogComponent} from '../log/log.component';
import {MatDialog} from '@angular/material';
import {AppsService} from '../../../../../services/apps.service';
import { TranslateService } from '@ngx-translate/core';

@Component({
  selector: 'app-node-app-button',
  templateUrl: './node-app-button.component.html',
  styleUrls: ['./node-app-button.component.scss']
})
export class NodeAppButtonComponent implements OnChanges {
  protected title: string;
  protected icon: string;
  @Input() enabled = true;
  @Input() active = false;
  @Input() hasMessages = false;
  @Input() showMore = true;
  @Input() node: Node;
  @Input() app: NodeApp | null;
  @Input() appFeedback: NodeFeedback | null;
  containerClass: string;
  protected menuItems: MenuItem[] = [];
  protected loading = false;

  public constructor(
    protected dialog: MatDialog,
    protected appsService: AppsService,
    protected translate: TranslateService,
  ) { }

  onAppClicked(): void
  {
    this.toggleAppRun();
  }

  toggleAppRun() {
    this.loading = true;
    if (this.isRunning) {
      this.stopApp();
    }
    else {
      this.startApp();
    }
  }

  get isRunning(): boolean {
    return !!this.app;
  }

  showLog() {
    this.dialog.open(LogComponent, {
      data: {
        app: this.app,
      },
      panelClass: 'app-log-dialog'
    });
  }

  ngOnChanges(changes: SimpleChanges): void {
    let appChanges = changes['app'];
    if (appChanges && appChanges.previousValue !== appChanges.currentValue) {
      this.loading = false;
    }

    this.containerClass = `${'d-flex flex-column align-items-center justify-content-center'} ${this.isRunning ? 'active' : ''}`;
    this.menuItems = this.getMenuItems();

    if (this.isRunning) {
      this.hasMessages = this.appFeedback && this.appFeedback.unread ? this.appFeedback.unread > 0 : false;
    }
  }

  protected getMenuItems(): MenuItem[] { return []; }

  get port() {
    let port = null;
    try {
      port = this.appFeedback.port.toString()
    } catch (e) {}
    return port;
  }

  get appName() {
    return this.app.attributes[0];
  }

  protected startApp() {}

  protected stopApp(): void {
    this.appsService.closeApp(this.appName).subscribe();
  };

  get isFailed() {
    return this.appFeedback && this.appFeedback.failed;
  }

  get statusIconName(): string {
    let statusName = 'stop';
    if (this.isFailed) {
      statusName = 'close';
    }
    else if (this.isRunning) {
      statusName = 'play_arrow';
    }
    return statusName;
  }

  get statusTooltip(): string {
    let key = 'apps.status-stopped-tooltip';

    if (this.isFailed) {
      key = 'apps.status-failed-tooltip';
    }
    else if (this.isRunning) {
      key = 'apps.status-running-tooltip';
    }
    return this.translate.instant(key);
  }

  get status(): string {
    let key = 'apps.status-stopped',
        addPort = false;

    if (this.isFailed) {
      key = 'apps.status-failed';
    }
    else if (this.isRunning) {
      key = 'apps.status-running';
      if (this.port) {
        addPort = true;
      }
    }

    let text = this.translate.instant(key);

    if (addPort) {
      text = text.concat(`: ${this.port}`);
    }

    return text;
  }
}

export interface MenuItem {
  name: string;
  callback: () => any;
  enabled: boolean;
}
