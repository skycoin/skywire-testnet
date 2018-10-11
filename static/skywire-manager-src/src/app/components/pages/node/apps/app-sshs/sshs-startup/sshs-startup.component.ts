import {StartupConfigComponent} from '../../startup-config/startup-config.component';
import { Component } from '@angular/core';

@Component({
  selector: 'app-sshs-startup-config',
  templateUrl: '../../startup-config/startup-config.component.html',
  styleUrls: ['../../startup-config/startup-config.component.css']
})
export class SshsStartupComponent extends StartupConfigComponent {
  hasKeyPair = false;
  appConfigField = 'sshs';
  autoStartTitle = 'apps.sshs.auto-startup';
}
