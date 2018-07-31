import { Component, OnInit } from '@angular/core';
import { FormControl, FormGroup } from '@angular/forms';
import { Router } from '@angular/router';

@Component({
  selector: 'app-settings',
  templateUrl: './settings.component.html',
  styleUrls: ['./settings.component.css']
})
export class SettingsComponent implements OnInit {
  form: FormGroup;

  constructor(
    private router: Router,
  ) { }

  ngOnInit() {
    this.form = new FormGroup({
      'refreshRate': new FormControl(''),
    });

    this.form.valueChanges.subscribe(value => {
      console.log(value.refreshRate);
    });
  }

  password() {
    this.router.navigate(['settings/password']);
  }
}
