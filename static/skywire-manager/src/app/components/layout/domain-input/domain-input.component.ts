import {Component, EventEmitter, Input, Output} from '@angular/core';
import {FormControl, Validators} from "@angular/forms";

@Component({
  selector: 'app-domain-input',
  templateUrl: './domain-input.component.html',
  styleUrls: ['./domain-input.component.css'],
  host: {class: 'domain-input-container'}
})
export class DomainInputComponent
{
  @Input() autofocus: boolean;
  @Input() value: string;
  @Input() required: boolean;
  @Output() onDomainChange = new EventEmitter<string>();
  editMode: boolean = false;
  validator: FormControl;

  constructor()
  {

  }

  getErrorMessage() {
    return this.validator.hasError('required') ? 'Domain is required' : '';
  }

  ngOnInit()
  {
    let validatorsList = [];
    if (this.required)
    {
      validatorsList.push(Validators.required)
    }
    this.validator = new FormControl('', validatorsList);
  }
}
