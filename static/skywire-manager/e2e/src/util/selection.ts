import {browser, by, element, ElementFinder, ExpectedConditions} from "protractor";

function findById(id: string): ElementFinder
{
  return element(by.id(id));
}

function waitForVisibility(element: ElementFinder)
{
  return browser.wait(ExpectedConditions.visibilityOf(element));
}

export {
  findById,
  waitForVisibility
}
