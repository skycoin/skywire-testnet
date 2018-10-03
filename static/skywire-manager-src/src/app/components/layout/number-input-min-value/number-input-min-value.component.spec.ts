import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { NumberInputMinValueComponent } from './number-input-min-value.component';

describe('NumberInputMinValueComponent', () => {
  let component: NumberInputMinValueComponent;
  let fixture: ComponentFixture<NumberInputMinValueComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ NumberInputMinValueComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(NumberInputMinValueComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
