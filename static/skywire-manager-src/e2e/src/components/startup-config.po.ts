import {clickElement, fillKeyPair, findById, getKeyPair} from "../util/selection";
import {by, element} from "protractor";

export class StartupConfigDialog {

  toggleAutomaticStart() {
    clickElement(findById('toggleAutomaticStartBtn'));
  }

  save() {
    clickElement(findById('saveStartupConfigBtn'));
  }

  isAutomaticStartToggled() {
    return element(by.css('mat-dialog-content')).element(by.id('toggleAutomaticStartBtn')).getAttribute('ng-reflect-checked').then((attr) => {
      return attr !== 'false';
    });
  }

  fillKeyPair(nodeKey: string, appKey: string) {
    fillKeyPair(element(by.css('mat-dialog-content')), nodeKey, appKey);
  }

  getKeyPair() {
    return getKeyPair(element(by.css('mat-dialog-content')));
  }
}
