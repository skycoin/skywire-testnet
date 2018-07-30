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
  @Output() input = new EventEmitter();
  @ViewChild(MatInput) inputElement: MatInput;
  private value: number = this.minVal;
  private error: string = "lalala";

  refreshSecondsFormControl = new FormControl('', [
    Validators.required,
    Validators.min(1),
  ]);

  matcher = new MyErrorStateMatcher();

  constructor() { }

  ngOnInit()
  {
    console.log(this.minVal);
  }

  onInput($evt)
  {
    if (this.refreshSecondsFormControl.valid)
    {
      console.log($evt.target.value);
    }
    /*let val = $evt.target.value;
    if (val && val >= this.minVal)
    {
      this.value = val;
      this.input.emit(this.value);
    }
    else
    {
      this.error = "Values must be grater than 0";
    }*/
  }
}

export class MyErrorStateMatcher implements ErrorStateMatcher
{
  isErrorState(control: FormControl | null, form: FormGroupDirective | NgForm | null): boolean {
    const isSubmitted = form && form.submitted;
    return (control && control.invalid && (control.dirty || control.touched || isSubmitted));
  }
}
