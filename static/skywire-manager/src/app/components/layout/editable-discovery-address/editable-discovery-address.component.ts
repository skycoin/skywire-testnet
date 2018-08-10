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
  @Input() required: boolean = false;
  @Input() autofocus: boolean;
  @Output() onValueEdited = new EventEmitter<DiscoveryAddress>();
  editMode: boolean = false;
  private valid: boolean = true;

  constructor() {}

  ngOnInit() {}

  onValueChanged({valid, value}: {valid: boolean, value: DiscoveryAddress})
  {
    this.valid = valid;
    if (valid)
    {
      this.value = value;
    }
  }

  set data({required, autofocus, value, subscriber}: {required: boolean, autofocus: boolean, value: DiscoveryAddress, subscriber: (next: DiscoveryAddress) => void})
  {
    this.required = required;
    this.autofocus = autofocus;
    this.value = value;
    this.onValueEdited.subscribe(subscriber);
  }

  onValueClicked()
  {
    this.toggleEditMode();
  }

  private toggleEditMode()
  {
    this.editMode = !this.editMode;
    this.triggerValueChanged();
  }

  private triggerValueChanged()
  {
    if (this.valid && !this.editMode)
    {
      this.onValueEdited.emit(this.value);
    }
  }
}
