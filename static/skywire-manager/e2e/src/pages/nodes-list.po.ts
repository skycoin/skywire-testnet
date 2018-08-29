import {PATHS} from "../../../src/app/app-routing.module";
import BasePage from "./base-page.po";
import {findById} from "../util/selection";
import {browser, by, ExpectedConditions} from "protractor";

export class NodesListPage extends BasePage {

  path = PATHS.nodes;
  private NODES_TABLE_ID = "nodeListTable";
  private ROW_INDEX = "nodeIndex";
  private ROW_LABEL = "nodeLabel";
  private ROW_KEY = "nodeKey";

  private getNodesTable() {
    return findById(this.NODES_TABLE_ID);
  }

  waitNodesTablesToBeLoaded() {
    browser.wait(ExpectedConditions.visibilityOf(this.getNodesTable()));
  }

  private getTableRows() {
    return this.getNodesTable().all(by.tagName('tr'));
  }

  getTableRowsCount() {
    this.waitNodesTablesToBeLoaded();
    return this.getTableRows().count();
  }

  getFirstNodeIndex() {
    this.waitNodesTablesToBeLoaded();
    return this.getFirstNodeValue(this.ROW_INDEX);
  }

  getFirstNodeValue(field: string) {
    return this.getFirstRow().element(by.id(field)).getText();
  }

  private getFirstRow() {
    return this.getTableRows().get(1);
  }

  getFirstNodeLabel() {
    return this.getFirstNodeValue(this.ROW_LABEL);
  }

  getFirstNodeKey() {
    return this.getFirstNodeValue(this.ROW_KEY);
  }
}
