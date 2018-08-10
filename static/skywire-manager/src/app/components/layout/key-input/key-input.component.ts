import {AfterViewInit, Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {publicKeyValidator} from "../../../forms/validators";
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
  styleUrls: ['./key-input.component.scss'],
  host: {class: 'key-input-container'}
})
export class KeyInputComponent implements OnInit, AfterViewInit
{
  @Output() onKeyChange = new EventEmitter<KeyInputEvent>();
  @Input() value: string;
  @Input() required: boolean;
  @Input() placeholder: string;
  @Input() autofocus: boolean = false;
  validator: FormControl;

  constructor()
  {}

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
    this.value = $evt.target.value;
    this.onKeyChange.emit({
      value: this.value,
      valid: this.validator.valid
    });
  }

  clear()
  {
    this.value = "";
  }

  ngOnInit()
  {
    this.validator = new FormControl('', [publicKeyValidator(this.required)]);
  }

  set data({required, placeholder, subscriber, clearInputEmitter}: {required: boolean, placeholder: string, subscriber: ({value, valid}: KeyInputEvent) => void, clearInputEmitter: EventEmitter<void>})
  {
    this.required = required;
    this.placeholder = placeholder;
    this.onKeyChange.subscribe(subscriber);
    clearInputEmitter.subscribe(this.clear.bind(this));
  }
}
