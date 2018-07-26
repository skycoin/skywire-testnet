import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';

@Component({
  selector: 'app-node-app-button',
  templateUrl: './node-app-button.component.html',
  styleUrls: ['./node-app-button.component.scss']
})
export class NodeAppButtonComponent implements OnInit
{
  @Input() title: string;
  @Input() icon: string;
  @Input() subtitle: string;
  @Input() active: boolean = false;
  @Input() hasMessages: boolean = false;
  @Input() showMore: boolean = false;
  @Output() onClickMore: EventEmitter<any> = new EventEmitter();
  @Output() onClick: EventEmitter<any> = new EventEmitter();
  private containerClass: string;

  constructor() { }

  handleClick(): void
  {
    this.onClick.emit();
  }

  handleClickMore(): void
  {
    this.onClickMore.emit();
  }

  ngOnInit()
  {
    this.containerClass =
      `${"d-flex flex-column align-items-center justify-content-center w-100"} ${this.active ? 'active' : ''}`
  }
}
