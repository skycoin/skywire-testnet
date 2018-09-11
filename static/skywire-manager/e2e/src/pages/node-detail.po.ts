import {PATHS} from "../../../src/app/app-routing.module";
import BasePage from "./base-page.po";
import {findById, waitForInvisibility, waitForVisibility} from "../util/selection";
import {NodesListPage} from "./nodes-list.po";
import {by, element} from "protractor";
import {APP_SOCKSC, APP_SSHC, APP_SSHS} from "../util/constants";
import {NodeAppsButtons} from "../components/node-apps.po";
import {StartupConfigDialog} from "../components/startup-config.po";

export class NodeDetailPage extends BasePage {

  path = PATHS.nodeDetail;
  private appsButtons = new NodeAppsButtons();

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
    this.clickElement('nodeListBtn');
  }

  private clickElement(elId: string) {
    const el = findById(elId);
    waitForVisibility(el);
    return el.click();
  }

  clickToggleSshsApp() {
    this.appsButtons.clickToggleSshsApp();
  }

  runningAppsCount() {
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

  startSockscApp() {
    this.appsButtons.startSockscApp();
  }

  startSshcApp() {
    this.appsButtons.startSshcApp();
  }

  waitAppCount0() {
    waitForInvisibility(this.runningApp(APP_SOCKSC));
    waitForInvisibility(this.runningApp(APP_SSHC));
    waitForInvisibility(this.runningApp(APP_SSHS));
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

  clickToggleSocksApp() {
    this.appsButtons.clickToggleSockscApp();
  }

  clickToggleSshcApp() {
    this.appsButtons.clickToggleSshcApp();
  }
}
