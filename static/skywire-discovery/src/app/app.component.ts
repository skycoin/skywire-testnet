import { Component, OnInit } from '@angular/core';
import { environment as env } from '../environments/environment';
@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent {
  isManager = env.isManager;
  listitems = [{ text: 'Node List', link: '/', icon: 'list' }, { text: 'Update Password', link: '/updatePass', icon: 'lock' }];
  option = { selected: true };
  constructor() {
  }
}
