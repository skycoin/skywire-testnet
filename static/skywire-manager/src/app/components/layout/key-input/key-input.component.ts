import {AfterViewInit, Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import PublicKeyValidator from "../../../forms/publicKeyValidator";
import {FormControl} from "@angular/forms";
import {MatInput} from "@angular/material";

export interface KeyInputEvent
{
  value: string;
  valid: boolean;
}

@Component({
  selector: 'app-key-input',
  templateUrl: './key-input.component.html',
  styleUrls: ['./key-input.component.css'],
  host: {class: 'key-input-container'}
})
export class KeyInputComponent implements OnInit, AfterViewInit
{
  @Output() onKeyChange = new EventEmitter<KeyInputEvent>();
  @Output() blur = new EventEmitter<void>();
  @Input() value: string;
  @Input() required: boolean = true;
  @Input() placeholder: string;
  validator: FormControl;

  constructor() {}

  @ViewChild(MatInput) keyInput: MatInput;

  ngAfterViewInit()
  {
    this.keyInput.focus();
  }

  onInput($evt)
  {
    console.log($evt.target.value);
    this.value = $evt.target.value;
    this.onKeyChange.emit({
      value: this.value,
      valid: this.validator.valid
    });
  }

  onBlur($evt)
  {
    this.blur.emit();
  }

  clear()
  {
    this.keyInput.value = null;
  }

  ngOnInit()
  {
    this.validator = new FormControl('', [PublicKeyValidator(this.required)]);
  }

}
