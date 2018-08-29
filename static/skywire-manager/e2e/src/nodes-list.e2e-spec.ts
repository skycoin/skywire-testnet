import {browser, by, element, ExpectedConditions} from "protractor";
import {NodesListPage} from "./pages/nodes-list.po";
import {LoginPage} from "./pages/login";

describe('Skyware Manager App', () => {
  let page: NodesListPage;

  beforeEach(() => {
    page = new NodesListPage();
  });

  it('List should display 1 node', () => {

    page.navigateTo();

    new LoginPage().login();

    // NodeService runs a timer, and it makes Protractor to wait forever. Make Protractor not wait.
    browser.waitForAngularEnabled(false);

    // Wait until the table is rendered, that means the NodeService request has been received.
    page.getTableRowsCount().then((count) => expect(count).toEqual(2));

    expect<any>(page.getFirstNodeIndex()).toEqual("1");
    expect<any>(page.getFirstNodeLabel()).toEqual("127.0.0.1");
    expect<any>(page.getFirstNodeKey()).toEqual("03f407f33e6fdbb3cec4b7b99dd338245e5272008f619f445402be21add0c7ac7e");
  });
});
