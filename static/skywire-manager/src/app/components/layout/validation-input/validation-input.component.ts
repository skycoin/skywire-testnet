import {AfterViewInit, Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {MatInput} from "@angular/material";
import {FormControl} from "@angular/forms";

export interface InputState
{
  valid: boolean;
  value: string;
}

@Component({
  selector: 'app-validation-input',
  templateUrl: './validation-input.component.html',
  styleUrls: ['./validation-input.component.scss'],
  host: {class: 'validation-input-container'}
})
export class ValidationInputComponent implements OnInit, AfterViewInit
{
  constructor() { }

  @Output() valueChanged = new EventEmitter<InputState>();
  @Output() onBlur = new EventEmitter();
  @ViewChild(MatInput) inputElement: MatInput;
  @Input() value: string;
  @Input() required: boolean;
  @Input() placeHolder: string;
  @Input() hint: string;
  @Input() autofocus: boolean;
  @Input() validator: FormControl;
  @Input() getErrorMessage: () => string;

  ngOnInit() {}

  ngAfterViewInit()
  {
    if (this.autofocus)
    {
      this.inputElement.focus();
    }
  }

  onInput($evt)
  {
    this.value = $evt.target.value;
    this.valueChanged.emit({
      value: this.value,
      valid: this.validator.valid
    });
  }
}
