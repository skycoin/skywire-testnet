import {Component, Input, OnInit} from '@angular/core';

@Component({
  selector: 'app-skycoin-logo',
  templateUrl: './skycoin-logo.component.html',
  styleUrls: ['./skycoin-logo.component.scss'],
})
export class SkycoinLogoComponent implements OnInit {
  @Input() direction: 'vertical' | 'horizontal' = 'horizontal';
  logoPath: string;

  constructor() { }

  get isVertical() {
    return this.direction === 'vertical';
  }

  get logoName(): string {
    let logoName = '';
    if (this.isVertical) {
      logoName = 'logo-vert';
    } else {
      logoName = 'logo-horiz';
    }
    return logoName;
  }

  ngOnInit() {
    this.logoPath = `/assets/img/${this.logoName}.png`;
  }
}
