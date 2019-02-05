import {StartupConfigComponent} from '../../startup-config/startup-config.component';
import { Component } from '@angular/core';

@Component({
  selector: 'app-socksc-startup-config',
  templateUrl: '../../startup-config/startup-config.component.html',
  styleUrls: ['../../startup-config/startup-config.component.css']
})
export class SockscStartupComponent extends StartupConfigComponent {
  appKeyConfigField = 'socksc_conf_appKey';
  appConfigField = 'socksc';
  nodeKeyConfigField = 'socksc_conf_nodeKey';
  autoStartTitle = 'apps.socksc.auto-startup';
}
