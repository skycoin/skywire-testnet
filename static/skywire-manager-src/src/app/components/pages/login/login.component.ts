import {Component, OnInit} from '@angular/core';
import {AuthService} from '../../../services/auth.service';
import {FormControl, FormGroup, Validators} from '@angular/forms';
import {Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {ErrorsnackbarService} from '../../../services/errorsnackbar.service';

@Component({
  selector: 'app-login',
  templateUrl: './login.component.html',
  styleUrls: ['./login.component.scss']
})
export class LoginComponent implements OnInit {
  form: FormGroup;
  loading = false;

  constructor(
    private authService: AuthService,
    private translate: TranslateService,
    private router: Router,
    private snackbar: ErrorsnackbarService,
  ) { }

  ngOnInit() {
    this.form = new FormGroup({
      'password': new FormControl('', Validators.required),
    });
  }

  setError(error: string) {
    this.snackbar.open(error, null);
  }

  onLoginSuccess() {
    this.router.navigate(['nodes']);
  }

  onLoginError() {
    this.loading = false;
    this.setError(this.translate.instant('login.incorrect-password'));
  }

  login() {
    if (!this.form.valid) {
      return;
    }

    this.loading = true;
    this.authService.login(this.form.get('password').value).subscribe(
      () => this.onLoginSuccess(),
      () => this.onLoginError()
    );
  }
}
