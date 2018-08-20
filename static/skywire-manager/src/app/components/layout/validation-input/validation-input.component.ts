import { AfterViewInit, Component, EventEmitter, HostBinding, Input, Output, ViewChild } from '@angular/core';
import {MatInput} from '@angular/material';
import {FormControl} from '@angular/forms';

export interface InputState {
  valid: boolean;
  value: string;
}

@Component({
  selector: 'app-validation-input',
  templateUrl: './validation-input.component.html',
  styleUrls: ['./validation-input.component.scss'],
})
export class ValidationInputComponent implements AfterViewInit {
  @HostBinding('attr.class') hostClass = 'validation-input-container';
  @Output() valueChanged = new EventEmitter<InputState>();
  @ViewChild(MatInput) inputElement: MatInput;
  @Input() value: string;
  @Input() required: boolean;
  @Input() placeHolder: string;
  @Input() hint: string;
  @Input() autofocus: boolean;
  @Input() validator: FormControl;
  @Input() getErrorMessage: () => string;

  ngAfterViewInit() {
    if (this.autofocus) {
      this.inputElement.focus();
    }
  }

  onInput($evt) {
    this.value = $evt.target.value;
    this.valueChanged.emit({
      value: this.value,
      valid: this.validator.valid
    });
  }
}
