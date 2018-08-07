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

  constructor() { }

  ngOnInit()
  {
    console.log('init');
  }

  onAppKeyChanged({value, valid} : KeyInputEvent)
  {
    if (valid)
    {
      this.value = value;
    }
  }

  onKeyClicked($event)
  {
    this.toggleEditMode();
  }

  onAppKeyBlurred($event)
  {
    this.toggleEditMode();
    this.onValueEdited.emit(this.value);
  }

  private toggleEditMode()
  {
    this.editMode = !this.editMode;
  }
}
