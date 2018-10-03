import {Component, EventEmitter, Input, OnInit, Output, Type} from '@angular/core';
import {MatTableDataSource} from '@angular/material';

@Component({
  selector: 'app-datatable',
  templateUrl: './datatable.component.html',
  styleUrls: ['./datatable.component.css']
})
export class DatatableComponent implements OnInit {
  dataSource = new MatTableDataSource<string>();

  @Input() data: string[];
  @Output() save = new EventEmitter<string[]>();

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

  ngOnInit() {
    this.updateValues(this.data || [], false);
  }

  private updateValues(list: string[], save: boolean = true) {
    this.dataSource.data = list.concat([]);
    if (save && this.save) {
      this.save.emit(list.concat([]));
    }
  }

  onAddValueChanged({value, valid}): void {
    this.valueToAdd = valid ? value : null;
  }

  onAddBtnClicked() {
    this.data.push(this.valueToAdd);
    this.updateValues(this.data);

    this.valueToAdd = null;
    this.clearInputEmitter.emit();
  }

  onRemoveBtnClicked(position) {
    this.data.splice(position, 1);
    this.updateValues(this.data);
  }

  onValueAtPositionChanged(position: number, value: any) {
    const dataCopy = this.data;
    dataCopy[position] = value;
    this.updateValues(dataCopy);
  }

  _getAddRowData() {
    const data = this.getAddRowData();
    data.subscriber = this.onAddValueChanged.bind(this);
    data.clearInputEmitter = this.clearInputEmitter;
    return data;
  }

  _getEditableRowData(position: number, currentValue: string) {
    const data = this.getEditableRowData(position, currentValue);
    data.subscriber = this.onValueAtPositionChanged.bind(this, position);
    data.required = true;
    return data;
  }
}

export interface DatatableProvider<T> {
  getEditableRowComponentClass: () => Type<any>;
  getAddRowComponentClass: () => Type<any>;
  getAddRowData: () => any;
  getEditableRowData: (index: number, currentValue: T) => any;
  save: (values: T[]) => void;
}
