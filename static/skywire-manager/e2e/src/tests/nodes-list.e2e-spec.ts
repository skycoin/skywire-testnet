import {browser, by, element, ExpectedConditions} from "protractor";
import {NodesListPage} from "../pages/nodes-list.po";
import {NodeDetailPage} from "../pages/node-detail.po";

describe('Nodelist view', () => {
  let page: NodesListPage;

  beforeEach(() => {
    // NodeService runs a timer, and it makes Protractor to wait forever. Make Protractor not wait.
    browser.restart();
    page = new NodesListPage();
  });

  it('List should display 1 node', () => {

    browser.waitForAngularEnabled(false);
    page.navigateTo();

    // Wait until the table is rendered, that means the NodeService request has been received.
    expect(page.getTableRowsCount()).toEqual(2);
    expect(page.getFirstNodeIndex()).toEqual("1");
    expect(page.getFirstNodeLabel()).toEqual("127.0.0.1");
    expect(page.getFirstNodeKey()).toEqual("03f407f33e6fdbb3cec4b7b99dd338245e5272008f619f445402be21add0c7ac7e");
    expect(page.getFirstNodeTooltip()).toEqual("Online: the node is correctly detected by the Skycoin network.");
  });

  it('Click node should bring to node detail view', () => {

    browser.waitForAngularEnabled(false);
    page.navigateTo();

    page.clickFirstNode();

    expect(new NodeDetailPage().isVisible()).toBeTruthy();
  });
});
