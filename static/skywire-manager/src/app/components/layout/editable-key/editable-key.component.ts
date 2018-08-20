import { Component, EventEmitter, HostBinding, Input, Output } from '@angular/core';
import {KeyInputEvent} from '../key-input/key-input.component';

@Component({
  selector: 'app-editable-key',
  templateUrl: './editable-key.component.html',
  styleUrls: ['./editable-key.component.scss'],
})
export class EditableKeyComponent {
  @HostBinding('attr.class') hostClass = 'editable-key-container';
  @Input() value: string;
  @Input() autofocus = false;
  @Input() required = false;
  @Output() valueEdited = new EventEmitter<string>();
  editMode = false;
  private valid = true;

  onAppKeyChanged({value, valid}: KeyInputEvent) {
    this.valid = valid;
    if (valid) {
      this.value = value;
    }
  }

  toggleEditMode() {
    this.editMode = !this.editMode;
    this.triggerValueChanged();
  }

  private triggerValueChanged() {
    if (!this.editMode && this.valid) {
      this.valueEdited.emit(this.value);
    }
  }

  set data(data: Data) {
    this.required = data.required;
    this.autofocus = data.autofocus;
    this.value = data.value;
    this.valueEdited.subscribe(data.subscriber);
  }
}

interface Data {
  required: boolean;
  autofocus: boolean;
  value: string;
  subscriber: (next: string) => void;
}
