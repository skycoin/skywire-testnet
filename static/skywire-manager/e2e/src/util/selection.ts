import {by, element, ElementFinder} from "protractor";

function findById(id: string): ElementFinder
{
  return element(by.id(id));
}

export {
  findById
}
