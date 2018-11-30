import {browser, by, element, ExpectedConditions} from 'protractor';
import {NodesListPage} from '../pages/nodes-list.po';
import {NodeDetailPage} from '../pages/node-detail.po';
import {LoginPage} from '../pages/login';
import {snackBarContainsText} from '../util/selection';

describe('Login view', () => {
  let page: LoginPage;

  beforeEach(() => {
    // NodeService runs a timer, and it makes Protractor to wait forever. Make Protractor not wait.
    browser.restart();
    page = new LoginPage();
  });

  it('Login with bad password should display error snackbar', () => {

    page.navigateTo();

    // Wait until the table is rendered, that means the NodeService request has been received.
    page.badLogin();

    // This won't always be true, so skip it.
    // expect(page.getFirstNodeTooltip()).toEqual("Online: the node is correctly detected by the Skycoin network.");
    snackBarContainsText('Incorrect password');
  });

  it('Login with correct password should bring to nodes list view', () => {

    browser.waitForAngularEnabled(false);

    page.navigateTo();

    page.login();

    expect(new NodesListPage().isVisible()).toBeTruthy();
  });
});
