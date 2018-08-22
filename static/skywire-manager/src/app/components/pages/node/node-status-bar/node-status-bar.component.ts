import {Component, Input, OnChanges, OnInit} from '@angular/core';
import {NodeData} from "../../../../app.datatypes";
import {TranslateService} from "@ngx-translate/core";

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
    if (this.isOnline)
    {
      key = 'node.online-tooltip';
    }
    else
    {
      key = 'node.offline-tooltip';
    }
    this.translate.get(key).subscribe((text) => this.onlineTooltip = text);
  }

  /**
   * Node is online if at least one discovery is seeing it.
   */
  private get isOnline()
  {
    let isOnline = false;
    Object.keys(this.nodeData.info.discoveries).map((discovery) =>
    {
      isOnline = isOnline || this.nodeData.info.discoveries[discovery];
    });
    return isOnline;
  }
}
