import {StartupConfigComponent} from '../../startup-config/startup-config.component';

export class SshsStartupComponent extends StartupConfigComponent {
  hasKeyPair = false;
  appConfigField = 'sshs';
  autoStartTitle = 'apps.sshs.auto-startup';

  protected get isFormValid() {
    return true;
  }
}
