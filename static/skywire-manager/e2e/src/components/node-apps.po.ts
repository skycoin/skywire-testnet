import {findById, waitForVisibility} from "../util/selection";
import {by, ElementFinder} from "protractor";
import {NODE_PUBLIC_KEY} from "../util/constants";

export class NodeAppsButtons {

  private SSHS_BTN: string = 'sshsAppBtn';

  private clickElement(elId: string) {
    const el = findById(elId);
    waitForVisibility(el);
    return el.click();
  }

  clickStartSshsApp() {
    this.clickElement(this.SSHS_BTN);
  }

  clickStartSshcApp() {
    this.clickElement('sshcAppBtn');
  }

  clickStartSockscApp() {
    this.clickElement('sockscAppBtn');
  }

  fillKeyPair(parentElement: ElementFinder, nodeKey: string, appKey: string) {
    parentElement.element(by.id('nodeKeyField')).element(by.tagName('input')).sendKeys(nodeKey);
    parentElement.element(by.id('appKeyField')).element(by.tagName('input')).sendKeys(appKey);
  }

  startSockscApp() {
    this.clickStartSockscApp();

    const dialog = findById('sockscConnectContainer');
    waitForVisibility(dialog);

    this.fillKeyPair(dialog, NODE_PUBLIC_KEY, NODE_PUBLIC_KEY);

    findById('startSockscAppBtn').click();
  }

  startSshcApp() {
    this.clickStartSshcApp();
    const dialog = findById('sshcConnectContainer');
    waitForVisibility(dialog);

    this.fillKeyPair(dialog, NODE_PUBLIC_KEY, NODE_PUBLIC_KEY);

    findById('startSshcAppBtn').click();
  }

  clickSshsStartupConfig() {
    this.getSshsButton();
  }

  private getSshsButton() {
    return findById(this.SSHS_BTN).element(by.id('showMoreBtn'));
  }
}
