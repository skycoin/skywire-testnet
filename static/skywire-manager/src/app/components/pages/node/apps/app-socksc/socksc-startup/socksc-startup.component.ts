import {StartupConfigComponent} from '../../startup-config/startup-config.component';

export class SockscStartupComponent extends StartupConfigComponent {
  appKeyConfigField = 'socksc_conf_appKey';
  appConfigField = 'socksc';
  nodeKeyConfigField = 'socksc_conf_nodeKey';
  autoStartTitle = 'apps.socksc.auto-startup';
}
