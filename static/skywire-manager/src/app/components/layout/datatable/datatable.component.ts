import {Component, EventEmitter, Input, OnInit, Output, Type} from '@angular/core';
import {MatTableDataSource} from "@angular/material";

@Component({
  selector: 'app-datatable',
  templateUrl: './datatable.component.html',
  styleUrls: ['./datatable.component.css']
})
export class DatatableComponent implements OnInit
{
  dataSource = new MatTableDataSource<string>();

  @Input() data: string[];
  @Output() onSave = new EventEmitter<string[]>();

  // Header
  displayedColumns = [ 'index', 'key', 'remove' ];

  // Editabe rows
  @Input() getEditableRowData: (index: number, currentValue: string) => any;
  @Input() getEditableRowComponentClass: () => Type<any>;

  // Add input section
  @Input() getAddRowData: () => any;
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

  onValueAtPositionChanged(position: number, value: string)
  {
    let dataCopy = this.data;
    dataCopy[position] = value;
    this.updateValues(dataCopy);
  }

  _getAddRowData()
  {
    let data = this.getAddRowData();
    data.subscriber = this.onAddValueChanged.bind(this);
    return data;
  }

  _getEditableRowData(position: number, currentValue: string)
  {
    let data = this.getEditableRowData(position, currentValue);
    data.subscriber = this.onValueAtPositionChanged.bind(this);
    return data;
  }
}
