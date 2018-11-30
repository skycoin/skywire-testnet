import {browser} from 'protractor';

export default abstract class BasePage {

  protected path: string;

  navigateTo() {
    return browser.get(`#/${this.path}`);
  }
}
