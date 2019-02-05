import {clickElement, fillKeyPair, findById, waitForVisibility} from '../util/selection';
import {by, element, ElementFinder} from 'protractor';
import {NODE_PUBLIC_KEY} from '../util/constants';

export class NodeAppsButtons {

  private SSHS_BTN = 'sshsAppBtn';
  private SOCKSC_BTN = 'sockscAppBtn';
  private SSHC_BTN = 'sshcAppBtn';

  getToggleAppButton(appBtnId) {
    return findById(appBtnId).element(by.id('toggleAppButton'));
  }

  clickToggleSshsApp() {
    clickElement(this.getToggleAppButton(this.SSHS_BTN));
  }

  clickToggleSshcApp() {
    clickElement(this.getToggleAppButton(this.SSHC_BTN));
  }

  clickToggleSockscApp() {
    clickElement(this.getToggleAppButton(this.SOCKSC_BTN));
  }

  startSockscApp() {
    this.clickToggleSockscApp();

    const dialog = findById('sockscConnectContainer');
    waitForVisibility(dialog);

    fillKeyPair(dialog, NODE_PUBLIC_KEY, NODE_PUBLIC_KEY);

    findById('startSockscAppBtn').click();
  }

  startSshcApp() {
    this.clickToggleSshcApp();
    const dialog = findById('sshcConnectContainer');
    waitForVisibility(dialog);

    fillKeyPair(dialog, NODE_PUBLIC_KEY, NODE_PUBLIC_KEY);

    findById('startSshcAppBtn').click();
  }

  clickSshsStartupConfig() {
    this.getSshsButton().element(by.id('showMoreBtn')).click();
    clickElement(element.all(by.css('.mat-menu-item')).first());
  }

  clickSockscStartupConfig() {
    this.getSockscButton().element(by.id('showMoreBtn')).click();
    clickElement(element.all(by.css('.mat-menu-item')).first());
  }

  private getSockscButton() {
    const el = findById(this.SOCKSC_BTN);
    waitForVisibility(el);
    return el;
  }

  private getSshsButton() {
    const el = findById(this.SSHS_BTN);
    waitForVisibility(el);
    return el;
  }
}
