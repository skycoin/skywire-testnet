import { Component } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import {StorageService} from "./services/storage.service";

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
    translate.addLangs(['en', 'es']);
    translate.use(storage.getDefaultLanguage());
  }
}
