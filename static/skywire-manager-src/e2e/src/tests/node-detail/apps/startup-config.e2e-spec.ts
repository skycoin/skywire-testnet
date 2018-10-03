import {browser} from "protractor";
import {NodeDetailPage} from "../../../pages/node-detail.po";
import {assertEqual} from "@angular/core/src/render3/assert";
import {NODE_PUBLIC_KEY, PUBLIC_KEY_1, PUBLIC_KEY_2} from "../../../util/constants";
import {getKeyPair} from "../../../util/selection";

/*describe('Apps startup config', () => {
  let page: NodeDetailPage;

  beforeEach(() => {
    browser.restart();
    browser.waitForAngularEnabled(false);
    page = new NodeDetailPage();
  });

  it('SSHs startup config.', () => {
    browser.waitForAngularEnabled(false);
    page.navigateTo();

    let configDialog = page.openSshStartupConfig(),
        toggled = configDialog.isAutomaticStartToggled();

    configDialog.toggleAutomaticStart();
    configDialog.save();

    page.openSshStartupConfig();

    expect(configDialog.isAutomaticStartToggled()).toEqual(!toggled);
  });

  it('SOCKSc startup config.', () => {
    browser.waitForAngularEnabled(false);
    page.navigateTo();

    let configDialog = page.openSockscsStartupConfig(),
        toggled = configDialog.isAutomaticStartToggled();

    configDialog.toggleAutomaticStart();
    configDialog.fillKeyPair(PUBLIC_KEY_1, PUBLIC_KEY_2);

    configDialog.save();

    page.openSshStartupConfig();

    expect(configDialog.isAutomaticStartToggled()).toEqual(!toggled);
    expect(configDialog.getKeyPair()).toEqual({nodeKey: PUBLIC_KEY_1, appKey: PUBLIC_KEY_2});
  });
});*/
