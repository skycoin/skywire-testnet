import { Component } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import {StorageService} from './services/storage.service';
import {getLangs} from './utils/languageUtils';
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
    private storage: StorageService,
    private location: Location,
    private router: Router,
  ) {
    translate.addLangs(getLangs());
    translate.use(storage.getDefaultLanguage());
    translate.onDefaultLangChange.subscribe(({lang}) => storage.setDefaultLanguage(lang));

    router.events.subscribe(() => {
      this.showFooter = !location.isCurrentPathEqualTo('/login');
    });
  }
}
