import { AfterViewInit, Component, EventEmitter, HostBinding, Input, OnInit, Output, ViewChild } from '@angular/core';
import {publicKeyValidator} from '../../../forms/validators';
import {FormControl} from '@angular/forms';
import {MatInput} from '@angular/material';

export interface KeyInputEvent {
  value: string;
  valid: boolean;
}

@Component({
  selector: 'app-key-input',
  templateUrl: './key-input.component.html',
  styleUrls: ['./key-input.component.scss'],
})
export class KeyInputComponent implements OnInit, AfterViewInit {
  @HostBinding('attr.class') hostClass = 'key-input-container';
  @Output() keyChange = new EventEmitter<KeyInputEvent>();
  @Input() value: string = "";
  @Input() required: boolean;
  @Input() placeholder: string;
  @Input() autofocus = false;
  validator: FormControl;

  @ViewChild(MatInput) keyInput: MatInput;

  ngAfterViewInit() {
    if (this.autofocus) {
      this.keyInput.focus();
    }
  }

  onInput($evt) {
    this.value = $evt.target.value;
    this.keyChange.emit({
      value: this.value,
      valid: this.validator.valid
    });
  }

  clear() {
    this.value = '';
  }

  ngOnInit() {
    this.validator = new FormControl('', [publicKeyValidator(this.required)]);
  }

  set data(data: Data) {
    this.required = data.required;
    this.placeholder = data.placeholder;
    this.keyChange.subscribe(data.subscriber);
    data.clearInputEmitter.subscribe(this.clear.bind(this));
  }
}

interface Data {
  required: boolean;
  placeholder: string;
  subscriber: ({value, valid}: KeyInputEvent) => void;
  clearInputEmitter: EventEmitter<void>;
}
