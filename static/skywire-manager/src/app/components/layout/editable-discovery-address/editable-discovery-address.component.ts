import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {DiscoveryAddress} from "../discovery-address-input/discovery-address-input.component";

@Component({
  selector: 'app-editable-discovery-address',
  templateUrl: './editable-discovery-address.component.html',
  styleUrls: ['./editable-discovery-address.component.css']
})
export class EditableDiscoveryAddressComponent implements OnInit
{
  @Input() value: DiscoveryAddress;
  @Output() onValueEdited = new EventEmitter<DiscoveryAddress>();
  editMode: boolean = false;

  constructor() {}

  ngOnInit() {}

  onValueChanged(value: DiscoveryAddress)
  {
    this.value = value;
  }

  set data({value, subscriber}: {value: DiscoveryAddress, subscriber: (next: DiscoveryAddress) => void})
  {
    this.value = value;
    this.onValueEdited.subscribe(subscriber);
  }

  onValueClicked()
  {
    this.toggleEditMode();
  }

  onDiscoveryAddressBlurred()
  {
    this.toggleEditMode();
    this.onValueEdited.emit(this.value);
  }

  private toggleEditMode()
  {
    this.editMode = !this.editMode;
  }
}
