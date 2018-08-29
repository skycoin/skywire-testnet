import {browser, by, element, ExpectedConditions} from "protractor";
import {NodesListPage} from "./pages/nodes-list.po";
import {LoginPage} from "./pages/login";

describe('NodeDetail view', () => {
  let page: NodesListPage;

  beforeEach(() => {
    page = new NodesListPage();
  });

  afterEach(() => {
    browser.restart();
  });
});
