import {Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {MatInput} from "@angular/material";
import {FormControl, Validators} from "@angular/forms";

@Component({
  selector: 'app-validation-input',
  templateUrl: './validation-input.component.html',
  styleUrls: ['./validation-input.component.css']
})
export class ValidationInputComponent implements OnInit
{
  constructor() { }

  @Output() inputCorrect = new EventEmitter();
  @ViewChild(MatInput) inputElement: MatInput;
  @Input() value: number;
  @Input() placeHolder: string;
  @Input() hint: string;
  @Input() validator: FormControl;
  @Input() getErrorMessage: () => string;

  ngOnInit()
  {
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
