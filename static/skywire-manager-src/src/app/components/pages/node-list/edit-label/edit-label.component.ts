import { Component, Inject, OnInit } from '@angular/core';
import { NodeService } from '../../../../services/node.service';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material';

@Component({
  selector: 'app-edit-label',
  templateUrl: './edit-label.component.html',
  styleUrls: ['./edit-label.component.scss']
})
export class EditLabelComponent implements OnInit {
  label: string;

  constructor(
    public dialogRef: MatDialogRef<EditLabelComponent>,
    @Inject(MAT_DIALOG_DATA) private data: any,
    private nodeService: NodeService,
  ) { }

  ngOnInit() {
    this.label = this.nodeService.getLabel(this.data.node);
  }

  save() {
    this.nodeService.setLabel(this.data.node, this.label);
    this.dialogRef.close();
  }
}
