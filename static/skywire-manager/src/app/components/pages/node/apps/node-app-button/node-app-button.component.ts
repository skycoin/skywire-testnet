import {Component, Input, OnChanges} from '@angular/core';
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
  @Input() subtitle: string;
  @Input() active = false;
  @Input() hasMessages = false;
  @Input() showMore = true;
  @Input() node: Node;
  @Input() app: NodeApp | null;
  @Input() appFeedback: NodeFeedback | null;
  private containerClass: string;
  protected menuItems: MenuItem[] = [];
  private failed: boolean;

  public constructor(
    protected dialog: MatDialog,
    protected appsService: AppsService,
    protected translate: TranslateService,
  ) { }

  onAppClicked(): void {
    this.startApp();
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

  ngOnChanges(): void {
    this.containerClass = `${'d-flex flex-column align-items-center justify-content-center w-100'} ${this.isRunning ? 'active' : ''}`;
    this.menuItems = this.getMenuItems();

    if (this.isRunning) {
      this.getSubtitle();
      this.hasMessages = this.appFeedback && this.appFeedback.unread ? this.appFeedback.unread > 0 : false;
    }
  }

  protected getMenuItems(): MenuItem[] { return []; }

  private getPortString() {
    return `${this.translate.instant('common.port')}: ${this.appFeedback.port.toString()}`;
  }

  private getSubtitle() {
    this.failed = false;
    this.subtitle = null;

    if (this.appFeedback) {
      if (this.appFeedback.failed) {
        this.failed = true;
      } else if (this.appFeedback.port) {
        this.subtitle = this.getPortString();
      }
    }
  }

  protected startApp() {}
}

export interface MenuItem {
  name: string;
  callback: Function;
  enabled: boolean;
}
