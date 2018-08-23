import {Component, OnInit} from '@angular/core';
import {AuthService} from '../../../services/auth.service';
import {FormControl, FormGroup, Validators} from '@angular/forms';
import {Router} from '@angular/router';
import {MatSnackBar} from "@angular/material";
import {TranslateService} from "@ngx-translate/core";
import {ErrorsnackbarService} from "../../../services/errorsnackbar.service";

@Component({
  selector: 'app-login',
  templateUrl: './login.component.html',
  styleUrls: ['./login.component.scss']
})
export class LoginComponent implements OnInit {
  form: FormGroup;
  error: string;

  constructor(
    private authService: AuthService,
    private translate: TranslateService,
    private router: Router,
    private snackbar: ErrorsnackbarService,
  ) {
  }

  ngOnInit() {
    this.form = new FormGroup({
      'password': new FormControl('', Validators.required),
    });
  }

  setError(error: string)
  {
    this.snackbar.open(error, null);
  }

  login()
  {
    this.authService.login(this.form.get('password').value).subscribe(
      () => this.router.navigate(['nodes']),
      () => this.setError(this.translate.instant('login.incorrect-password')),
    );
  }
}
