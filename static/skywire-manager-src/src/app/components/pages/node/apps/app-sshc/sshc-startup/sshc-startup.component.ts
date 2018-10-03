import {StartupConfigComponent} from '../../startup-config/startup-config.component';
import { Component } from '@angular/core';

@Component({
  selector: 'app-sshc-startup-config',
  templateUrl: '../../startup-config/startup-config.component.html',
  styleUrls: ['../../startup-config/startup-config.component.css']
})
export class SshcStartupComponent extends StartupConfigComponent {
  appKeyConfigField = 'sshc_conf_appKey';
  appConfigField = 'sshc';
  nodeKeyConfigField = 'sshc_conf_nodeKey';
  autoStartTitle = 'apps.sshc.auto-startup';
}
