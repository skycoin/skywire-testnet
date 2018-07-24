import { Component, Input, Input } from '@angular/core';
import { Node, NodeApp } from '../../../../app.datatypes';

@Component({
  selector: 'app-apps',
  templateUrl: './apps.component.html',
  styleUrls: ['./apps.component.css']
})
export class AppsComponent {
  @Input() node: Node;
  @Input() apps: NodeApp[] = [];

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
