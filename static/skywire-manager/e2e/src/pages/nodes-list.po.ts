import {PATHS} from "../../../src/app/app-routing.module";
import BasePage from "./base-page.po";
import {findById, waitForVisibility} from "../util/selection";
import {by, element} from "protractor";
import {LoginPage} from "./login";
import {APP_SOCKSC, APP_SSHC, APP_SSHS} from "../util/constants";

export class NodesListPage extends BasePage {

  path = PATHS.nodes;
  private ROW_INDEX = "nodeIndex";
  private ROW_LABEL = "nodeLabel";
  private ROW_KEY = "nodeKey";
  private ROW_STATUS_ONLINE = "nodeStatusOnline";

  private getNodesList() {
    return findById("nodeListTable").element(by.tagName('tbody'));
  }

  navigateTo() {
    let result = new LoginPage().navigateTo();
    new LoginPage().login();
    return result;
  }

  waitNodesTablesToBeLoaded() {
    waitForVisibility(this.getNodesList());
  }

  private getTableRows() {
    return this.getNodesList().all(by.tagName('tr'));
  }

  getNodesListCount() {
    this.waitNodesTablesToBeLoaded();
    return this.getTableRows().count();
  }

  getFirstNodeIndex() {
    this.waitNodesTablesToBeLoaded();
    return this.getFirstNodeField(this.ROW_INDEX).getText();
  }

  getFirstNodeField(field: string) {
    return this.getFirstRow().element(by.id(field));
  }

  private getFirstRow() {
    return this.getTableRows().first();
  }

  getFirstNodeLabel() {
    this.waitNodesTablesToBeLoaded();
    return this.getFirstNodeField(this.ROW_LABEL).getText();
  }

  getFirstNodeKey() {
    this.waitNodesTablesToBeLoaded();
    return this.getFirstNodeField(this.ROW_KEY).getText();
  }

  getFirstNodeTooltip() {
    this.waitNodesTablesToBeLoaded();
    waitForVisibility(this.getFirstNodeField(this.ROW_STATUS_ONLINE));
    return this.getFirstNodeField(this.ROW_STATUS_ONLINE).getAttribute("title");
  }

  clickFirstNode() {
    this.waitNodesTablesToBeLoaded();
    this.getFirstRow().click();
  }

  isVisible() {
    this.waitNodesTablesToBeLoaded();
    return this.getNodesList().isDisplayed();
  }
}
