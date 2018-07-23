import { Component, Input } from '@angular/core';
import { NodeTransport } from '../../../../app.datatypes';

@Component({
  selector: 'app-transports',
  templateUrl: './transports.component.html',
  styleUrls: ['./transports.component.css']
})
export class TransportsComponent {
  @Input() transports: NodeTransport[];

  constructor() { }
}
