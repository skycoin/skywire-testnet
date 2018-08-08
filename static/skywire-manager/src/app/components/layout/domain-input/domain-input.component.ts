import {Component, Input} from '@angular/core';
import {FormControl, Validators} from "@angular/forms";

@Component({
  selector: 'app-domain-input',
  templateUrl: './domain-input.component.html',
  styleUrls: ['./domain-input.component.css']
})
export class DomainInputComponent
{
  @Input() autofocus: boolean;
  @Input() value: string;
  editMode: boolean = false;

  validator = new FormControl('', [
    Validators.required
  ]);

  getErrorMessage() {
    return this.validator.hasError('required') ? 'Domain is required' : '';
  }

  ngOnInit()
  {
  }
}
