import {PATHS} from "../../../src/app/app-routing.module";
import BasePage from "./base-page.po";
import {findById, waitForVisibility} from "../util/selection";

export class NodeDetailPage extends BasePage {

  path = PATHS.nodeDetail;

  getContainer() {
    return findById('nodeDetailView');
  }

  isVisible() {
    waitForVisibility(this.getContainer());
    return this.getContainer().isDisplayed();
  }
}
