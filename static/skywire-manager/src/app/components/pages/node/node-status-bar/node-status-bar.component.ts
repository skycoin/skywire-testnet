import {Component, Input, OnChanges, OnInit} from '@angular/core';
import {NodeData} from "../../../../app.datatypes";
import {TranslateService} from "@ngx-translate/core";
import {isOnline} from "../../../../utils/nodeUtils";

@Component({
  selector: 'app-node-status-bar',
  templateUrl: './node-status-bar.component.html',
  styleUrls: ['./node-status-bar.component.scss']
})
export class NodeStatusBarComponent implements OnInit, OnChanges
{
  @Input() nodeData: NodeData;
  onlineTooltip: string | any;

  constructor(private translate: TranslateService) { }

  ngOnInit()
  {
    this.getOnlineTooltip();
  }

  ngOnChanges(): void
  {
    this.getOnlineTooltip();
  }

  getOnlineTooltip(): void
  {
    let key;
    if (isOnline(this.nodeData.info))
    {
      key = 'node.online-tooltip';
    }
    else
    {
      key = 'node.offline-tooltip';
    }
    this.translate.get(key).subscribe((text) => this.onlineTooltip = text);
  }
}
