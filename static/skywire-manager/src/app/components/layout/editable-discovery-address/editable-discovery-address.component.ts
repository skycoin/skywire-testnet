import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {DiscoveryAddress} from "../../../app.datatypes";

@Component({
  selector: 'app-editable-discovery-address',
  templateUrl: './editable-discovery-address.component.html',
  styleUrls: ['./editable-discovery-address.component.css']
})
export class EditableDiscoveryAddressComponent implements OnInit
{
  @Input() value: DiscoveryAddress;
  @Input() autofocus: boolean;
  @Output() onValueEdited = new EventEmitter<DiscoveryAddress>();
  editMode: boolean = false;

  constructor() {}

  ngOnInit() {}

  onValueChanged({valid, value}: {valid: boolean, value: DiscoveryAddress})
  {
    if (valid)
    {
      this.value = value;
    }
  }

  set data({autofocus, value, subscriber}: {autofocus: boolean, value: DiscoveryAddress, subscriber: (next: DiscoveryAddress) => void})
  {
    this.autofocus = autofocus;
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
