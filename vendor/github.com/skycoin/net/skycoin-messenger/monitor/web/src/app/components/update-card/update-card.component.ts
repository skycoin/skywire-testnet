import { Component, OnInit, ViewEncapsulation } from '@angular/core';
import { ApiService } from '../../service/api/api.service';

const NOUPGRADE = 'No Upgrade Available';
const UPGRADE = 'Upgrade Available';

@Component({
  selector: 'app-update-card',
  templateUrl: './update-card.component.html',
  styleUrls: ['./update-card.component.scss'],
  encapsulation: ViewEncapsulation.None
})
export class UpdateCardComponent implements OnInit {
  progressValue = 0;
  progressTask = null;
  updateStatus = NOUPGRADE;
  hasUpdate = false;
  constructor(private api: ApiService) { }

  ngOnInit() {
    this.api.checkUpdate('One', '0.0.1').subscribe((res: Update) => {
      this.hasUpdate = res.Update;
      if (this.hasUpdate) {
        this.updateStatus = UPGRADE;
      }
    });
  }
  startDownload(ev: Event) {
    ev.stopImmediatePropagation();
    ev.stopPropagation();
    ev.preventDefault();
    this.progressTask = setInterval(() => {
      this.progressValue += 1;
      if (this.progressValue >= 100) {
        clearInterval(this.progressTask);
      }
    }, 100);
  }
  getUpgradeStatus() {
    return false;
  }
}

export interface Update {
  Force?: boolean;
  Update?: boolean;
  Latest?: string;
}
