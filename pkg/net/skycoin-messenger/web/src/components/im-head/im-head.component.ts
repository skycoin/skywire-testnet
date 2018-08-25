import { Component, OnInit, ViewEncapsulation, Input, HostListener } from '@angular/core';
import { HeadColorMatch, SocketService } from '../../providers';
import { MdDialog } from '@angular/material';
import { ImInfoDialogComponent } from '../im-info-dialog/im-info-dialog.component';
import * as jdenticon from 'jdenticon';
import { DomSanitizer } from '@angular/platform-browser';

@Component({
  selector: 'app-im-head',
  templateUrl: './im-head.component.html',
  styleUrls: ['./im-head.component.scss'],
  encapsulation: ViewEncapsulation.None
})
export class ImHeadComponent implements OnInit {
  @Input() key = '';
  @Input() canClick = true;
  name = '';
  img;
  icon: HeadColorMatch = { bg: '#fff', color: '#000' };
  constructor(private socket: SocketService, private dialog: MdDialog, private _domsanitizer: DomSanitizer) { }

  ngOnInit() {
    if (this.key === '') {
      this.name = '?';
    } else {
      if (this.socket.userInfo.get(this.key) !== undefined) {
        const icon = this.socket.userInfo.get(this.key).Icon;
        if (icon !== undefined) {
          this.icon = icon;
        }
      }
      this.img = this._domsanitizer.bypassSecurityTrustHtml(jdenticon.toSvg(this.key, 52));
    }

  }

  @HostListener('click', ['$event'])
  _click(ev: Event) {
    if (!this.canClick) {
      return;
    }
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    const ref = this.dialog.open(ImInfoDialogComponent, {
      position: { top: '10%' },
      // panelClass: 'alert-dialog-panel',
      backdropClass: 'alert-backdrop',
      width: '23rem'
    });
    ref.componentInstance.key = this.key;
  }
}

