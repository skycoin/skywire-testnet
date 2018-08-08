import {Component, EventEmitter, Input, OnInit, Output, Type, ViewChild} from '@angular/core';
import {MatTableDataSource} from "@angular/material";
import {KeyInputComponent} from "../key-input/key-input.component";

@Component({
  selector: 'app-datatable',
  templateUrl: './datatable.component.html',
  styleUrls: ['./datatable.component.css']
})
export class DatatableComponent implements OnInit
{
  dataSource = new MatTableDataSource<string>();

  @Input() data: string[];
  @Output() onSave: EventEmitter<string[]>;

  // Header
  displayedColumns = [ 'index', 'key', 'remove' ];

  // Editabe rows
  @Input() getEditableRowData: (index: number, currentValue: string, callback: ()=>any) => any;
  @Input() getEditableRowComponentClass: () => Type<any>;

  // Add input section
  @Input() getAddRowData: (any) => any;
  @Input() getAddRowComponentClass: () => Type<any>;

  @Input() meta: {
    headerTitle: string,
    removeRowTooltipText: string,
    addButtonTitle: string
  };

  valueToAdd: string;

  constructor() { }

  ngOnInit()
  {
    this.updateValues(this.data || []);
  }

  private updateValues(list: string[])
  {
    this.dataSource.data = list;
    if (this.onSave)
    {
      this.onSave.emit(list);
    }
  }

  onAddValueChanged({value, valid}): void
  {
    if (valid)
    {
      this.valueToAdd = value;
    }
  }

  onAddBtnClicked()
  {
    this.data.push(this.valueToAdd);
    this.updateValues(this.data);

    this.valueToAdd = null;
  }

  onRemoveBtnClicked(position)
  {
    this.data.splice(position, 1);
    this.updateValues(this.data);
  }

  onKeyAtPositionChanged(position: number, keyValue: string)
  {
    let dataCopy = this.data;
    dataCopy[position] = keyValue;
    this.updateValues(dataCopy);
  }
}
