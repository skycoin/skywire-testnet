import { Component, OnInit, ViewEncapsulation } from '@angular/core';
import { ApiService } from '../../service';
import { Router } from '@angular/router';
import { FormControl, FormGroup, Validators } from '@angular/forms';

@Component({
  selector: 'app-login-page',
  templateUrl: 'login.component.html',
  styleUrls: ['./login.component.scss'],
  encapsulation: ViewEncapsulation.None
})

export class LoginComponent implements OnInit {
  loginForm = new FormGroup({
    pass: new FormControl('', [Validators.required, Validators.minLength(4), Validators.maxLength(20)]),
  });
  status = 0;
  constructor(private api: ApiService, private router: Router) { }

  ngOnInit() {
  }
  init() {
    this.status = 0;
  }
  login(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    if (this.loginForm.valid) {
      const data = new FormData();
      data.append('pass', this.loginForm.get('pass').value);
      this.api.login(data).subscribe(result => {
        if (result) {
          this.router.navigate(['']);
        }
      }, err => {
        this.status = 1;
      });
    }
  }
}
