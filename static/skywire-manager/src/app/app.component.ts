import { Component } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Router } from '@angular/router';
import { Location } from '@angular/common';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  showFooter = false;

  constructor(
    private translate: TranslateService,
    private location: Location,
    private router: Router,
  ) {
    translate.setDefaultLang('en');
    translate.use('en');

    router.events.subscribe(() => {
      this.showFooter = !location.isCurrentPathEqualTo('/login');
    });
  }
}
