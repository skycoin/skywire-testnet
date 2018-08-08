import {Component, EventEmitter, Input, OnInit, Output, Type} from '@angular/core';
import {MatTableDataSource} from "@angular/material";

@Component({
  selector: 'app-datatable',
  templateUrl: './datatable.component.html',
  styleUrls: ['./datatable.component.css']
})
export class DatatableComponent implements OnInit
{
  private dataSource = new MatTableDataSource<string>();

  @Input() data: string[];
  @Output() onSave: EventEmitter<string[]>;

  // Header
  @Input() displayedColumns: string[];
  @Input() headerTitle: string;

  // Remove row section
  @Input() removeRowTooltipText: string;

  // Editabe rows
  @Input() getEditableRowData: any;
  @Input() getEditableRowComponentClass: Type<any>;

  // Add input section
  @Input() getAddRowData: any;
  @Input() getAddRowComponentClass: Type<any>;
  @Input() addButtonTitle: string;
  @Input() onAddBtnClicked: () => void;
  @Input() valueToAdd: string;

  constructor() { }

  ngOnInit()
  {
    this.updateValues(this.data || []);
  }

  private updateValues(list: string[])
  {
    this.dataSource.data = list;
    this.onSave.emit(list);
  }
}
