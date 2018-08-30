import {
  AfterViewInit,
  Component,
  EventEmitter,
  HostBinding,
  Input,
  OnChanges,
  OnInit,
  Output, SimpleChanges,
  ViewChild
} from '@angular/core';
import {publicKeyValidator} from '../../../forms/validators';
import {FormControl} from '@angular/forms';
import {MatInput} from '@angular/material';
import {Observable} from "rxjs";

export interface KeyInputEvent {
  value: string;
  valid: boolean;
}

@Component({
  selector: 'app-key-input',
  templateUrl: './key-input.component.html',
  styleUrls: ['./key-input.component.scss'],
})
export class KeyInputComponent implements OnInit, AfterViewInit, OnChanges {
  @HostBinding('attr.class') hostClass = 'key-input-container';
  @Output() keyChange = new EventEmitter<KeyInputEvent>();
  @Input() value = '';
  @Input() required: boolean;
  @Input() placeholder: string;
  @Input() autofocus = false;
  validator: FormControl;

  @ViewChild(MatInput) keyInput: MatInput;

  ngAfterViewInit() {
    if (this.autofocus) {
      this.keyInput.focus();
    }
  }

  onInput($evt) {
    this.value = $evt.target.value;
    this.emitState();
  }

  clear() {
    this.value = '';
  }

  ngOnInit() {
    this.createFormControl();
  }

  set data(data: Data) {
    this.required = data.required;
    this.placeholder = data.placeholder;
    this.keyChange.subscribe(data.subscriber);
    data.clearInputEmitter.subscribe(this.clear.bind(this));
  }

  ngOnChanges(changes: SimpleChanges): void {
    //console.log(`keyinput onchanges ${JSON.stringify(changes)}`);
    this.createFormControl();

    // setTimeout to avoid "ExpressionChangedAfterItHasBeenCheckedError" error...
    setTimeout(() => this.emitState(), 0);
  }

  createFormControl()
  {
    this.validator = new FormControl(this.value, [publicKeyValidator(this.required)]);
  }

  private emitState() {
    this.keyChange.emit({
      value: this.value,
      valid: this.validator.valid
    });
  }
}

interface Data {
  required: boolean;
  placeholder: string;
  subscriber: ({value, valid}: KeyInputEvent) => void;
  clearInputEmitter: EventEmitter<void>;
}
