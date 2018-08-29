import {PATHS} from "../../../src/app/app-routing.module";
import BasePage from "./base-page.po";
import {findById} from "../util/selection";
import {browser, ExpectedConditions} from "protractor";

export class NodeDetailPage extends BasePage {

  path = PATHS.nodeDetail;

  getContainer() {
    return findById('nodeDetailView');
  }

  isVisible() {
    browser.wait(ExpectedConditions.visibilityOf(this.getContainer()));
    return this.getContainer().isDisplayed();
  }
}
