import { Component } from '@angular/core';
import { Router } from '@angular/router';

@Component({
  selector: 'footer-component',
  templateUrl: './footer.component.html',
  styleUrls: ['./footer.component.scss']
})
export class FooterComponent {
  constructor(
    private router: Router,
  ) { }

  settings() {
    this.router.navigate(['settings']);
  }
}
