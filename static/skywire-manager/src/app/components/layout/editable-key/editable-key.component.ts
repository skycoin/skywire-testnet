import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {KeyInputEvent} from "../key-input/key-input.component";

@Component({
  selector: 'app-editable-key',
  templateUrl: './editable-key.component.html',
  styleUrls: ['./editable-key.component.scss'],
  host: {class: 'editable-key-container'}
})
export class EditableKeyComponent implements OnInit {
  @Input() value: string;
  @Input() autofocus: boolean = false;
  @Output() onValueEdited = new EventEmitter<string>();
  editMode: boolean = false;
  private valid: boolean = true;

  constructor() {}

  ngOnInit() {}

  onAppKeyChanged({value, valid} : KeyInputEvent)
  {
    this.valid = valid;
    if (valid)
    {
      this.value = value;
    }
  }

  private toggleEditMode()
  {
    this.editMode = !this.editMode;
    this.triggerValueChanged();
  }

  private triggerValueChanged()
  {
    if (!this.editMode && this.valid)
    {
      this.onValueEdited.emit(this.value);
    }
  }

  set data({autofocus, value, subscriber}: {autofocus: boolean, value: string, subscriber: (next: string) => void})
  {
    this.autofocus = autofocus;
    this.value = value;
    this.onValueEdited.subscribe(subscriber);
  }
}
