import { Component } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import {StorageService} from "./services/storage.service";
import {getLangs} from "./utils/languageUtils";

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent {
  constructor(
    private translate: TranslateService,
    private storage: StorageService
  )
  {
    translate.onDefaultLangChange.subscribe(({lang}) => storage.setDefaultLanguage(lang));
    translate.addLangs(getLangs());
    translate.use(storage.getDefaultLanguage());
  }
}
