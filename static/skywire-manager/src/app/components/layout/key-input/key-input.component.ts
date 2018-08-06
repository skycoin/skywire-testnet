import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import PublicKeyValidator from "../../../forms/publicKeyValidator";
import {FormControl} from "@angular/forms";

export interface KeyInputEvent
{
  value: string;
  valid: boolean;
}

@Component({
  selector: 'app-key-input',
  templateUrl: './key-input.component.html',
  styleUrls: ['./key-input.component.css']
})
export class KeyInputComponent implements OnInit
{
  @Output() inputCorrect = new EventEmitter<KeyInputEvent>();
  @Input() value: string;
  @Input() placeholder: string;
  private validator = new FormControl('', [PublicKeyValidator]);

  constructor() {}

  onInput($evt)
  {
    console.log($evt.target.value);
    this.value = $evt.target.value;
    this.inputCorrect.emit({
      value: this.value,
      valid: this.validator.valid
    });
  }

  ngOnInit()
  {
  }

}
