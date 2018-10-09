import { Component, ViewEncapsulation, Input, ViewChild, OnInit } from '@angular/core';
import { ImHistoryMessage, SocketService, EmojiService } from '../../providers';
import { MdMenuTrigger } from '@angular/material';

@Component({
  selector: 'app-im-history-message',
  templateUrl: './im-history-message.component.html',
  styleUrls: ['./im-history-message.component.scss'],
  encapsulation: ViewEncapsulation.None
})
export class ImHistoryMessageComponent implements OnInit {
  selfId = '';
  @ViewChild(MdMenuTrigger) contextMenu: MdMenuTrigger;
  @Input() index: number;
  @Input() chat: ImHistoryMessage = null;
  constructor(private socket: SocketService, private emoji: EmojiService) {
    this.selfId = this.socket.key;
  }
  ngOnInit() {
    this.chat.Msg = this.emoji.toImage(this.chat.Msg);
  }
  rightClick(ev: Event) {
    // ev.preventDefault();
    // this.contextMenu.openMenu();
  }
}
