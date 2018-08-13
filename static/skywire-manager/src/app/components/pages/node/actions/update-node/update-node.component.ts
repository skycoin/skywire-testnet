import { Component, OnInit } from '@angular/core';
import {NodeService} from "../../../../../services/node.service";

@Component({
  selector: 'app-update-node',
  templateUrl: './update-node.component.html',
  styleUrls: ['./update-node.component.css']
})
export class UpdateNodeComponent implements OnInit
{
  constructor(private nodeService: NodeService) { }

  isLoading: boolean = false;
  isUpdateAvailable: boolean = false;

  ngOnInit()
  {
    this.fetchUpdate();
  }

  private fetchUpdate()
  {
    this.isLoading = true;
    this.nodeService.checkUpdate().subscribe(this.onFetchUpdateSuccess.bind(this), this.onFetchUpdateError.bind(this));
  }

  private onFetchUpdateSuccess(updateAvailable: boolean)
  {
    this.isLoading = false;
    this.isUpdateAvailable = true; //updateAvailable;
  }

  private onFetchUpdateError(e)
  {
    this.isLoading = false;
    console.warn('check update problem', e)
  }

  onUpdateClicked($event)
  {
    this.isLoading = true;
    this.nodeService.update().subscribe(this.onUpdateSuccess.bind(this), this.onUpdateError.bind(this));
  }

  onUpdateSuccess(updated: boolean)
  {
    this.isLoading = false;
  }

  onUpdateError(e)
  {
    this.isLoading = false;
  }
}
