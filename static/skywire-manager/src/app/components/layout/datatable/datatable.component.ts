import {Component, EventEmitter, Input, OnChanges, OnInit, Output, Type} from '@angular/core';
import {MatTableDataSource} from "@angular/material";

@Component({
  selector: 'app-datatable',
  templateUrl: './datatable.component.html',
  styleUrls: ['./datatable.component.css']
})
export class DatatableComponent implements OnInit, OnChanges
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
  private clearInputEmitter = new EventEmitter();

  constructor() { }

  ngOnInit()
  {
    this.updateValues(this.data || []);
  }

  private updateValues(list: string[])
  {
    this.dataSource.data = list.concat([]);
    if (this.onSave)
    {
      this.onSave.emit(list.concat([]));
    }
  }

  onAddValueChanged({value, valid}): void
  {
    this.valueToAdd = valid ? value : null;
  }

  onAddBtnClicked()
  {
    this.data.push(this.valueToAdd);
    this.updateValues(this.data);

    this.valueToAdd = null;
    this.clearInputEmitter.emit();
  }

  onRemoveBtnClicked(position)
  {
    this.data.splice(position, 1);
    this.updateValues(this.data);
  }

  onEditBtnClicked(position)
  {

  }

  onValueAtPositionChanged(position: number, value: any)
  {
    let dataCopy = this.data;
    dataCopy[position] = value;
    this.updateValues(dataCopy);
  }

  _getAddRowData()
  {
    let data = this.getAddRowData();
    data.subscriber = this.onAddValueChanged.bind(this);
    data.clearInputEmitter = this.clearInputEmitter;
    return data;
  }

  _getEditableRowData(position: number, currentValue: string)
  {
    let data = this.getEditableRowData(position, currentValue);
    data.subscriber = this.onValueAtPositionChanged.bind(this, position);
    return data;
  }

  ngOnChanges(): void
  {
    //this.updateValues(this.data || []);
  }
}

export interface DatatableProvider<T>
{
  getEditableRowComponentClass: () => Type<any>;
  getAddRowComponentClass: () => Type<any>;
  getAddRowData: () => any;
  getEditableRowData: (index: number, currentValue: T) => any;
  save: (values: T[]) => void;
}
