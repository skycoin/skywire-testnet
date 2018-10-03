import { Component, OnInit, ViewEncapsulation, Input, ViewChildren, QueryList, Output, EventEmitter } from '@angular/core';
import { ImRecentItemComponent } from '../im-recent-item/im-recent-item.component';
import { SocketService, UserService, ModalService } from '../../providers';
// import { ToolService } from '../../providers/tool/tool.service';
import { MdDialog } from '@angular/material';
import { CreateChatDialogComponent } from '../create-chat-dialog/create-chat-dialog.component';
import { PerfectScrollbarConfigInterface } from 'ngx-perfect-scrollbar';
import { ImInfoDialogComponent } from '../im-info-dialog/im-info-dialog.component';

@Component({
  selector: 'app-im-recent-bar',
  templateUrl: './im-recent-bar.component.html',
  styleUrls: ['./im-recent-bar.component.scss'],
  encapsulation: ViewEncapsulation.None
})
export class ImRecentBarComponent implements OnInit {
  config: PerfectScrollbarConfigInterface = {};
  chatting = '';
  @ViewChildren(ImRecentItemComponent) items: QueryList<ImRecentItemComponent>;
  @Input() list = [];
  constructor(
    private socket: SocketService,
    private user: UserService,
    private dialog: MdDialog,
    // private tool: ToolService,
    private modal: ModalService) { }

  ngOnInit() {
  }
  Test(content: any) {
    // this.modal.open(content);
    const body = document.querySelector('body');
    const input = document.createElement('input');
    const img = new Image();
    input.type = 'file';
    input.name = 'file';
    input.addEventListener('change', (ev) => {
      const read: FileReader = new FileReader();
      read.onload = (event) => {
        console.log('ev:', event.target['result']);
        const blob = new Blob([event.target['result']], { type: 'image/jpeg' });
        img.src = URL.createObjectURL(blob);
        body.appendChild(img);
      }
      read.readAsArrayBuffer(ev.target['files'][0]);
      // img.src = re
    })
    input.click();
  }
  selectItem(item: ImRecentItemComponent) {
    if (item.active) {
      this.chatting = '';
      this.socket.chattingUser = '';
      return;
    }
    item.info.unRead = 0;
    this.chatting = item.info.name;
    this.socket.chattingUser = item.info.name;
    const tmp = this.items.filter((el) => {
      return el.info.name !== item.info.name;
    });
    tmp.forEach(el => {
      el.active = false;
    });
  }

  openCreate(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    const def = this.dialog.open(CreateChatDialogComponent, { position: { top: '10%' }, width: '350px' });
    def.afterClosed().subscribe(key => {
      if (key !== '' && key) {
        key = key.trim()
        this.items.forEach(el => {
          el.active = false;
        })
        const icon = this.user.getRandomMatch();
        this.socket.recent_list.push({ name: key, last: '', icon: icon });
        this.chatting = key;
        this.socket.chattingUser = key;
        setTimeout(() => {
          this.items.last.active = true;
        }, 10);
        this.socket.userInfo.set(key, { Icon: icon })
      }
    })
  }

  info(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    if (this.socket.key === '') {
      // this.tool.ShowDangerAlert('Faild', 'The server failed to get the key failed');
      return;
    }
    const input = document.createElement('input');
    document.body.appendChild(input);
    // tslint:disable-next-line:no-unused-expression
    input['value'] = this.socket.key;
    input.select();
    const successful = document.execCommand('copy');
    input.remove();
    const ref = this.dialog.open(ImInfoDialogComponent, {
      position: { top: '10%' },
      // panelClass: 'alert-dialog-panel',
      backdropClass: 'alert-backdrop',
      width: '23rem'
    });
    ref.componentInstance.key = this.socket.key;
    ref.componentInstance.hint = successful;
  }
}
