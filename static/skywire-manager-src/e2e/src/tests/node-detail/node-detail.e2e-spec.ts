import {browser, by, element, ExpectedConditions} from "protractor";
import {NodesListPage} from "../../pages/nodes-list.po";
import {LoginPage} from "../../pages/login";
import {NodeDetailPage} from "../../pages/node-detail.po";
import {NODE_PUBLIC_KEY} from "../../util/constants";

describe('NodeDetail', () => {
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
});
