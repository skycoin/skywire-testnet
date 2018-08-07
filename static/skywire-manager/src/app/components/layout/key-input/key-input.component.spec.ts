import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { KeyInputComponent } from './key-input.component';

describe('KeyInputComponent', () => {
  let component: KeyInputComponent;
  let fixture: ComponentFixture<KeyInputComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ KeyInputComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(KeyInputComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
