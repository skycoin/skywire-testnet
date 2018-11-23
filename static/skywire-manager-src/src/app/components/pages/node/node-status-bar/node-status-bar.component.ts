import {Component, Input, OnChanges, OnInit} from '@angular/core';
import {NodeData} from '../../../../app.datatypes';
import {TranslateService} from '@ngx-translate/core';
import {isDiscovered} from '../../../../utils/nodeUtils';

@Component({
  selector: 'app-node-status-bar',
  templateUrl: './node-status-bar.component.html',
  styleUrls: ['./node-status-bar.component.scss']
})
export class NodeStatusBarComponent implements OnInit, OnChanges {
  @Input() nodeData: NodeData;
  onlineTooltip: string | any;

  constructor(private translate: TranslateService) { }

  ngOnInit() {
    this.getOnlineTooltip();
  }

  ngOnChanges(): void {
    this.getOnlineTooltip();
  }

  get isDiscovered(): boolean {
    return isDiscovered(this.nodeData.info);
  }

  getOnlineTooltip(): void {
    this.translate.get(this.isDiscovered ? 'node.statuses.discovered-tooltip' : 'node.statuses.online-tooltip')
      .subscribe((text: string) => this.onlineTooltip = text);
  }
}
