import {PATHS} from "../../../src/app/app-routing.module";
import BasePage from "./base-page.po";
import {findById, waitForInvisibility, waitForVisibility} from "../util/selection";
import {NodesListPage} from "./nodes-list.po";
import {by, element, ElementFinder} from "protractor";
import {APP_SOCKSC, APP_SSHC, APP_SSHS, NODE_PUBLIC_KEY} from "../util/constants";

export class NodeDetailPage extends BasePage {

  path = PATHS.nodeDetail;

  navigateTo() {
    let nodeListPage = new NodesListPage(),
        result = nodeListPage.navigateTo();

    nodeListPage.clickFirstNode();

    return result;
  }

  getContainer() {
    return findById('nodeDetailView');
  }

  isVisible() {
    waitForVisibility(this.getContainer());
    return this.getContainer().isDisplayed();
  }

  getPublicKey() {
    return findById('nodePublicKey').getText();
  }

  getNodeStatus() {
    return findById('nodeOnlineStatus').getText();
  }

  getNodeProtocol() {
    return findById('nodeProtocol').getText();
  }

  clickNodesListButton() {
    this.clickButton('nodeListBtn');
  }

  private clickButton(btnId: string) {
    const el = findById(btnId);
    waitForVisibility(el);
    return el.click();
  }

  clickStartSshsApp() {
    this.clickButton('sshsAppBtn');
  }

  clickStartSshcApp() {
    this.clickButton('sshcAppBtn');
  }

  clickStartSockscApp() {
    this.clickButton('sockscAppBtn');
  }

  runningAppsCount() {
    waitForInvisibility(this.runningApp(APP_SOCKSC));
    waitForInvisibility(this.runningApp(APP_SSHC));
    waitForInvisibility(this.runningApp(APP_SSHS));
    return this.getRunningAppsTable().element(by.tagName('tbody')).all(by.tagName('tr')).count();
  }

  runningApp(appName: string) {
    return element(by.cssContainingText('.node-app-attr', appName));
  }

  isAppRunning(appName: string) {
    const el = this.runningApp(appName);
    waitForVisibility(el);
    return el.isDisplayed();
  }

  isSshsAppRunning() {
    return this.isAppRunning(APP_SSHS)
  }

  isSockscAppRunning() {
    return this.isAppRunning(APP_SOCKSC)
  }

  isSshcAppRunning() {
    return this.isAppRunning(APP_SSHC)
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

  clickStopApp() {
    this.clickFirstAppCloseButton();
  }

  private clickFirstAppCloseButton() {
    this.getFirstRunningAppRow().element(by.id('stopAppBtn')).click();
  }

  private getRunningAppsTable() {
    return findById('nodeRunningAppsTable');
  }

  private getFirstRunningAppRow() {
    return this.getRunningAppsTable().all(by.tagName('tr')).get(1);
  }
}
