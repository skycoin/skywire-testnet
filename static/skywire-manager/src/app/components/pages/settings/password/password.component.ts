import { Component, OnInit } from '@angular/core';
import { AbstractControl, FormControl, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../../../services/auth.service';
import { Location } from '@angular/common';
import { MatSnackBar } from '@angular/material';

@Component({
  selector: 'app-password',
  templateUrl: './password.component.html',
  styleUrls: ['./password.component.css']
})
export class PasswordComponent implements OnInit {
  form: FormGroup;

  constructor(
    private authService: AuthService,
    private router: Router,
    private location: Location,
    private snackbar: MatSnackBar,
  ) { }

  ngOnInit() {
    this.form = new FormGroup({
      'oldPassword': new FormControl('', Validators.required),
      'newPassword': new FormControl('', Validators.compose([Validators.required, Validators.minLength(4), Validators.maxLength(20)])),
      'newPasswordConfirmation': new FormControl('', [this.validatePasswords.bind(this)]),
    }, {
      validators: [this.validatePasswords.bind(this)],
    });
  }

  changePassword() {
    if (this.form.valid) {
      this.authService.changePassword(this.form.get('oldPassword').value, this.form.get('newPassword').value)
        .subscribe(
          () => {
            this.router.navigate(['login']);
            this.snackbar.open('Log in with your new password');
          },
          (err) =>
          {
            this.snackbar.open(err.message);
          },
        );
    }
  }

  back() {
    this.location.back();
  }

  private validatePasswords(control: AbstractControl)
  {
    if (this.form)
    {
      return this.form.get('newPassword').value !== this.form.get('newPasswordConfirmation').value
        ? { invalid: true } : null;
    }
    else
    {
      return null;
    }
  }
}
