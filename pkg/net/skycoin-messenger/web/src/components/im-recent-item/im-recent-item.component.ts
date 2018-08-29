import {
  Component,
  Input,
  ViewEncapsulation,
  HostListener,
  HostBinding, Output,
  EventEmitter,
  OnInit
} from '@angular/core';
import { RecentItem, HeadColorMatch, EmojiService } from '../../providers';

@Component({
  selector: 'app-im-recent-item',
  templateUrl: './im-recent-item.component.html',
  styleUrls: ['./im-recent-item.component.scss'],
  encapsulation: ViewEncapsulation.None
})
export class ImRecentItemComponent implements OnInit {
  @Input() info: RecentItem;
  @HostBinding('class.active') active = false;
  @Output('onClick') onClick: EventEmitter<ImRecentItemComponent> = new EventEmitter();
  constructor(private emoji: EmojiService) { }
  ngOnInit() {
    this.info.last = this.emoji.toImage(this.info.last);
  }
  @HostListener('click', ['$event'])
  _click(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    this.onClick.emit(this);
    this.active = !this.active;
  }
}
