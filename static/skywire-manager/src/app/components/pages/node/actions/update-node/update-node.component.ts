import { Component, OnInit } from '@angular/core';
import {NodeService} from "../../../../../services/node.service";
import {MatDialogRef} from "@angular/material";

@Component({
  selector: 'app-update-node',
  templateUrl: './update-node.component.html',
  styleUrls: ['./update-node.component.css']
})
export class UpdateNodeComponent implements OnInit
{
  updateError: string = null;
  constructor(
    private nodeService: NodeService,
    private dialogRef: MatDialogRef<UpdateNodeComponent>,
  ) { }

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
    this.isUpdateAvailable = updateAvailable;
  }

  private onFetchUpdateError(e)
  {
    this.isLoading = false;
    console.warn('check update problem', e);
  }

  onUpdateClicked($event)
  {
    this.isLoading = true;
    this.updateError = null;
    this.nodeService.update().subscribe(this.onUpdateSuccess.bind(this), this.onUpdateError.bind(this));
  }

  onUpdateSuccess(updated: boolean)
  {
    this.isLoading = false;
    if (updated)
    {
      this.dialogRef.close({
        updated: true
      });
    }
    else
    {
      this.onUpdateError();
    }
  }

  onUpdateError()
  {
    this.updateError = "Could not install node update. Please, try again later";
    this.isLoading = false;
  }
}
