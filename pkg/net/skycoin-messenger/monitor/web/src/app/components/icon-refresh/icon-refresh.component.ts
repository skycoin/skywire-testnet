import { Component, OnInit, HostListener, ViewChild, ElementRef, Renderer } from '@angular/core';
import { MatIcon } from '@angular/material';

@Component({
  selector: 'app-refresh-icon',
  templateUrl: 'icon-refresh.component.html',
  styleUrls: ['./icon-refresh.component.scss']
})

export class IconRefreshComponent implements OnInit {
  @ViewChild('icon') icon: MatIcon;
  constructor(private render: Renderer) { }

  ngOnInit() { }

  stop() {
    this.render.setElementClass(this.icon._elementRef.nativeElement, 'refresh-process', false);
  }
  start() {
    this.render.setElementClass(this.icon._elementRef.nativeElement, 'refresh-process', true);
  }
  // @HostListener('click', ['$event'])
  // _click(ev: Event) {
  //   ev.stopImmediatePropagation();
  //   ev.stopPropagation();
  //   ev.preventDefault();
  //   this.start();
  // }
}
