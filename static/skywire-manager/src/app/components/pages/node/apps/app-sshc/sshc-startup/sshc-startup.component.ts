import {StartupConfigComponent} from '../../startup-config/startup-config.component';

export class SshcStartupComponent extends StartupConfigComponent {
  appKeyConfigField = 'sshc_conf_appKey';
  appConfigField = 'sshc';
  nodeKeyConfigField = 'sshc_conf_nodeKey';
  autoStartTitle = 'Automatically start SSH server';
}
