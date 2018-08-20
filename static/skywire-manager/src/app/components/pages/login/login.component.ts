import {Component, OnInit} from '@angular/core';
import {AuthService} from '../../../services/auth.service';
import {FormControl, FormGroup, Validators} from '@angular/forms';
import {Router} from '@angular/router';
import {catchError} from 'rxjs/operators';

@Component({
  selector: 'app-login',
  templateUrl: './login.component.html',
  styleUrls: ['./login.component.scss']
})
export class LoginComponent implements OnInit {
  form: FormGroup;
  error: string;
  errorDelay = 3000;

  constructor(
    private authService: AuthService,
    private router: Router
  ) {
  }

  ngOnInit() {
    this.form = new FormGroup({
      'password': new FormControl('', Validators.required),
    });
  }

  setError(error: string) {
    this.error = error;
    setTimeout(() => {
      this.error = '';
    }, this.errorDelay);
  }

  login() {
    if (this.form.valid) {
      this.authService.login(this.form.get('password').value).subscribe(
        () => this.router.navigate(['nodes']),
        () => this.setError('login.incorrect-password'),
      );
    }
  }
}
