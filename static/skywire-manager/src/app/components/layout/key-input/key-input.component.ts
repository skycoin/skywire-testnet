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
  @Input() autofocus: boolean = false;
  validator: FormControl;

  constructor()
  {
    console.log(`${this.autofocus}`);
  }

  @ViewChild(MatInput) keyInput: MatInput;

  ngAfterViewInit()
  {
    if (this.autofocus)
    {
      this.keyInput.focus();
    }
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
    this.value = "";
  }

  ngOnInit()
  {
    console.log(`2 - ${this.autofocus}`);
    this.validator = new FormControl('', [PublicKeyValidator(this.required)]);
  }

  set data({required, placeholder, onKeyChangeSubscriber}: {required: boolean, placeholder: string, onKeyChangeSubscriber: ({value, valid}: KeyInputEvent) => void})
  {
    this.required = required;
    this.placeholder = placeholder;
    this.onKeyChange.subscribe(onKeyChangeSubscriber);
  }
}
