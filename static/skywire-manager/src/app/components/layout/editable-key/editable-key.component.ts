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

  constructor() {}

  ngOnInit() {}

  onAppKeyChanged({value, valid} : KeyInputEvent)
  {
    if (valid)
    {
      this.value = value;
    }
  }

  onKeyClicked()
  {
    this.toggleEditMode();
  }

  onAppKeyBlurred()
  {
    this.toggleEditMode();
    this.onValueEdited.emit(this.value);
  }

  private toggleEditMode()
  {
    this.editMode = !this.editMode;
  }

  set data({autofocus, value, subscriber}: {autofocus: boolean, value: string, subscriber: (next: string) => void})
  {
    this.autofocus = autofocus;
    this.value = value;
    this.onValueEdited.subscribe(subscriber);
  }
}
