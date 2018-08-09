import {AfterViewInit, Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {MatInput} from "@angular/material";
import {FormControl} from "@angular/forms";

@Component({
  selector: 'app-validation-input',
  templateUrl: './validation-input.component.html',
  styleUrls: ['./validation-input.component.scss'],
  host: {class: 'validation-input-container'}
})
export class ValidationInputComponent implements OnInit, AfterViewInit
{
  constructor() { }

  @Output() inputCorrect = new EventEmitter();
  @Output() onBlur = new EventEmitter();
  @ViewChild(MatInput) inputElement: MatInput;
  @Input() value: string;
  @Input() required: boolean;
  @Input() placeHolder: string;
  @Input() hint: string;
  @Input() autofocus: string;
  @Input() validator: FormControl;
  @Input() getErrorMessage: () => string;

  ngOnInit()
  {
    if (this.autofocus)
    {
      this.inputElement.focus();
    }
  }

  ngAfterViewInit()
  {
    if (this.autofocus)
    {
      this.keyInput.focus();
    }
  }

  onInput($evt)
  {
    if (this.validator.valid)
    {
      this.value = $evt.target.value;
      this.inputCorrect.emit(this.value);
    }
  }
}
