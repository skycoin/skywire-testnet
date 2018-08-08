import {Component, Input, OnInit} from '@angular/core';

@Component({
  selector: 'app-editable-discovery-address',
  templateUrl: './editable-discovery-address.component.html',
  styleUrls: ['./editable-discovery-address.component.css']
})
export class EditableDiscoveryAddressComponent implements OnInit
{
  @Input() value: string;
  private editMode: boolean = false;

  constructor() { }

  ngOnInit() {
  }

  set data({ value }: {value: string})
  {
    this.value = value;
  }

  onValueClicked($event)
  {
    this.editMode = !this.editMode;
  }
}
