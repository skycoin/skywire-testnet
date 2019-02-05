import {Component, EventEmitter, Input, Output} from '@angular/core';
import {DiscoveryAddress} from '../../../app.datatypes';

@Component({
  selector: 'app-editable-discovery-address',
  templateUrl: './editable-discovery-address.component.html',
  styleUrls: ['./editable-discovery-address.component.css']
})
export class EditableDiscoveryAddressComponent {
  @Input() value: DiscoveryAddress;
  @Input() required = false;
  @Input() autofocus: boolean;
  @Input() discovered: boolean;
  @Output() valueEdited = new EventEmitter<DiscoveryAddress>();
  editMode = false;
  valid = true;

  onValueChanged({valid, value}: {valid: boolean, value: DiscoveryAddress}) {
    this.valid = valid;
    this.discovered = undefined;

    if (valid) {
      this.value = value;
    }
  }

  set data(data: Data) {
    this.required = data.required;
    this.autofocus = data.autofocus;
    this.discovered = data.discovered;
    this.value = data.value;
    this.valueEdited.subscribe(data.subscriber);
  }

  onValueClicked() {
    this.toggleEditMode();
  }

  toggleEditMode() {
    this.editMode = !this.editMode;
    this.triggerValueChanged();
  }

  private triggerValueChanged() {
    if (this.valid && !this.editMode) {
      this.valueEdited.emit(this.value);
    }
  }
}

interface Data {
  required: boolean;
  autofocus: boolean;
  discovered: boolean;
  value: DiscoveryAddress;
  subscriber: (next: DiscoveryAddress) => void;
}
