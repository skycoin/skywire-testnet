import { Component, EventEmitter, HostBinding, Input, OnInit, Output } from '@angular/core';
import {FormControl, Validators} from '@angular/forms';
import {InputState} from '../validation-input/validation-input.component';
import {domainValidator} from '../../../forms/validators';
import {TranslateService} from "@ngx-translate/core";

@Component({
  selector: 'app-domain-input',
  templateUrl: './domain-input.component.html',
  styleUrls: ['./domain-input.component.css'],
})
export class DomainInputComponent implements OnInit {
  @HostBinding('attr.class') hostClass = 'domain-input-container';
  @Input() autofocus: boolean;
  @Input() value: string;
  @Input() required = false;
  @Output() domainChange = new EventEmitter<InputState>();
  validator: FormControl;

  constructor(
    private translate: TranslateService
  ){}

  getErrorMessage() {
    let key = this.validator.hasError('required') ?
      'inputs.errors.domain-required'
      :
      'inputs.errors.domain-format';

    return this.translate.instant(key);
  }

  ngOnInit() {
    const validatorsList = [domainValidator];
    if (this.required) {
      validatorsList.push(Validators.required);
    }
    this.validator = new FormControl('', validatorsList);
  }
}
