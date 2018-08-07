import {StartupConfigComponent} from "../../startup-config/startup-config.component";

export class SshsStartupComponent extends StartupConfigComponent
{
  hasKeyPair = false;
  appConfigField = "sshs";
  autoStartTitle = "Automatically start SSH server";

  protected get isFormValid()
  {
    return true;
  }
}
