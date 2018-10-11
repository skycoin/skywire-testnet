import {browser, by, element, ElementFinder, ExpectedConditions} from "protractor";

export function findById(id: string): ElementFinder {
  return element(by.id(id));
}

export function waitForVisibility(element: ElementFinder) {
  return browser.wait(ExpectedConditions.visibilityOf(element));
}

export function waitForInvisibility(element: ElementFinder) {
  return browser.wait(ExpectedConditions.invisibilityOf(element));
}

export function clickElement(element: ElementFinder) {
  this.waitForVisibility(element);
  return element.click();
}

export function fillKeyPair(parentElement: ElementFinder, nodeKey: string, appKey: string) {
  parentElement.element(by.id('nodeKeyField')).element(by.tagName('input')).sendKeys(nodeKey);
  parentElement.element(by.id('appKeyField')).element(by.tagName('input')).sendKeys(appKey);
}

export function getKeyPair(parentElement: ElementFinder) {
  return parentElement.element(by.id('nodeKeyField')).element(by.tagName('input')).getText().then((nodeKey) => {
    parentElement.element(by.id('appKeyField')).element(by.tagName('input')).getText().then((appKey) => {
      return {nodeKey, appKey}
    });
  });
}

export function snackBarContainsText(text) {
  return element(by.cssContainingText('.mat-simple-snackbar', text)).isPresent();
}
