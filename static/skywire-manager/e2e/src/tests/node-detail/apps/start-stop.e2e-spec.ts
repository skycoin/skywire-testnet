import {browser, by, element, ExpectedConditions} from "protractor";
import {NodesListPage} from "../../../pages/nodes-list.po";
import {LoginPage} from "../../../pages/login";
import {NodeDetailPage} from "../../../pages/node-detail.po";
import {NODE_PUBLIC_KEY} from "../../../util/constants";

describe('Apps start/stop', () => {
  let page: NodeDetailPage;

  beforeEach(() => {
    browser.restart();
    browser.waitForAngularEnabled(false);
    page = new NodeDetailPage();
  });

  it('SSHS app starts and stops correctly.', () => {

    browser.waitForAngularEnabled(false);
    page.navigateTo();

    page.clickToggleSshsApp();

    expect(page.isSshsAppRunning()).toBeTruthy();

    page.clickToggleSshsApp();

    page.waitAppCount0();

    expect(page.runningAppsCount()).toEqual(0);
  });

  it('SOCKSc app starts and stops correctly.', () => {

    browser.waitForAngularEnabled(false);

    page.navigateTo();

    page.startSockscApp();

    expect(page.isSockscAppRunning()).toBeTruthy();

    page.clickToggleSocksApp();

    page.waitAppCount0();

    expect(page.runningAppsCount()).toEqual(0);
  });

  it('SSHS app starts and stops correctly.', () => {

    browser.waitForAngularEnabled(false);
    page.navigateTo();

    page.startSshcApp();

    expect(page.isSshcAppRunning()).toBeTruthy();

    page.clickToggleSshcApp();

    page.waitAppCount0();

    expect(page.runningAppsCount()).toEqual(0);
  });
});
