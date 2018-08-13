import {Component, EventEmitter, Input, Output} from '@angular/core';
import {FormControl, Validators} from "@angular/forms";
import {InputState} from "../validation-input/validation-input.component";
import {domainValidator} from "../../../forms/validators";

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
  @Input() required: boolean = false;
  @Output() onDomainChange = new EventEmitter<InputState>();
  validator: FormControl;

  constructor()
  {

  }

  getErrorMessage()
  {
    return this.validator.hasError('required') ? 'Domain is required' : 'Format must be domain:port';
  }

  ngOnInit()
  {
    let validatorsList = [domainValidator];
    if (this.required)
    {
      validatorsList.push(Validators.required)
    }
    this.validator = new FormControl('', validatorsList);
  }
}
