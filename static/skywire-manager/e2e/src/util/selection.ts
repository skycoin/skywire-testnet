import {browser, by, element, ElementFinder, ExpectedConditions} from "protractor";

export function findById(id: string): ElementFinder
{
  return element(by.id(id));
}

export function waitForVisibility(element: ElementFinder)
{
  return browser.wait(ExpectedConditions.visibilityOf(element));
}

export function waitForInvisibility(element: ElementFinder)
{
  return browser.wait(ExpectedConditions.invisibilityOf(element));
}
