import { Component, OnInit, ViewEncapsulation } from '@angular/core';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { ApiService } from '../../service';
import { Router } from '@angular/router';
import swal from 'sweetalert2';
import { error } from 'selenium-webdriver';

@Component({
  selector: 'app-updatepass-page',
  templateUrl: 'update-pass.component.html',
  styleUrls: ['./update-pass.component.scss'],
  encapsulation: ViewEncapsulation.None
})

export class UpdatePassComponent implements OnInit {
  updateForm = new FormGroup({
    oldpass: new FormControl('', [Validators.required, Validators.minLength(4), Validators.maxLength(20)]),
    newpass: new FormControl('', [Validators.required, Validators.minLength(4), Validators.maxLength(20)]),
  });
  status = 0;
  serverHint: any = 'Please confirm that the original password is correct and then try to update again.';
  constructor(private api: ApiService, private router: Router) { }

  ngOnInit() {
    this.api.checkLogin().subscribe(result => {
    });
  }
  init() {
    this.status = 0;
  }
  update(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    const data = new FormData();
    data.append('oldPass', this.updateForm.get('oldpass').value);
    data.append('newPass', this.updateForm.get('newpass').value);
    this.api.updatePass(data).subscribe(result => {
      if (result) {
        swal({
          title: 'Warning',
          text: 'The password has been changed. Click the button to jump to the login page.',
          type: 'warning',
          allowOutsideClick: false,
          allowEscapeKey: false
        }).then(() => {
          this.router.navigate([{ outlets: { user: ['login'] } }]);
        });
      } else {
        this.status = 1;
        this.serverHint = result;
      }
    }, err => {
      this.status = 1;
      this.serverHint = err;
    });
  }
}
