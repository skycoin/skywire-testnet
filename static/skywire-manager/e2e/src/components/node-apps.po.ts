import {clickElement, fillKeyPair, findById, waitForVisibility} from "../util/selection";
import {by, element, ElementFinder} from "protractor";
import {NODE_PUBLIC_KEY} from "../util/constants";

export class NodeAppsButtons {

  private SSHS_BTN: string = 'sshsAppBtn';
  private SOCKSC_BTN: string = 'sockscAppBtn';

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

  startSockscApp() {
    this.clickStartSockscApp();

    const dialog = findById('sockscConnectContainer');
    waitForVisibility(dialog);

    fillKeyPair(dialog, NODE_PUBLIC_KEY, NODE_PUBLIC_KEY);

    findById('startSockscAppBtn').click();
  }

  startSshcApp() {
    this.clickStartSshcApp();
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
    let el = findById(this.SOCKSC_BTN);
    waitForVisibility(el);
    return el;
  }

  private getSshsButton() {
    let el = findById(this.SSHS_BTN);
    waitForVisibility(el);
    return el;
  }
}
