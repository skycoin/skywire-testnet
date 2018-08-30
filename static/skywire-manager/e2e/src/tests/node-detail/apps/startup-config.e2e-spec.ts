import {browser, by, element, ExpectedConditions} from "protractor";
import {NodesListPage} from "../../../pages/nodes-list.po";
import {LoginPage} from "../../../pages/login";
import {NodeDetailPage} from "../../../pages/node-detail.po";
import {NODE_PUBLIC_KEY} from "../../../util/constants";

describe('Apps startup config', () => {
  let page: NodeDetailPage;

  beforeEach(() => {
    browser.restart();
    browser.waitForAngularEnabled(false);
    page = new NodeDetailPage();
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
