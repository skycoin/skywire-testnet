import {PATHS} from '../../../src/app/app-routing.module';
import BasePage from './base-page.po';
import {findById} from '../util/selection';

export class LoginPage extends BasePage {

  path = PATHS.login;

  login() {
    return this.loginWithPassword('123123123');
  }

  badLogin() {
    return this.loginWithPassword('badpassword');
  }

  loginWithPassword(password: string) {
    findById('passwordInput').sendKeys(password);
    return findById('loginButton').click();
  }
}
