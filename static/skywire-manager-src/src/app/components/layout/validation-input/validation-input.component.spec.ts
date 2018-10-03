import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { ValidationInputComponent } from './validation-input.component';

describe('ValidationInputComponent', () => {
  let component: ValidationInputComponent;
  let fixture: ComponentFixture<ValidationInputComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ ValidationInputComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ValidationInputComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
