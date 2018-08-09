import {Component, Input, OnInit} from '@angular/core';
import {DiscoveryAddress} from "../discovery-address-input/discovery-address-input.component";

@Component({
  selector: 'app-editable-discovery-address',
  templateUrl: './editable-discovery-address.component.html',
  styleUrls: ['./editable-discovery-address.component.css']
})
export class EditableDiscoveryAddressComponent implements OnInit
{
  @Input() value: DiscoveryAddress;
  editMode: boolean = false;

  constructor() { }

  ngOnInit()
  {

  }

  set data({value}: {value: DiscoveryAddress})
  {
    this.value = value;
  }

  onValueClicked($event)
  {
    this.editMode = !this.editMode;
  }
}
