import {Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {FormControl, FormGroupDirective, NgForm, Validators} from '@angular/forms';
import {ErrorStateMatcher} from '@angular/material/core';
import {MatInput} from "@angular/material";

@Component({
  selector: 'app-number-input-min-value',
  templateUrl: './number-input-min-value.component.html',
  styleUrls: ['./number-input-min-value.component.css']
})
export class NumberInputMinValueComponent implements OnInit {

  @Input() minVal = 0;
  @Output() inputCorrect = new EventEmitter();
  @ViewChild(MatInput) inputElement: MatInput;
  @Input() value: number;
  @Input() fieldName: string;

  matcher = new MyErrorStateMatcher();
  private minError: string;
  private requiredError: string;
  private refreshSecondsFormControl: FormControl;

  constructor() { }

  ngOnInit()
  {
    this.minError = `Enter a number greater than ${this.minVal}`;
    this.requiredError = `${this.fieldName} can't be empty`;

    this.refreshSecondsFormControl = new FormControl('', [
      Validators.required,
      Validators.min(this.minVal),
    ]);
  }

  onInput($evt)
  {
    if (this.refreshSecondsFormControl.valid)
    {
      console.log($evt.target.value);
      this.value = $evt.target.value;
      this.inputCorrect.emit(this.value);
    }
  }
}

export class MyErrorStateMatcher implements ErrorStateMatcher
{
  isErrorState(control: FormControl | null, form: FormGroupDirective | NgForm | null): boolean {
    const isSubmitted = form && form.submitted;
    return (control && control.invalid && (control.dirty || control.touched || isSubmitted));
  }
}
