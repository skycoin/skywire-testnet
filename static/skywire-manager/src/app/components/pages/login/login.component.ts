import { Component, OnInit } from '@angular/core';
import { AuthService } from '../../../services/auth.service';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';

@Component({
  selector: 'app-login',
  templateUrl: './login.component.html',
  styleUrls: ['./login.component.css']
})
export class LoginComponent implements OnInit {
  form: FormGroup;
  error: string;

  constructor(
    private authService: AuthService,
    private router: Router,
  ) { }

  ngOnInit() {
    this.form = new FormGroup({
      'password': new FormControl('', Validators.required),
    });
  }

  login() {
    if (this.form.valid) {
      this.authService.login(this.form.get('password').value).subscribe(status => {
        if (status) {
          this.router.navigate(['nodes']);
        } else {
          this.error = 'Incorrect password';
        }
      });
    }
  }
}
