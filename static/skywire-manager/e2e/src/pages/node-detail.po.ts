import {PATHS} from "../../../src/app/app-routing.module";
import BasePage from "./base-page.po";
import {findById, waitForVisibility} from "../util/selection";
import {NodesListPage} from "./nodes-list.po";
import {by, element, ElementFinder} from "protractor";
import {NODE_PUBLIC_KEY} from "../util/constants";

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
    const el = findById('nodeListBtn');
    waitForVisibility(el);
    return el.click();
  }

  private clickAppButton(btnId: string) {
    const el = findById(btnId);
    waitForVisibility(el);
    return el.click();
  }

  clickStartSshsApp() {
    this.clickAppButton('sshsAppBtn');
  }

  clickStartSshcApp() {
    this.clickAppButton('sshcAppBtn');
  }

  clickStartSockscApp() {
    this.clickAppButton('sockscAppBtn');
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
}
