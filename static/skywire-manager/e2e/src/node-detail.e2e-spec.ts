import {browser, by, element, ExpectedConditions} from "protractor";
import {NodesListPage} from "./pages/nodes-list.po";
import {LoginPage} from "./pages/login";
import {NodeDetailPage} from "./pages/node-detail.po";
import {NODE_PUBLIC_KEY} from "./util/constants";

describe('NodeDetail view', () => {
  let page: NodeDetailPage;

  beforeEach(() => {
    browser.restart();
    browser.waitForAngularEnabled(false);
    page = new NodeDetailPage();
  });

  it('NodeDetail shows correct node information', () => {

    browser.waitForAngularEnabled(false);
    page.navigateTo();

    expect(page.isVisible()).toBeTruthy();

    expect(page.getPublicKey()).toEqual(NODE_PUBLIC_KEY);
    expect(page.getNodeStatus()).toEqual("Online");
    expect(page.getNodeProtocol()).toEqual("TCP");
  });

  it('Nodes list button goes back to node list view', () => {

    browser.waitForAngularEnabled(false);
    page.navigateTo();

    page.clickNodesListButton();

    expect(new NodesListPage().isVisible()).toBeTruthy();
  });

  it('SSHS app starts and stops correctly.', () => {

    browser.waitForAngularEnabled(false);
    page.navigateTo();

    page.clickStartSshsApp();

    expect(page.isSshsAppRunning()).toBeTruthy();

    page.clickStopApp();

    expect(page.runningAppsCount()).toEqual(0);
  });

  it('SOCKSc app starts and stops correctly.', () => {

    browser.waitForAngularEnabled(false);

    page.navigateTo();

    page.startSockscApp();

    expect(page.isSockscAppRunning()).toBeTruthy();

    page.clickStopApp();

    expect(page.runningAppsCount()).toEqual(0);
  });

  it('SSHS app starts and stops correctly.', () => {

    browser.waitForAngularEnabled(false);
    page.navigateTo();

    page.startSshcApp();

    expect(page.isSshcAppRunning()).toBeTruthy();

    page.clickStopApp();

    expect(page.runningAppsCount()).toEqual(0);
  });
});
