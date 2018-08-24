import { Component, OnInit } from '@angular/core';
import { FormControl, FormGroup } from '@angular/forms';
import { Router } from '@angular/router';
import { Location } from '@angular/common';
import { TranslateService } from '@ngx-translate/core';
import {StorageService} from '../../../services/storage.service';
import {getNativeName} from '../../../utils/languageUtils';

interface LangOption {
  id: string;
  name: string;
}

@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.scss']
})
export class SettingsComponent implements OnInit {
  form: FormGroup;
  readonly timesList = ['3', '4', '5', '10', '15', '20', '30', '60'];
  langList: LangOption[] = [];

  currentLang: string;
  currentRefreshRate: string;

  constructor(
    private router: Router,
    private location: Location,
    private translate: TranslateService,
    private storage: StorageService,
  ) {
    this.buildLangOptions();
  }

  ngOnInit() {
    const currentLang = this.storage.getDefaultLanguage();
    this.currentRefreshRate = this.storage.getRefreshTime().toString();

    this.form = new FormGroup({
      'refreshRate': new FormControl(this.currentRefreshRate || '5'),
      'language': new FormControl(currentLang),
    });

    this.form.valueChanges.subscribe(({refreshRate, language}) => {
      this.changeRefreshRate(refreshRate);
      this.changeLanguage(language);
    });
  }

  password() {
    this.router.navigate(['settings/password']);
  }

  changeLanguage(lang: string) {
    this.translate.use(lang);
    this.translate.setDefaultLang(lang);
  }

  back() {
    this.location.back();
  }

  private changeRefreshRate(refreshRate: number): void {
    this.storage.setRefreshTime(refreshRate);
  }

  private buildLangOptions() {
    const langCodes = this.translate.getLangs();
    langCodes.forEach((code) => {
      this.langList.push({
        id: code,
        name: getNativeName(code)
      });
    });
  }

  onChangePasswordClicked() {
    this.password();
  }
}
